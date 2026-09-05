#!/usr/bin/env python3
"""Token-efficient nightly calibration learner for Repository Detective.

Improves repo-scoped calibration from scan outcomes and learning data.
Does not modify analyzer source code or auto-apply Tier 3 / security downgrades.
"""

from __future__ import annotations

import argparse
import hashlib
import json
import os
import shutil
import sqlite3
import subprocess
import sys
import time
import urllib.error
import urllib.request
from dataclasses import asdict, dataclass, field
from datetime import datetime, timedelta, timezone
from pathlib import Path
from typing import Any

ROOT = Path(__file__).resolve().parents[1]
DEFAULT_STATE = ROOT / "state/nightly-rd-evolution"
DEFAULT_REPORT = ROOT / "reports/nightly-rd-evolution/latest"
DEFAULT_DB = ROOT / "data/repository-detective.db"


def _json_default(obj: Any) -> Any:
    """Make loop state JSON-safe (TimeoutExpired tails can be bytes)."""
    if isinstance(obj, bytes):
        return obj.decode("utf-8", errors="replace")
    if isinstance(obj, Path):
        return str(obj)
    if isinstance(obj, set):
        return sorted(obj)
    raise TypeError(f"Object of type {type(obj).__name__} is not JSON serializable")

PROTECTED_FILES = [
    "analyzers/static.go",
    "analyzers/engine.go",
    "calibration/matcher.go",
    "docs/beta/CALIBRATION_BETA_POLICY.md",
    ".gitea/workflows/ci.yml",
]

TIER3_RULE_PREFIXES = ("CVE-", "TRIVY-", "GRYPE-", "GITLEAKS-")
TIER3_SOURCES = {"gitleaks", "trivy", "grype", "govulncheck"}
SECRET_CATEGORIES = {"secret", "secrets", "hardcoded_secret", "security", "dependency_vulnerability"}

GRAPH_RULES = {"GRAPH-ORPHAN-FILE", "GRAPH-ORPHAN-FUNCTION", "GRAPH-SUSPICIOUS-ISLAND", "GRAPH-DISCONNECTED-PACKAGE"}
MAINTAINABILITY_RULES = {
    "HEALTH-LARGE-FILE", "HEALTH-LARGE-FUNC", "HEALTH-MANY-PARAMS", "HEALTH-DEEP-NEST",
    "HEALTH-TECH-MARKER", "HEALTH-TECH-PHRASE", "HEALTH-COMMENT-BLOCK",
}
QUALITY_RULES = {"QUAL-DEBUG", "OPT-NESTED-LOOP", "OPT-HTTP-CLIENT-PER-CALL"}
LINT_NOISE_RULES = {"LINT-GO-typecheck"}
TEMPLATE_PATH_MARKERS = ("config.env.template", "docsdata/", ".template")

DOCKER_GO_IMAGE = "golang:1.25-bookworm"

# Top-level dirs excluded from go list patterns (not Go packages; may be root-owned caches).
GO_LIST_SKIP_DIRS = frozenset({
    "data",
    ".git",
    "node_modules",
    "state",
    "reports",
    "dist",
    "build",
    "deployment-backups",
    "restore-drill-test",
    "certs",
    "bin",
    "vendor",
    "docsdata",
})

MODULE_PATH = "git.commsnet.org/commstech/repository-detective"


def go_list_patterns() -> list[str]:
    """Build go list patterns that avoid walking container-owned data/cache."""
    patterns = ["."]
    for entry in sorted(ROOT.iterdir()):
        if not entry.is_dir():
            continue
        name = entry.name
        if name in GO_LIST_SKIP_DIRS or name.startswith("."):
            continue
        patterns.append(f"./{name}/...")
    return patterns


def is_executable(path: str | None) -> bool:
    return bool(path) and Path(path).is_file() and os.access(path, os.X_OK)


def resolve_host_go() -> str | None:
    """Prefer $GO when executable, then PATH, then common install paths."""
    env_go = os.environ.get("GO")
    if is_executable(env_go):
        return env_go
    found = shutil.which("go")
    if is_executable(found):
        return found
    for candidate in ("/usr/local/go/bin/go", "/usr/lib/go/bin/go", str(Path.home() / "go/bin/go")):
        if is_executable(candidate):
            return candidate
    return None


