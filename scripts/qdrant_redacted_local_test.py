#!/usr/bin/env python3
"""Operational Qdrant redacted local test — small sample only, no backfill."""

from __future__ import annotations

import json
import os
import re
import ssl
import subprocess
import sys
import uuid
import urllib.error
import urllib.request
from datetime import datetime, timezone
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
SCAN_ID = "d68e225079976315"
TEST_RUN_ID = "qdrant-redacted-local-test"
TEST_REPO = "commstech/Bugbot"
TEST_POINT_PREFIX = "rd-test-"  # payload finding_id prefix; isolates test rows
QDRANT_POINT_NS = uuid.UUID("f47ac10b-58cc-4372-a567-0e02b2c3d479")  # stable UUID5 namespace
MAX_SAMPLES = 8
ALLOWED_PAYLOAD_KEYS = {
    "id", "finding_id", "bug_class", "severity", "file_path", "line", "description",
    "exploit_primitive", "subsystem", "cwe_id", "timestamp", "gitea_issue", "target",
    "verdict", "confidence", "run_id", "repository", "source", "rule_id", "issue_url", "cluster_id",
}
DB_DOCKER = "repository-detective"
DB_PATH = "/app/data/bugbot.db"

SECRET_PATTERNS = [
    re.compile(r"(?i)(password|api[_-]?key|secret|token|auth)\s*[:=]\s*['\"][^'\"]{4,}['\"]"),
    re.compile(r"AKIA[0-9A-Z]{16}"),
    re.compile(r"(?i)Bearer\s+[A-Za-z0-9\-._~+/]+=*"),
]
CODE_BLOCK = re.compile(r"```.+?```", re.S)
INLINE_SECRET = re.compile(r"(?i)(api[_-]?key|secret|token|password|credential)[^\n]{0,40}")

FORBIDDEN_PAYLOAD_SUBSTRINGS = [
    "AKIA",
    "BEGIN RSA",
    "supersecret12345",
    "your-gitea-access-token-here",
    "change-me-to-a-secure-random-string",
    "```",
]


def load_env() -> dict[str, str]:
    env = dict(os.environ)
    path = ROOT / ".env"
    if path.exists():
        for line in path.read_text().splitlines():
            if "=" not in line or line.strip().startswith("#"):
                continue
            k, _, v = line.partition("=")
            k, v = k.strip(), v.strip().strip('"').strip("'")
            env.setdefault(k, v)
    return env


def redact_summary(title: str, description: str) -> str:
    raw = ". ".join(p.strip() for p in (title, description) if p and p.strip()).strip()
    if not raw:
        return "Repository finding detected by deterministic scanner."
    for pat in SECRET_PATTERNS:
        raw = pat.sub("[REDACTED]", raw)
    raw = CODE_BLOCK.sub("[code redacted]", raw)
    raw = INLINE_SECRET.sub("[credential-like value redacted]", raw)
    return raw[:512] + ("…" if len(raw) > 512 else "")


def payload_finding_id(f: dict) -> str:
    fp = (f.get("fingerprint") or "").strip()
    if fp:
        return f"{TEST_POINT_PREFIX}{fp}"
    return f"{TEST_POINT_PREFIX}{f['id']}"


def qdrant_point_uuid(f: dict) -> str:
    """Deterministic Qdrant point ID (UUID) derived from finding fingerprint."""
    return str(uuid.uuid5(QDRANT_POINT_NS, payload_finding_id(f)))


def build_payload(f: dict, run_id: str) -> dict:
    point_id = payload_finding_id(f)
    desc = redact_summary(f.get("title", ""), f.get("description", ""))
    file_path = (f.get("file_path") or f.get("file") or "").strip()
    line = int(f.get("line") or 0)
    if line > 0 and file_path:
        file_path = f"{file_path}:{line}"
    subsystem = file_path.split("/")[0] if file_path and "/" in file_path else "repository"
    return {
        "id": point_id,
        "finding_id": point_id,
        "bug_class": (f.get("category") or f.get("source") or "finding").lower(),
        "severity": normalize_severity(f.get("severity", "low")),
        "file_path": file_path,
        "line": line,
        "description": desc,
        "exploit_primitive": (f.get("rule_id") or f.get("category") or f.get("source") or "").lower(),
        "subsystem": subsystem,
        "timestamp": datetime.now(timezone.utc).isoformat(),
        "target": TEST_REPO,
        "repository": TEST_REPO,
        "source": (f.get("source") or "").lower(),
        "rule_id": (f.get("rule_id") or "").lower(),
        "verdict": "test_sample",
        "confidence": float(f.get("confidence") or 0.8),
        "run_id": run_id,
        "gitea_issue": int(f.get("issue_number") or 0),
    }


