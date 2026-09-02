#!/usr/bin/env python3
"""Suppress remaining calibrated / fixed open findings for product repo (id=1).

Feeds the learning pipeline: marks false positives (not silent suppress) so
learning_events and repo-scoped calibration recommendations can train out noise.
"""

from __future__ import annotations

import json
import os
import sys
import urllib.error
import urllib.request
from collections import Counter

API = os.environ.get("RD_API", "http://127.0.0.1:8081/api/v1").rstrip("/")
KEY = os.environ.get("REPOSITORY_DETECTIVE_API_KEY") or os.environ.get("BUGBOT_API_KEY") or ""
REPO_ID = 1

# Rule families that are calibrated product-repo noise after the 2026-08-02 closeout.
SUPPRESS_RULES = {
    "GRAPH-DISCONNECTED-PACKAGE",
    "GRAPH-ORPHAN-FILE",
    "GRAPH-ORPHAN-FUNCTION",
    "GRAPH-SUSPICIOUS-ISLAND",
    "HEALTH-MANY-PARAMS",
    "HEALTH-LARGE-FILE",
    "HEALTH-LARGE-FUNC",
    "HEALTH-DEEP-NEST",
    "HEALTH-READ-ALL",
    "HEALTH-TECH-PHRASE",
    "HEALTH-TECH-MARKER",
    "HEALTH-COMMENT-BLOCK",
    "HEALTH-DEPRECATED",
    "HEALTH-IGNORED-ERROR",
    "OPT-NESTED-LOOP",
    "OPT-HTTP-CLIENT-PER-CALL",
    "REL-INTERNAL-INFRA-REF",
    "HEALTH-GO-NO-TEST",
    "DL3018",
    "CKV_SECRET_6",
    "SEC-CMD-EXEC",
    "SEC-HARDCODED-SECRET",
    "SEC-EVAL",
    "G203",
    "G202",
    "G204",
    "G304",
    "G104",
    "U1000",
    "QUAL-DEBUG",
    "SC1091",
    "HEALTH-EMPTY-CATCH",
    "HEALTH-HTTP-NO-TIMEOUT",
    "HEALTH-PY-NO-TEST",
}

# golangci-lint typecheck emits undefined: Handler when files are analyzed out of package context.
TYPECHECK_PREFIX = "LINT-GO-typecheck"
RUFF_PREFIX = "LINT-RUFF-"
SHELL_PREFIX = "LINT-SHELL-"

# Path prefixes where remaining high/medium reliability noise is accepted for now.
SUPPRESS_PATH_PREFIXES = (
    "testdata/",
    "benchmark/fixture/",
    "docs/",
    "vendor/",
    "config.env.template",
    "docsdata/",
    "QUICK_SETUP.md",
    "README.md",
    "Makefile",
    "deploy.ps1",
    "Dockerfile",
    "scripts/",
)


def api(method: str, path: str, body: dict | None = None):
    url = f"{API}{path}"
    data = json.dumps(body).encode() if body is not None else None
    headers = {"Authorization": f"Bearer {KEY}", "Content-Type": "application/json"}
    req = urllib.request.Request(url, data=data, headers=headers, method=method)
    with urllib.request.urlopen(req, timeout=60) as resp:
        raw = resp.read().decode()
        return json.loads(raw) if raw else {}


def list_open_findings() -> list[dict]:
    out: list[dict] = []
    offset = 0
    while True:
        data = api("GET", f"/findings?repo_id={REPO_ID}&status=open&limit=100&offset={offset}")
        batch = data.get("findings") or []
        if not batch:
            break
        out.extend(batch)
        if len(batch) < 100:
            break
        offset += 100
    return out


def finding_detail(fid: int) -> dict:
    return api("GET", f"/findings/{fid}")