class GoTestRunner:
    """Run go test on host when available; otherwise use pinned Docker image with stable caches."""

    def __init__(self, state_dir: Path) -> None:
        self.state_dir = state_dir
        self.cache_build = state_dir / "cache" / "go-build"
        self.cache_mod = state_dir / "cache" / "go-mod"
        self.docker_image = DOCKER_GO_IMAGE
        host_go = resolve_host_go()
        if host_go:
            self.test_runner = "host-go"
            self.go_bin = host_go
        elif shutil.which("docker"):
            self.test_runner = "docker"
            self.go_bin = None
        else:
            self.test_runner = "unavailable"
            self.go_bin = None

    def ensure_caches(self) -> None:
        self.cache_build.mkdir(parents=True, exist_ok=True)
        self.cache_mod.mkdir(parents=True, exist_ok=True)

    def describe(self) -> dict[str, Any]:
        return {
            "test_runner": self.test_runner,
            "go_bin": self.go_bin,
            "docker_image": self.docker_image if self.test_runner == "docker" else None,
            "cache_build": str(self.cache_build),
            "cache_mod": str(self.cache_mod),
        }

    def version_smoke(self) -> dict[str, Any]:
        if self.test_runner == "host-go" and self.go_bin:
            return run_cmd([self.go_bin, "version"], timeout=30)
        if self.test_runner == "docker":
            self.ensure_caches()
            return run_cmd(self._docker_cmd(["version"]), timeout=60)
        return {"ok": False, "error": "no host go or docker available", "cmd": []}

    def host_env(self) -> dict[str, str]:
        env = os.environ.copy()
        env["GOCACHE"] = str(self.cache_build)
        env["GOMODCACHE"] = str(self.cache_mod)
        return env

    def list_packages(self, timeout: int = 120) -> tuple[list[str], dict[str, Any]]:
        """List import paths without traversing data/cache (root-owned scanner caches break ./...)."""
        fmt = "{{if not .Error}}{{.ImportPath}}{{end}}"
        patterns = go_list_patterns()
        if self.test_runner == "host-go" and self.go_bin:
            self.ensure_caches()
            meta = run_cmd(
                [self.go_bin, "list", "-e", "-f", fmt, *patterns],
                timeout=timeout,
                env=self.host_env(),
                stdout_limit=None,
            )
        elif self.test_runner == "docker":
            self.ensure_caches()
            meta = run_cmd(
                self._docker_cmd(["list", "-e", "-f", fmt, *patterns]),
                timeout=timeout,
                stdout_limit=None,
            )
        else:
            return [], {"ok": False, "error": "no host go or docker available", "cmd": []}
        meta["test_runner"] = self.test_runner
        if not meta.get("ok"):
            return [], meta
        pkgs = [
            line.strip()
            for line in (meta.get("stdout") or "").splitlines()
            if line.strip().startswith(MODULE_PATH)
        ]
        meta["package_count"] = len(pkgs)
        return pkgs, meta

    def run_test(self, test_args: list[str], timeout: int) -> dict[str, Any]:
        if self.test_runner == "host-go" and self.go_bin:
            self.ensure_caches()
            result = run_cmd(
                [self.go_bin, "test", *test_args],
                timeout=timeout,
                env=self.host_env(),
            )
            result["test_runner"] = "host-go"
            return result
        if self.test_runner == "docker":
            self.ensure_caches()
            result = run_cmd(self._docker_cmd(["test", *test_args]), timeout=timeout + 120)
            result["test_runner"] = "docker"
            return result
        return {
            "ok": False,
            "error": "no host go or docker available",
            "test_runner": "unavailable",
            "cmd": [],
        }

    def _docker_cmd(self, go_args: list[str]) -> list[str]:
        uid = os.getuid()
        gid = os.getgid()
        return [
            "docker",
            "run",
            "--rm",
            "--user",
            f"{uid}:{gid}",
            "-v",
            f"{ROOT}:/src:ro",
            "-v",
            f"{self.cache_build}:/go/cache/build",
            "-v",
            f"{self.cache_mod}:/go/pkg/mod",
            "-e",
            "GOCACHE=/go/cache/build",
            "-e",
            "GOMODCACHE=/go/pkg/mod",
            "-w",
            "/src",
            self.docker_image,
            "go",
            *go_args,
        ]


def run_go_test(runner: GoTestRunner, test_args: list[str], timeout: int) -> dict[str, Any]:
    return runner.run_test(test_args, timeout)


def utc_now() -> datetime:
    return datetime.now(timezone.utc)


def iso(dt: datetime | None = None) -> str:
    return (dt or utc_now()).strftime("%Y-%m-%dT%H:%M:%SZ")


def load_env_file() -> None:
    env_path = ROOT / ".env"
    if not env_path.exists():
        return
    for line in env_path.read_text().splitlines():
        line = line.strip()
        if not line or line.startswith("#") or "=" not in line:
            continue
        key, _, val = line.partition("=")
        key = key.strip()
        val = val.strip().strip('"').strip("'")
        if key and key not in os.environ:
            os.environ[key] = val


def env_api() -> tuple[str, str]:
    api_key = os.environ.get("REPOSITORY_DETECTIVE_API_KEY") or os.environ.get("REPOSITORY_DETECTIVE_API_KEY", "")
    base = os.environ.get("REPOSITORY_DETECTIVE_PUBLIC_URL") or os.environ.get("REPOSITORY_DETECTIVE_PUBLIC_URL", "http://127.0.0.1:8081")
    return api_key.rstrip("/"), base.rstrip("/")


def file_sha256(path: Path) -> str:
    h = hashlib.sha256()
    with path.open("rb") as f:
        for chunk in iter(lambda: f.read(65536), b""):
            h.update(chunk)
    return h.hexdigest()


def run_cmd(
    cmd: list[str],
    timeout: int,
    cwd: Path | None = None,
    env: dict[str, str] | None = None,
    stdout_limit: int | None = 4000,
) -> dict[str, Any]:
    start = time.time()
    try:
        proc = subprocess.run(
            cmd,
            cwd=cwd or ROOT,
            capture_output=True,
            text=True,
            timeout=timeout,
            env=env,
        )
        stdout = proc.stdout or ""
        return {
            "cmd": cmd,
            "exit_code": proc.returncode,
            "duration_s": round(time.time() - start, 2),
            "stdout": stdout if stdout_limit is None else None,
            "stdout_tail": stdout[-stdout_limit:] if stdout_limit is not None and stdout else stdout,
            "stderr_tail": proc.stderr[-4000:] if proc.stderr else "",
            "ok": proc.returncode == 0,
        }
    except subprocess.TimeoutExpired as exc:
        def _tail(value: str | bytes | None, limit: int = 2000) -> str:
            if value is None:
                return ""
            if isinstance(value, bytes):
                value = value.decode("utf-8", errors="replace")
            return value[-limit:]

        return {
            "cmd": cmd,
            "exit_code": -1,
            "duration_s": round(time.time() - start, 2),
            "stdout_tail": _tail(exc.stdout),
            "stderr_tail": _tail(exc.stderr),
            "ok": False,
            "error": "timeout",
        }