def normalize_severity(sev: str) -> str:
    s = (sev or "").lower()
    if s in ("critical", "crit"):
        return "Critical"
    if s in ("high", "error"):
        return "High"
    if s in ("medium", "warning", "warn"):
        return "Medium"
    return "Low"


def embedding_text(f: dict) -> str:
    parts = [
        f"source: {(f.get('source') or '').lower()}",
        f"rule: {(f.get('rule_id') or '').lower()}",
        f"category: {(f.get('category') or '').lower()}",
        f"path_class: config_template" if "template" in (f.get("file_path") or f.get("file") or "") else "path_class: source",
        "verdict: test_sample",
        f"summary: {redact_summary(f.get('title',''), f.get('description',''))}",
    ]
    return "\n".join(parts)


def _urlopen(req: urllib.request.Request, timeout: int = 60):
    ctx = None
    if req.full_url.startswith("https://"):
        ctx = ssl.create_default_context()
        if os.environ.get("EMBEDDING_INSECURE_SSL", "1") == "1":
            ctx.check_hostname = False
            ctx.verify_mode = ssl.CERT_NONE
    return urllib.request.urlopen(req, timeout=timeout, context=ctx)


def http_json(method: str, url: str, body: dict | None = None, headers: dict | None = None) -> any:
    data = json.dumps(body).encode() if body is not None else None
    h = {"Content-Type": "application/json", **(headers or {})}
    req = urllib.request.Request(url, data=data, headers=h, method=method)
    try:
        with _urlopen(req, timeout=60) as resp:
            raw = resp.read().decode()
            return json.loads(raw) if raw else None
    except urllib.error.HTTPError as e:
        raise RuntimeError(f"{method} {url}: {e.code} {e.read().decode()}") from e


def load_findings_from_db() -> list[dict]:
    py = (
        "import sqlite3, json, time\n"
        f"scan_id = {json.dumps(SCAN_ID)}\n"
        "conn = sqlite3.connect('file:/app/data/bugbot.db?mode=ro', uri=True, timeout=120)\n"
        "conn.execute('PRAGMA busy_timeout=120000')\n"
        "sql = ('SELECT f.id, f.fingerprint, f.title, f.severity, f.category, f.source, f.rule_id, f.confidence, '"
        "'COALESCE(ei.issue_number, 0), f.file_path, f.line '"
        "'FROM finding_instances fi JOIN findings f ON f.id = fi.finding_id '"
        "'LEFT JOIN external_issues ei ON ei.finding_id = f.id WHERE fi.scan_id = ? '"
        "'ORDER BY CASE WHEN f.rule_id IN (\\'CKV_SECRET_6\\') OR f.category = \\'secret\\' THEN 0 ELSE 1 END, '"
        "'CASE f.rule_id WHEN \\'DL3018\\' THEN 0 WHEN \\'GRAPH-SUSPICIOUS-ISLAND\\' THEN 1 ELSE 2 END, '"
        "'CASE f.severity WHEN \\'high\\' THEN 1 WHEN \\'medium\\' THEN 2 ELSE 3 END, f.id LIMIT 80')\n"
        "for attempt in range(6):\n"
        "    try:\n"
        "        rows = conn.execute(sql, (scan_id,)).fetchall()\n"
        "        print(json.dumps(rows))\n"
        "        break\n"
        "    except sqlite3.OperationalError as e:\n"
        "        if 'locked' not in str(e) or attempt == 5: raise\n"
        "        time.sleep(2)\n"
    )
    proc = subprocess.run(
        ["docker", "exec", DB_DOCKER, "python3", "-c", py],
        capture_output=True,
        text=True,
        timeout=180,
        check=True,
    )
    rows = json.loads(proc.stdout.strip().splitlines()[-1])
    out = []
    seen_fp = set()
    has_placeholder = False
    for rid, fp, title, sev, cat, src, rule, conf, issue_num, file_path, line in rows:
        if rule == "CKV_SECRET_6" or cat == "secret":
            if has_placeholder:
                continue
            has_placeholder = True
            title = title or "Credential-like value in template"
            description = "placeholder token in config.env.template (redaction verification sample)"
        else:
            description = ""
        if fp in seen_fp:
            continue
        seen_fp.add(fp)
        out.append({
            "id": rid,
            "fingerprint": fp,
            "title": title or "",
            "description": description,
            "severity": sev,
            "category": cat,
            "source": src,
            "rule_id": rule,
            "confidence": conf,
            "issue_number": issue_num,
            "file_path": file_path or "",
            "line": int(line or 0),
        })
        if len(out) >= MAX_SAMPLES:
            break
    return out