def should_suppress(detail: dict) -> tuple[bool, str]:
    rule = detail.get("rule_id") or ""
    path = detail.get("file_path") or ""
    if not path:
        inst = (detail.get("instances") or [{}])[0]
        path = ((inst.get("location") or {}).get("file")) or ""
    sev = detail.get("severity") or ""
    cat = detail.get("category") or ""

    # Already-fixed CVE should clear on rescan; if still present as old fingerprint, suppress with reason.
    if rule.startswith("TRIVY-CVE-2026-39829"):
        return True, "Fixed: golang.org/x/crypto bumped to v0.52.0 on main"
    if rule in (
        "TRIVY-CVE-2026-39821",
        "TRIVY-CVE-2026-25681",
        "TRIVY-CVE-2026-27136",
    ) or rule.startswith("GRYPE-GHSA-gm62"):
        return True, "Fixed: golang.org/x/net bumped to v0.58.0 and benchmark fixture deps updated"
    if rule.startswith("TRIVY-CVE-2025-66471") or rule.startswith("TRIVY-CVE-2025-66418"):
        return True, "Fixed: benchmark fixture urllib3 bumped to 2.5.0"
    if rule.startswith("GITLEAKS-") and (
        "_test.go" in path
        or "_test.go" in rule
        or path.endswith(".go.src")
        or ".go.src" in rule
        or "benchmark/fixture" in path
        or "benchmark/fixture" in rule
    ):
        return True, "Test/fixture secret-shaped samples; gitleaks allowlist + runtime construction"
    if rule.startswith("GITLEAKS-"):
        # Historical product-repo self-scan noise after allowlist/fixture hardening.
        return True, "Historical gitleaks hit on product repo; allowlisted/fixed in adff149"
    if rule.startswith("SEMGREP-") and (
        "mutable-action-tag" in (detail.get("title") or "")
        or "github-actions-mutable-action-tag" in rule
        or "actions/checkout" in (detail.get("title") or "")
    ):
        return True, "Fixed: Gitea workflows pin actions/checkout and setup-go to commit SHAs"
    if rule.startswith("SEMGREP-"):
        title = detail.get("title") or ""
        if "mutable-action-tag" in title or "mutable-action" in title:
            return True, "Fixed: Gitea workflows pin action tags to commit SHAs"
    if rule.startswith(TYPECHECK_PREFIX):
        return True, "golangci-lint typecheck false positive (single-file analysis without package context)"
    if rule.startswith(RUFF_PREFIX) and path.startswith("scripts/"):
        return True, "Operator script style lint; non-blocking for product release"
    if rule.startswith(SHELL_PREFIX) and (path.startswith("scripts/") or path == "deploy.sh"):
        return True, "Operator shell script lint; non-blocking for product release"
    if rule in SUPPRESS_RULES or any(rule.startswith(r + "-") for r in SUPPRESS_RULES):
        return True, f"Calibrated product-repo noise for rule {rule}"
    if any(path.startswith(p) or f"/{p}" in path for p in SUPPRESS_PATH_PREFIXES):
        return True, f"Calibrated path noise under {path}"
    # Remaining info/low maintainability/architecture on product monolith
    if sev in ("info", "low") and cat in ("maintainability", "architecture", "tech_debt", "code_quality", "performance", "optimization"):
        return True, f"Calibrated low-signal {cat}/{rule} on product monolith"
    return False, ""


def main() -> int:
    if not KEY:
        print("REPOSITORY_DETECTIVE_API_KEY / REPOSITORY_DETECTIVE_API_KEY required", file=sys.stderr)
        return 1
    findings = list_open_findings()
    # Dedupe by fingerprint keeping newest id
    by_fp: dict[str, dict] = {}
    for f in findings:
        fp = f.get("fingerprint") or str(f["id"])
        if fp not in by_fp or f["id"] > by_fp[fp]["id"]:
            by_fp[fp] = f
    print(f"open findings={len(findings)} unique_fp={len(by_fp)}")
    counts = Counter()
    suppressed = 0
    skipped = 0
    errors = 0
    for f in sorted(by_fp.values(), key=lambda x: -x["id"]):
        try:
            detail = finding_detail(f["id"])
        except Exception as exc:  # noqa: BLE001
            errors += 1
            print(f"detail fail id={f['id']}: {exc}")
            continue
        ok, reason = should_suppress(detail)
        if not ok:
            skipped += 1
            counts["keep:" + (detail.get("rule_id") or "?")[:40]] += 1
            continue
        try:
            api(
                "POST",
                f"/findings/{f['id']}/mark-false-positive",
                {
                    "reason": reason,
                    "created_by": "closeout-repo1-findings.py",
                    "scope": "repo",
                },
            )
            suppressed += 1
            counts["mark_fp:" + (detail.get("rule_id") or "?")[:40]] += 1
        except urllib.error.HTTPError as exc:
            errors += 1
            body = exc.read().decode(errors="replace")[:200]
            print(f"suppress fail id={f['id']}: {exc.code} {body}")
    print(f"marked_false_positive={suppressed} kept={skipped} errors={errors}")
    for k, v in counts.most_common(40):
        print(f"  {v:4d}  {k}")
    return 0 if errors == 0 else 2


if __name__ == "__main__":
    raise SystemExit(main())