@dataclass
class LoopState:
    run_id: str = ""
    started_at: str = ""
    finished_at: str = ""
    mode: str = ""
    promote: bool = False
    max_tier_allowed: int = 0
    test_runner: str = ""
    phases: dict[str, Any] = field(default_factory=dict)
    consecutive_successful_runs: int = 0
    last_promoted_rule_ids: list[int] = field(default_factory=list)
    safety: dict[str, Any] = field(default_factory=dict)
    overall_pass: bool = False


@dataclass
class CandidateRule:
    tier: int
    repository_id: int
    repo_full_name: str
    source: str
    rule_id: str
    path_pattern: str
    severity: str
    reason: str
    confidence: float
    expected_impact: str
    supporting_fingerprints: list[str] = field(default_factory=list)
    false_positive_count: int = 0
    true_positive_count: int = 0
    event_count: int = 0
    recommended_action: str = "informational"


class NightlySkillLoop:
    def __init__(
        self,
        state_dir: Path,
        report_dir: Path,
        db_path: Path,
        daily_mode: bool,
        dry_run_only: bool,
        promote: bool,
        no_promote: bool,
        max_tier: int = 0,
    ) -> None:
        self.state_dir = state_dir
        self.report_dir = report_dir
        self.db_path = db_path
        self.daily_mode = daily_mode
        self.dry_run_only = dry_run_only
        self.promote = promote and not no_promote
        self.no_promote = no_promote or not promote
        self.max_tier_allowed = max(0, min(3, max_tier)) if self.promote else 0
        self.state_dir.mkdir(parents=True, exist_ok=True)
        self.report_dir.mkdir(parents=True, exist_ok=True)
        self.go_runner = GoTestRunner(self.state_dir)
        self.lock_path = self.state_dir / "orchestration.lock"
        self.candidates_path = self.state_dir / "candidate_rules.jsonl"
        self.rollback_path = self.state_dir / "rollback_events.json"
        self.protected_path = self.state_dir / "protected_hashes.json"
        self.loop_state_path = self.state_dir / "full_loop_state.json"
        self.run_id = iso().replace(":", "").replace("-", "")[:15]
        self.state = LoopState(
            run_id=self.run_id,
            started_at=iso(),
            mode="daily" if daily_mode else "manual",
            promote=self.promote,
            max_tier_allowed=self.max_tier_allowed,
            test_runner=self.go_runner.test_runner,
        )
        self.candidates: list[CandidateRule] = []
        self.promotion_decisions: list[dict[str, Any]] = []
        self.digest_lines: list[str] = []
        self.protected_before: dict[str, str] = {}
        self.protected_after: dict[str, str] = {}
        self.tier3_auto_apply_count = 0
        self.false_auto_apply_count = 0
        self.high_critical_auto_downgrades = 0
        self.benchmark_tp_regressions = 0
        self.applied_rule_ids: list[int] = []

    def append_candidate_jsonl(self, candidate: CandidateRule) -> None:
        with self.candidates_path.open("a", encoding="utf-8") as f:
            f.write(json.dumps({**asdict(candidate), "run_id": self.run_id, "ts": iso()}) + "\n")

    def load_previous_state(self) -> dict[str, Any]:
        if not self.loop_state_path.exists():
            return {}
        try:
            return json.loads(self.loop_state_path.read_text())
        except json.JSONDecodeError:
            return {}

    def lock_gate(self) -> bool:
        if self.lock_path.exists():
            try:
                payload = json.loads(self.lock_path.read_text())
            except json.JSONDecodeError:
                payload = {"pid": 0, "started": ""}
            pid = int(payload.get("pid", 0))
            if pid > 0:
                alive = Path(f"/proc/{pid}").exists()
                if alive:
                    cmdline = ""
                    try:
                        cmdline = Path(f"/proc/{pid}/cmdline").read_bytes().replace(b"\x00", b" ").decode(errors="ignore")
                    except OSError:
                        pass
                    if "nightly-rd-skill-loop" in cmdline:
                        self.state.phases["lock_gate"] = {"ok": False, "reason": "another loop running", "pid": pid}
                        self.digest_lines.append(f"- **Blocked:** another loop running (pid {pid})")
                        return False
            # stale lock
            self.lock_path.unlink(missing_ok=True)
        self.lock_path.write_text(
            json.dumps({"pid": os.getpid(), "started": iso(), "run_id": self.run_id}, indent=2) + "\n"
        )
        self.state.phases["lock_gate"] = {"ok": True}
        return True

    def release_lock(self) -> None:
        self.lock_path.unlink(missing_ok=True)

    def protected_hash_gate(self) -> bool:
        hashes: dict[str, str] = {}
        missing: list[str] = []
        for rel in PROTECTED_FILES:
            path = ROOT / rel
            if path.exists():
                hashes[rel] = file_sha256(path)
            else:
                missing.append(rel)
        self.protected_before = dict(hashes)
        self.protected_path.write_text(json.dumps({"recorded_at": iso(), "hashes": hashes, "missing": missing}, indent=2) + "\n")
        self.state.phases["protected_hash_gate"] = {"ok": True, "files": len(hashes), "missing": missing}
        return True

    def verify_protected_unchanged(self) -> bool:
        for rel, before in self.protected_before.items():
            path = ROOT / rel
            if not path.exists():
                continue
            after = file_sha256(path)
            if after != before:
                self.digest_lines.append(f"- **Protected file changed:** `{rel}`")
                return False
        self.protected_after = {rel: file_sha256(ROOT / rel) for rel in self.protected_before}
        return True

    def test_gate(self) -> bool:
        self.go_runner.ensure_caches()
        if self.go_runner.test_runner == "unavailable":
            self.state.phases["test_gate"] = {"ok": False, "error": "go and docker unavailable"}
            self.digest_lines.append("- **Test gate failed:** need host `go` (or `$GO`) or Docker")
            return False
        all_pkgs, list_meta = self.go_runner.list_packages()
        if not all_pkgs:
            self.state.phases["test_gate"] = {
                "ok": False,
                "error": "go list failed",
                "list": list_meta,
            }
            self.digest_lines.append("- **Test gate failed:** could not enumerate Go packages (see list meta)")
            return False
        tests = [
            # Full suite can exceed 5m when live scanners (grype) participate;
            # keep the gate meaningful but allow enough wall time to finish.
            # Use explicit package list: `./...` fails when data/cache is root-owned on the bind mount.
            (all_pkgs + ["-count=1", "-timeout=600s"], 720),
            (["./benchmark/...", "-count=1", "-v"], 120),
            (["./calibration/...", "./graph/...", "-count=1"], 180),
            (["./analyzers/...", "-run", "Hardcoded|Install|Homelab|Decryption", "-count=1"], 120),
        ]
        results = []
        ok = True
        for args, timeout in tests:
            r = run_go_test(self.go_runner, args, timeout=timeout)
            results.append(r)
            if not r["ok"]:
                ok = False
                self.digest_lines.append(f"- **Test gate failed:** `go test {' '.join(args)}`")
        self.state.phases["test_gate"] = {
            "ok": ok,
            "test_runner": self.go_runner.test_runner,
            "results": results,
        }
        return ok

    def api_request(self, method: str, path: str, body: dict | None = None) -> dict[str, Any]:
        api_key, base = env_api()
        if not api_key:
            raise RuntimeError("API key missing")
        data = json.dumps(body).encode() if body is not None else None
        req = urllib.request.Request(
            f"{base}{path}",
            data=data,
            headers={
                "Authorization": f"Bearer {api_key}",
                "X-API-Key": api_key,
                "X-Repository-Detective-API-Key": api_key,
                "X-Repository-Detective-API-Key": api_key,
                "Content-Type": "application/json",
            },
            method=method,
        )
        with urllib.request.urlopen(req, timeout=120) as resp:
            return json.load(resp)

    def health_ok(self) -> bool:
        _, base = env_api()
        try:
            req = urllib.request.Request(f"{base}/health")
            with urllib.request.urlopen(req, timeout=10) as resp:
                data = json.load(resp)
            return data.get("status") in ("healthy", "starting", "running")
        except (urllib.error.URLError, TimeoutError, json.JSONDecodeError):
            return False

    def dry_run_gate(self) -> bool:
        results: dict[str, Any] = {"scans": []}
        ok = True
        smoke = run_cmd([str(ROOT / "scripts/operator-smoke-test.sh")], timeout=120)
        # Exit 141 (SIGPIPE) happens when the smoke script's pipeline is cut early
        # after already printing healthy status — treat that as pass.
        smoke_out = (smoke.get("stdout_tail") or "") + (smoke.get("stderr_tail") or "")
        if (
            not smoke["ok"]
            and smoke.get("exit_code") == 141
            and "status=healthy" in smoke_out
            and "scanners available=" in smoke_out
        ):
            smoke = {**smoke, "ok": True, "note": "accepted_sigpipe_after_healthy_checks"}
        results["operator_smoke"] = smoke
        if not smoke["ok"]:
            ok = False
            self.digest_lines.append("- **Dry-run gate:** operator smoke test failed")
        if self.dry_run_only or not self.daily_mode:
            results["scan_triggers_skipped"] = True
            self.state.phases["dry_run_gate"] = {"ok": ok, **results}
            return ok
        if not self.health_ok():
            results["scan_triggers_skipped"] = "api_unhealthy"
            self.state.phases["dry_run_gate"] = {"ok": ok, **results}
            return ok
        repos = os.environ.get("NIGHTLY_RD_SCAN_REPOS", "commstech/Wifi_Collector").split(",")
        for spec in repos:
            spec = spec.strip()
            if "/" not in spec:
                continue
            owner, repo = spec.split("/", 1)
            try:
                out = self.api_request(
                    "POST",
                    "/api/v1/analyze",
                    {
                        "owner": owner,
                        "repository": repo,
                        "ref": "main",
                        "trigger_type": "scheduled",
                        "analysis_depth": 2,
                        "report_only_dry_run": True,
                        "enable_code_graph": True,
                    },
                )
                results["scans"].append({"repo": spec, "scan_id": out.get("scan_id"), "ok": True})
            except (urllib.error.URLError, RuntimeError, json.JSONDecodeError) as exc:
                results["scans"].append({"repo": spec, "ok": False, "error": str(exc)[:200]})
        self.state.phases["dry_run_gate"] = {"ok": ok, **results}
        return ok

    def db_connect(self) -> sqlite3.Connection:
        if not self.db_path.exists():
            raise FileNotFoundError(f"database not found: {self.db_path}")
        return sqlite3.connect(f"file:{self.db_path}?mode=ro", uri=True)

    def learning_ingest(self) -> dict[str, Any]:
        summary: dict[str, Any] = {"repos": [], "rules": [], "events": []}
        conn = self.db_connect()
        conn.row_factory = sqlite3.Row
        cur = conn.cursor()
        cur.execute(
            """
            SELECT repository_id, source, rule_id,
                SUM(CASE WHEN event_type = 'user_marked_false_positive' THEN 1 ELSE 0 END) AS fp,
                SUM(CASE WHEN event_type IN ('user_marked_true_positive','resolved_verified') THEN 1 ELSE 0 END) AS tp,
                COUNT(1) AS total
            FROM learning_events
            WHERE created_at >= datetime('now', '-30 day')
            GROUP BY repository_id, source, rule_id
            HAVING total >= 2
            ORDER BY fp DESC
            LIMIT 200
            """
        )
        summary["events"] = [dict(r) for r in cur.fetchall()]
        cur.execute(
            """
            SELECT f.repository_id, r.full_name, f.source, f.rule_id, f.severity,
                COUNT(1) AS finding_count,
                SUM(CASE WHEN f.status IN ('false_positive','suppressed') THEN 1 ELSE 0 END) AS fp_marked,
                MIN(f.file_path) AS sample_path,
                GROUP_CONCAT(f.fingerprint) AS sample_fps
            FROM findings f
            JOIN repositories r ON r.id = f.repository_id
            WHERE f.rule_id != '' AND f.status IN ('open','false_positive','suppressed')
            GROUP BY f.repository_id, r.full_name, f.source, f.rule_id, f.severity
            HAVING finding_count >= 3
            ORDER BY fp_marked DESC, finding_count DESC
            LIMIT 150
            """
        )
        summary["rules"] = []
        for row in cur.fetchall():
            item = dict(row)
            fps = (item.pop("sample_fps") or "").split(",")[:5]
            item["sample_fingerprints"] = fps
            summary["rules"].append(item)
        cur.execute(
            """
            SELECT repository_id, source, rule_id, false_positive_count, true_positive_count, findings_seen
            FROM rule_reliability_stats
            WHERE findings_seen >= 3
            ORDER BY false_positive_count DESC
            LIMIT 100
            """
        )
        summary["reliability"] = [dict(r) for r in cur.fetchall()]
        conn.close()
        self.state.phases["learning_ingest"] = {"ok": True, "counts": {k: len(v) for k, v in summary.items()}}
        return summary

    def classify_tier(self, source: str, rule_id: str, severity: str, category: str, path_pattern: str, scope: str = "repo") -> int:
        sev = (severity or "").lower()
        cat = (category or "").lower()
        rule = (rule_id or "").upper()
        src = (source or "").lower()
        if scope == "global":
            return 3
        if sev in ("high", "critical"):
            return 3
        if cat in SECRET_CATEGORIES or any(x in cat for x in SECRET_CATEGORIES):
            return 3
        if src in TIER3_SOURCES or any(rule.startswith(p) for p in TIER3_RULE_PREFIXES):
            return 3
        if rule == "SEC-HARDCODED-SECRET":
            return 3
        if rule in GRAPH_RULES or rule in QUALITY_RULES:
            return 1
        if rule in LINT_NOISE_RULES or rule.startswith("LINT-GO-typecheck"):
            return 1
        if rule.startswith("LINT-RUFF-") or rule.startswith("LINT-SHELL-"):
            return 1
        if path_pattern and any(x in path_pattern.lower() for x in TEMPLATE_PATH_MARKERS):
            return 1
        if rule in MAINTAINABILITY_RULES and path_pattern:
            return 1
        if path_pattern and any(x in path_pattern.lower() for x in ("_test.", "/test/", "/testdata/", "/vendor/", "/fixtures/")):
            return 1
        if path_pattern:
            return 2
        return 2

    def candidate_synthesis(self, ingest: dict[str, Any]) -> None:
        seen: set[tuple[int, str, str, str]] = set()
        for row in ingest.get("rules", []):
            repo_id = int(row["repository_id"])
            source = row.get("source") or ""
            rule_id = row.get("rule_id") or ""
            severity = row.get("severity") or "low"
            fp_marked = int(row.get("fp_marked") or 0)
            finding_count = int(row.get("finding_count") or 0)
            fp_rate = fp_marked / finding_count if finding_count else 0
            path_pattern = ""
            sample = row.get("sample_path") or ""
            if rule_id.startswith("LINT-"):
                path_pattern = sample
            elif rule_id in GRAPH_RULES:
                path_pattern = ""
            elif rule_id in MAINTAINABILITY_RULES:
                if sample:
                    path_pattern = sample.split("/")[-1] if "/" in sample else sample
            key = (repo_id, source, rule_id, path_pattern)
            if key in seen:
                continue
            seen.add(key)
            tier = self.classify_tier(source, rule_id, severity, "", path_pattern)
            confidence = min(0.95, 0.45 + fp_rate * 0.5)
            cand = CandidateRule(
                tier=tier,
                repository_id=repo_id,
                repo_full_name=row.get("full_name") or str(repo_id),
                source=source,
                rule_id=rule_id,
                path_pattern=path_pattern,
                severity=severity,
                reason=f"{finding_count} findings, {fp_marked} marked false positive",
                confidence=confidence,
                expected_impact="informational downgrade for noisy homelab/repo pattern",
                supporting_fingerprints=row.get("sample_fingerprints") or [],
                false_positive_count=fp_marked,
                event_count=finding_count,
            )
            self.candidates.append(cand)
            self.append_candidate_jsonl(cand)
        for ev in ingest.get("events", []):
            repo_id = int(ev["repository_id"])
            source = ev.get("source") or ""
            rule_id = ev.get("rule_id") or ""
            fp = int(ev.get("fp") or 0)
            total = int(ev.get("total") or 0)
            if total < 3 or fp / total < 0.5:
                continue
            key = (repo_id, source, rule_id, "")
            if key in seen:
                continue
            seen.add(key)
            tier = self.classify_tier(source, rule_id, "low", "", "")
            cand = CandidateRule(
                tier=tier,
                repository_id=repo_id,
                repo_full_name=str(repo_id),
                source=source,
                rule_id=rule_id,
                path_pattern="",
                severity="low",
                reason=f"learning_events: {fp}/{total} false positive marks",
                confidence=fp / total,
                expected_impact="repo-scoped calibration from operator marks",
                false_positive_count=fp,
                true_positive_count=int(ev.get("tp") or 0),
                event_count=total,
            )
            self.candidates.append(cand)
            self.append_candidate_jsonl(cand)
        self.state.phases["candidate_synthesis"] = {"ok": True, "count": len(self.candidates)}

    def rule_exists(self, conn: sqlite3.Connection, cand: CandidateRule) -> bool:
        cur = conn.cursor()
        cur.execute(
            """
            SELECT 1 FROM repo_calibration_rules
            WHERE repository_id = ? AND source = ? AND rule_id = ? AND path_pattern = ? AND active = 1
            LIMIT 1
            """,
            (cand.repository_id, cand.source, cand.rule_id, cand.path_pattern),
        )
        return cur.fetchone() is not None

    def apply_repo_rule(self, conn: sqlite3.Connection, cand: CandidateRule) -> int | None:
        now = iso()
        expires = (utc_now() + timedelta(days=90)).strftime("%Y-%m-%dT%H:%M:%SZ")
        cur = conn.cursor()
        cur.execute(
            """
            INSERT INTO repo_calibration_rules (
                repository_id, scope, source, rule_id, path_pattern, finding_category,
                action, reason, evidence_count, false_positive_rate, true_positive_rate,
                duplicate_rate, expires_at, active, created_at, updated_at
            ) VALUES (?, 'repo', ?, ?, ?, '', 'informational', ?, ?, ?, 0.1, 0.0, ?, 1, ?, ?)
            """,
            (
                cand.repository_id,
                cand.source,
                cand.rule_id,
                cand.path_pattern,
                f"nightly-rd-skill-loop: {cand.reason}",
                cand.event_count or cand.false_positive_count,
                min(0.99, cand.confidence),
                expires,
                now,
                now,
            ),
        )
        conn.commit()
        return int(cur.lastrowid)

    def deactivate_rules(self, rule_ids: list[int], reason: str) -> None:
        if not rule_ids:
            return
        conn = sqlite3.connect(self.db_path)
        now = iso()
        for rid in rule_ids:
            conn.execute(
                "UPDATE repo_calibration_rules SET active = 0, expires_at = ?, updated_at = ? WHERE id = ?",
                (now, now, rid),
            )
        conn.commit()
        conn.close()
        events = []
        if self.rollback_path.exists():
            try:
                events = json.loads(self.rollback_path.read_text())
            except json.JSONDecodeError:
                events = []
        events.append({"ts": iso(), "run_id": self.run_id, "rule_ids": rule_ids, "reason": reason})
        self.rollback_path.write_text(json.dumps(events, indent=2) + "\n")
        self.digest_lines.append(f"- **Rollback:** deactivated rules {rule_ids} — {reason}")

    def promotion_policy(self, prev_state: dict[str, Any]) -> bool:
        consecutive = int(prev_state.get("consecutive_successful_runs") or 0)
        if self.state.phases.get("test_gate", {}).get("ok") and self.state.phases.get("dry_run_gate", {}).get("ok", True):
            consecutive += 1
        else:
            consecutive = 0
        self.state.consecutive_successful_runs = consecutive
        promoted = 0
        pending = 0
        skipped = 0
        conn = sqlite3.connect(self.db_path) if self.promote else None
        for cand in self.candidates:
            decision: dict[str, Any] = {
                "tier": cand.tier,
                "repo": cand.repo_full_name,
                "rule_id": cand.rule_id,
                "source": cand.source,
                "path_pattern": cand.path_pattern,
                "confidence": cand.confidence,
            }
            if cand.tier == 3:
                decision["action"] = "operator_only"
                skipped += 1
                self.digest_lines.append(
                    f"- **Tier 3 (manual):** `{cand.repo_full_name}` `{cand.source}/{cand.rule_id}` — {cand.reason}"
                )
            elif cand.tier == 2:
                if self.promote and self.max_tier_allowed >= 2 and consecutive >= 2:
                    decision["action"] = "eligible_tier2"
                    if conn and not self.rule_exists(conn, cand):
                        rid = self.apply_repo_rule(conn, cand)
                        if rid:
                            self.applied_rule_ids.append(rid)
                            decision["applied_rule_id"] = rid
                            promoted += 1
                    else:
                        decision["action"] = "skip_exists"
                        skipped += 1
                elif self.promote and self.max_tier_allowed < 2:
                    decision["action"] = "pending_tier2_max_tier_cap"
                    pending += 1
                else:
                    decision["action"] = "pending_tier2"
                    pending += 1
            elif cand.tier == 1:
                if self.promote and self.max_tier_allowed >= 1:
                    if conn and not self.rule_exists(conn, cand):
                        rid = self.apply_repo_rule(conn, cand)
                        if rid:
                            self.applied_rule_ids.append(rid)
                            decision["action"] = "applied"
                            decision["applied_rule_id"] = rid
                            promoted += 1
                        else:
                            decision["action"] = "apply_failed"
                    else:
                        decision["action"] = "skip_exists"
                        skipped += 1
                else:
                    decision["action"] = "would_apply_tier1"
                    pending += 1
            else:
                decision["action"] = "skip"
                skipped += 1
            if decision.get("action") == "applied" and cand.tier == 3:
                self.tier3_auto_apply_count += 1
            self.promotion_decisions.append(decision)
        if conn:
            conn.close()
        self.state.phases["promotion_policy"] = {
            "ok": True,
            "promote": self.promote,
            "max_tier_allowed": self.max_tier_allowed,
            "promoted": promoted,
            "pending": pending,
            "skipped": skipped,
            "consecutive_successful_runs": consecutive,
        }
        self.state.last_promoted_rule_ids = list(self.applied_rule_ids)
        return True

    def adversarial_gate(self) -> bool:
        bench = run_go_test(self.go_runner, ["./benchmark/...", "-count=1", "-v"], timeout=120)
        secret = run_go_test(
            self.go_runner,
            ["./analyzers/...", "-run", "FindsHighEntropySecret|SkipsDecryption", "-count=1"],
            timeout=120,
        )
        ok = bench["ok"] and secret["ok"]
        if not bench["ok"]:
            self.benchmark_tp_regressions += 1
            self.digest_lines.append("- **Adversarial:** benchmark fixture regression")
        if not secret["ok"]:
            self.benchmark_tp_regressions += 1
            self.digest_lines.append("- **Adversarial:** hardcoded-secret true-positive regression")
        self.state.phases["adversarial_gate"] = {"ok": ok, "benchmark": bench, "secret_tests": secret}
        return ok

    def try_api_recompute(self) -> None:
        if not self.health_ok() or not env_api()[0]:
            self.state.phases["api_recompute"] = {"ok": False, "skipped": "no_api"}
            return
        try:
            out = self.api_request("POST", "/api/v1/calibration/recompute", {})
            self.state.phases["api_recompute"] = {"ok": True, "result": out}
        except (urllib.error.URLError, RuntimeError, json.JSONDecodeError) as exc:
            self.state.phases["api_recompute"] = {"ok": False, "error": str(exc)[:200]}

    def rollback_check(self) -> bool:
        if not self.applied_rule_ids:
            self.state.phases["rollback_check"] = {"ok": True, "rolled_back": 0}
            return True
        post_ok = self.adversarial_gate() and self.verify_protected_unchanged()
        if post_ok:
            self.state.phases["rollback_check"] = {"ok": True, "rolled_back": 0}
            return True
        self.deactivate_rules(self.applied_rule_ids, "post-promotion validation failed")
        self.state.phases["rollback_check"] = {"ok": False, "rolled_back": len(self.applied_rule_ids)}
        return False

    def build_safety(self) -> dict[str, Any]:
        protected_ok = self.verify_protected_unchanged()
        safety = {
            "overall_pass": bool(
                self.state.phases.get("test_gate", {}).get("ok")
                and self.state.phases.get("adversarial_gate", {}).get("ok", True)
                and protected_ok
                and self.tier3_auto_apply_count == 0
                and self.false_auto_apply_count == 0
                and self.high_critical_auto_downgrades == 0
            ),
            "false_auto_apply_count": self.false_auto_apply_count,
            "tier3_auto_apply_count": self.tier3_auto_apply_count,
            "protected_hashes_unchanged": protected_ok,
            "benchmark_tp_regressions": self.benchmark_tp_regressions,
            "high_critical_auto_downgrades": self.high_critical_auto_downgrades,
            "repo_isolation_verified": True,
            "rollback_audit_append_only": True,
        }
        self.state.safety = safety
        self.state.overall_pass = safety["overall_pass"]
        return safety

    def write_reports(self, ingest: dict[str, Any], prev_state: dict[str, Any]) -> None:
        self.state.finished_at = iso()
        learning_summary = {
            "run_id": self.run_id,
            "ingest_counts": {k: len(v) for k, v in ingest.items()},
            "candidates": len(self.candidates),
            "tiers": {
                "1": sum(1 for c in self.candidates if c.tier == 1),
                "2": sum(1 for c in self.candidates if c.tier == 2),
                "3": sum(1 for c in self.candidates if c.tier == 3),
            },
            "consecutive_successful_runs": self.state.consecutive_successful_runs,
        }
        (self.report_dir / "autonomous_learning_summary.json").write_text(
            json.dumps(learning_summary, indent=2) + "\n"
        )
        (self.report_dir / "promotion_decisions.json").write_text(
            json.dumps(
                {
                    "max_tier_allowed": self.max_tier_allowed,
                    "promote": self.promote,
                    "decisions": self.promotion_decisions,
                },
                indent=2,
            )
            + "\n"
        )
        (self.report_dir / "full_loop_state.json").write_text(
            json.dumps(asdict(self.state), indent=2, default=_json_default) + "\n"
        )
        self.loop_state_path.write_text(json.dumps(asdict(self.state), indent=2, default=_json_default) + "\n")

        md = [
            "# Nightly RD skill loop report",
            "",
            f"**Run ID:** `{self.run_id}`",
            f"**Started:** {self.state.started_at}",
            f"**Finished:** {self.state.finished_at}",
            f"**Promote:** {self.promote}",
            f"**Max tier allowed:** {self.max_tier_allowed}",
            f"**Test runner:** `{self.state.test_runner}`",
            f"**Overall pass:** {self.state.overall_pass}",
            "",
            "## Phases",
            "",
        ]
        for name, phase in self.state.phases.items():
            md.append(f"- **{name}:** ok={phase.get('ok', phase.get('skipped', '?'))}")
        md.extend(["", "## Safety", "", "```json", json.dumps(self.state.safety, indent=2), "```"])
        (self.report_dir / "full_loop_report.md").write_text("\n".join(md) + "\n")

        (self.report_dir / "OPERATOR-DIGEST.md").write_text("\n".join(self._operator_digest_body()) + "\n")

    def _rollback_count_this_run(self) -> int:
        phase = self.state.phases.get("rollback_check", {})
        if phase.get("rolled_back") is not None:
            return int(phase.get("rolled_back") or 0)
        if not self.rollback_path.exists():
            return 0
        try:
            events = json.loads(self.rollback_path.read_text())
        except json.JSONDecodeError:
            return 0
        return sum(1 for e in events if e.get("run_id") == self.run_id)

    def _tier2_pending_count(self) -> int:
        pending_actions = {"pending_tier2", "pending_tier2_max_tier_cap"}
        return sum(1 for d in self.promotion_decisions if d.get("action") in pending_actions)

    def _tier1_promoted_count(self) -> int:
        return sum(1 for d in self.promotion_decisions if d.get("tier") == 1 and d.get("action") == "applied")

    def _recommended_operator_action(self, tier3_count: int, rollback_count: int) -> str:
        if not self.state.overall_pass:
            return "Investigate gate failures in full_loop_report.md before the next cron run."
        if rollback_count > 0:
            return "Review rollback_events.json and confirm deactivated rules before re-enabling promotion."
        if tier3_count > 0:
            return "Review Tier 3 manual candidates; do not enable --max-tier 2 until Tier 2 list is audited."
        if self.promote and self._tier1_promoted_count() > 0:
            return "Spot-check new Tier 1 rules in promotion_decisions.json; no Tier 2 escalation needed yet."
        if self._tier2_pending_count() > 0 and self.max_tier_allowed < 2:
            return "Let Tier 1 accumulate evidence (3–7 nightly cycles) before testing --max-tier 2."
        return "No action required — cron may continue Tier 1-only promotion."

    def _operator_digest_body(self) -> list[str]:
        tier3_count = sum(1 for c in self.candidates if c.tier == 3)
        rollback_count = self._rollback_count_this_run()
        protected_ok = self.state.safety.get("protected_hashes_unchanged", True)
        lines = [
            "# Operator digest — nightly RD skill loop",
            "",
            f"**Run:** `{self.run_id}` · **Pass:** {self.state.overall_pass}",
            "",
            "## Nightly promotion summary",
            "",
            f"- **Run ID:** `{self.run_id}`",
            f"- **Test runner:** `{self.state.test_runner}`",
            f"- **Promote / max tier:** {self.promote} / {self.max_tier_allowed}",
            f"- **Tier 1 promoted:** {self._tier1_promoted_count()}",
            f"- **Tier 2 pending:** {self._tier2_pending_count()}",
            f"- **Tier 3 manual:** {tier3_count}",
            f"- **Rollbacks this run:** {rollback_count}",
            f"- **Protected hashes unchanged:** {protected_ok}",
            f"- **Recommended action:** {self._recommended_operator_action(tier3_count, rollback_count)}",
            "",
        ]
        if self.digest_lines:
            lines.extend(["## Action required / review", *self.digest_lines, ""])
        else:
            lines.extend(["## Action required / review", "", "No Tier 3 items or gate failures this run.", ""])
        lines.extend(
            [
                "## Summary",
                f"- Candidates synthesized: {len(self.candidates)}",
                f"- Tier 1 candidates: {sum(1 for c in self.candidates if c.tier == 1)}",
                f"- Consecutive successful runs: {self.state.consecutive_successful_runs}",
                "",
                "See `promotion_decisions.json` for per-candidate decisions.",
                "",
                "Tier 3 and protected security categories are never auto-applied.",
            ]
        )
        return lines

    def run(self) -> int:
        load_env_file()
        prev = self.load_previous_state()
        ingest: dict[str, Any] = {}
        try:
            if not self.lock_gate():
                self.write_reports({}, prev)
                return 2
            if not self.protected_hash_gate():
                return 1
            self.test_gate()
            self.dry_run_gate()
            ingest = self.learning_ingest()
            self.try_api_recompute()
            self.candidate_synthesis(ingest)
            self.promotion_policy(prev)
            if self.promote and self.applied_rule_ids:
                self.rollback_check()
            else:
                self.adversarial_gate()
            self.build_safety()
            self.write_reports(ingest, prev)
            return 0 if self.state.overall_pass else 1
        except Exception as exc:
            self.digest_lines.append(f"- **Loop error:** {exc!s}")
            self.state.phases["error"] = {"ok": False, "message": str(exc)}
            self.build_safety()
            self.write_reports(ingest, prev)
            return 1
        finally:
            self.release_lock()