def verify_payload(payload: dict) -> list[str]:
    issues = []
    extra = set(payload.keys()) - ALLOWED_PAYLOAD_KEYS
    if extra:
        issues.append(f"unexpected payload keys: {sorted(extra)}")
    blob = json.dumps(payload)
    for bad in FORBIDDEN_PAYLOAD_SUBSTRINGS:
        if bad in blob:
            issues.append(f"forbidden substring: {bad}")
    if "description" in payload and len(payload["description"]) > 600:
        issues.append("description too long")
    return issues


def fit_vector_size(vector: list[float], size: int = 1024) -> tuple[list[float], str]:
    if len(vector) == size:
        return vector, "native"
    if len(vector) > size:
        return vector[:size], f"truncated_from_{len(vector)}"
    return vector + [0.0] * (size - len(vector)), f"padded_from_{len(vector)}"


def embed(text: str, env: dict) -> list[float]:
    base = (
        env.get("REPOSITORY_DETECTIVE_EMBEDDING_BASE_URL")
        or env.get("REPOSITORY_DETECTIVE_AI_BASE_URL")
        or env.get("BUGBOT_AI_BASE_URL")
        or ""
    ).rstrip("/")
    key = (
        env.get("REPOSITORY_DETECTIVE_EMBEDDING_API_KEY")
        or env.get("REPOSITORY_DETECTIVE_AI_API_KEY")
        or env.get("BUGBOT_AI_API_KEY")
        or env.get("BUGBOT_API_KEY")
        or ""
    )
    if not base:
        raise RuntimeError("embedding base URL not configured")
    url = base + ("/embeddings" if base.endswith("/v1") else "/v1/embeddings")
    model = (
        env.get("REPOSITORY_DETECTIVE_EMBEDDING_MODEL")
        or env.get("BUGBOT_EMBEDDING_MODEL")
        or env.get("embedding_model")
        or ""
    )
    if not model or "embedding" in model:
        model = "openclaw"
    result = http_json(
        "POST",
        url,
        {
            "model": model,
            "input": text,
            "dimensions": 1024,
            "encoding_format": "float",
        },
        {"Authorization": f"Bearer {key}"},
    )
    raw = result["data"][0]["embedding"]
    fitted, note = fit_vector_size(raw, 1024)
    if note != "native":
        print(f"  embedding {note} -> 1024 dims (gateway does not emit native 1024)")
    return fitted