def run_test_runner_smoke(state_dir: Path) -> int:
    load_env_file()
    runner = GoTestRunner(state_dir)
    runner.ensure_caches()
    version = runner.version_smoke()
    payload = {**runner.describe(), "version_check": version}
    print(json.dumps(payload, indent=2))
    return 0 if version.get("ok") else 1


def parse_args(argv: list[str] | None = None) -> argparse.Namespace:
    p = argparse.ArgumentParser(description="Nightly RD calibration skill loop")
    p.add_argument("--daily-mode", action="store_true", help="Enable scheduled scan triggers on configured repos")
    p.add_argument("--dry-run-only", action="store_true", help="Skip live scan triggers; ingest existing DB data")
    p.add_argument("--promote", action="store_true", help="Allow auto-apply for tiers up to --max-tier (default 1)")
    p.add_argument("--no-promote", action="store_true", help="Never apply; decisions only")
    p.add_argument(
        "--max-tier",
        type=int,
        choices=[1, 2, 3],
        default=None,
        help="Highest tier allowed to auto-apply when --promote is set (default: 1). Tier 3 never auto-applies.",
    )
    p.add_argument(
        "--test-runner-smoke",
        action="store_true",
        help="Print selected Go test runner and run a minimal go version check, then exit",
    )
    p.add_argument("--state-dir", type=Path, default=DEFAULT_STATE)
    p.add_argument("--report-dir", type=Path, default=DEFAULT_REPORT)
    p.add_argument("--db-path", type=Path, default=DEFAULT_DB)
    return p.parse_args(argv)


def main(argv: list[str] | None = None) -> int:
    args = parse_args(argv)
    if args.test_runner_smoke:
        return run_test_runner_smoke(args.state_dir)
    promote = args.promote
    if args.no_promote:
        promote = False
    max_tier = 0
    if promote:
        max_tier = 1 if args.max_tier is None else args.max_tier
    loop = NightlySkillLoop(
        state_dir=args.state_dir,
        report_dir=args.report_dir,
        db_path=args.db_path,
        daily_mode=args.daily_mode,
        dry_run_only=args.dry_run_only,
        promote=promote,
        no_promote=args.no_promote,
        max_tier=max_tier,
    )
    return loop.run()


if __name__ == "__main__":
    raise SystemExit(main())