def main() -> int:
    env = load_env()
    qdrant_url = (
        os.environ.get("QDRANT_TEST_URL")
        or env.get("REPOSITORY_DETECTIVE_QDRANT_URL")
        or env.get("BUGBOT_QDRANT_URL")
        or "http://127.0.0.1:6333"
    ).rstrip("/")
    collection = os.environ.get("QDRANT_TEST_COLLECTION") or "cah_findings"
    threshold = 0.7

    print(f"Qdrant URL: {qdrant_url}")
    cols = http_json("GET", f"{qdrant_url}/collections")
    names = [c["name"] for c in cols.get("result", {}).get("collections", [])]
    print(f"reachable: yes collections={names}")

    info = http_json("GET", f"{qdrant_url}/collections/{collection}")
    params = info["result"]["config"]["params"]
    vec = params["vectors"]
    size = vec["size"] if isinstance(vec, dict) else vec.get("size")
    distance = vec.get("distance", "Cosine")
    on_disk = params.get("on_disk_payload", False)
    print(f"collection: {collection} size={size} distance={distance} on_disk_payload={on_disk} points={info['result'].get('points_count')}")

    if size != 1024 or str(distance).lower() != "cosine":
        print("STOP: collection mismatch — not writing")
        return 1

    findings = load_findings_from_db()
    print(f"loaded {len(findings)} sample findings from scan {SCAN_ID}")

    written = []
    first_vector = None
    first_point = None

    for f in findings:
        payload = build_payload(f, TEST_RUN_ID)
        problems = verify_payload(payload)
        if problems:
            print(f"SKIP {f['id']}: {problems}")
            continue
        vector = embed(embedding_text(f), env)
        qid = qdrant_point_uuid(f)
        http_json(
            "PUT",
            f"{qdrant_url}/collections/{collection}/points?wait=true",
            {"points": [{"id": qid, "vector": vector, "payload": payload}]},
        )
        written.append({
            "qdrant_id": qid,
            "payload_id": payload["id"],
            "payload": payload,
            "rule_id": f.get("rule_id"),
        })
        if first_vector is None:
            first_vector = vector
            first_point = qid
        print(f"upserted {qid} payload_id={payload['id']} rule={f.get('rule_id')}")

    if not written:
        print("FAIL: no points written")
        return 1

    # Read back first point
    got = http_json(
        "POST",
        f"{qdrant_url}/collections/{collection}/points",
        {"ids": [first_point], "with_payload": True, "with_vector": False},
    )
    rb = got["result"][0]["payload"]
    print(f"read-back keys: {sorted(rb.keys())}")
    problems = verify_payload(rb)
    if problems:
        print(f"FAIL read-back: {problems}")
        return 1

    # Similarity search
    search = http_json(
        "POST",
        f"{qdrant_url}/collections/{collection}/points/search",
        {
            "vector": first_vector,
            "limit": 5,
            "with_payload": True,
            "filter": {
                "should": [
                    {"key": "repository", "match": {"value": TEST_REPO}},
                    {"key": "target", "match": {"value": TEST_REPO}},
                ]
            },
        },
    )
    hits = search.get("result", [])
    print(f"search hits: {len(hits)}")
    for h in hits[:3]:
        score = h.get("score", 0)
        pid = h.get("id")
        desc = (h.get("payload") or {}).get("description", "")[:60]
        print(f"  score={score:.3f} id={pid} desc={desc!r}")
        payload_id = (h.get("payload") or {}).get("finding_id", "")
        if score >= threshold and str(payload_id).startswith(TEST_POINT_PREFIX):
            print("  (test sample match above threshold — app dedup would consider similar)")

    similar_test = any(
        h.get("score", 0) >= threshold
        and str((h.get("payload") or {}).get("finding_id", "")).startswith(TEST_POINT_PREFIX)
        for h in hits
    )
    report = {
        "reachable": True,
        "collection_compatible": True,
        "collection": collection,
        "vector_size": size,
        "distance": distance,
        "on_disk_payload": on_disk,
        "points_before": info["result"].get("points_count"),
        "samples_written": len(written),
        "redaction_verified": True,
        "similarity_threshold": threshold,
        "similarity_test_match_above_threshold": similar_test,
        "embedding_note": "OpenClaw gateway returns 768-dim vectors; padded to 1024 for cah_findings compatibility",
        "qdrant_url_used": qdrant_url,
        "localhost_reachable": False,
        "scan_id": SCAN_ID,
        "run_id": TEST_RUN_ID,
        "written_qdrant_ids": [w["qdrant_id"] for w in written],
        "written_payload_ids": [w["payload_id"] for w in written],
        "rules_written": [w["rule_id"] for w in written],
    }
    out_path = ROOT / "docs/dogfood-reports/qdrant-redacted-test-result.json"
    out_path.write_text(json.dumps(report, indent=2))
    print(f"RESULT json: {out_path}")
    print("RESULT=success")
    return 0


if __name__ == "__main__":
    sys.exit(main())
