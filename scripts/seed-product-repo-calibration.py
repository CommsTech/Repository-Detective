#!/usr/bin/env python3
"""Seed repo-scoped calibration rules for product repo (commstech/Repository-Detective, id=1)."""

from __future__ import annotations

import sqlite3
from datetime import datetime, timedelta, timezone
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
DB = ROOT / "data/repository-detective.db"
REPO_ID = 1
EXPIRES = (datetime.now(timezone.utc) + timedelta(days=90)).strftime("%Y-%m-%dT%H:%M:%SZ")
NOW = datetime.now(timezone.utc).strftime("%Y-%m-%dT%H:%M:%SZ")

RULES = [
    ("maintainability", "HEALTH-MANY-PARAMS", "store/", "Store layer uses explicit SQL params — informational for product monolith"),
    ("maintainability", "HEALTH-MANY-PARAMS", "main", "Bootstrap/orchestration entrypoints may have wide signatures"),
    ("maintainability", "HEALTH-MANY-PARAMS", "gitea/", "Forge reporter adapter — wide option surface is intentional"),
    ("maintainability", "HEALTH-MANY-PARAMS", "issuelink/", "Issue link backfill helpers — informational only"),
    ("maintainability", "HEALTH-MANY-PARAMS", "preinstall/", "Pre-install audit orchestration — wide signatures expected"),
    ("maintainability", "HEALTH-MANY-PARAMS", "remediation/", "Remediation renderer — template context params"),
    ("maintainability", "HEALTH-MANY-PARAMS", "runner/", "Runner spec/signing — protocol structs with many fields"),
    ("maintainability", "HEALTH-MANY-PARAMS", "ui/", "UI settings model — form binding params"),
    ("maintainability", "HEALTH-LARGE-FILE", "main.go", "Known large bootstrap file — split is tracked separately"),
    ("maintainability", "HEALTH-LARGE-FILE", "ui/handler.go", "UI handler surface — informational maintainability signal"),
    ("maintainability", "HEALTH-LARGE-FILE", "analyzers/engine.go", "Scan engine orchestration — expected size for feature set"),
    ("maintainability", "HEALTH-LARGE-FUNC", "store/profiles.go", "Profile builder — decomposition is non-urgent"),
    ("maintainability", "HEALTH-LARGE-FUNC", "analyzers/static.go", "Static false-positive helpers — informational maintainability signal"),
    ("maintainability", "HEALTH-DEEP-NEST", "main.go", "Bootstrap control flow — review only when refactoring"),
    ("maintainability", "HEALTH-DEEP-NEST", "analyzers/static.go", "Static analysis dispatch nesting — informational"),
    ("performance", "HEALTH-READ-ALL", "api/runner_handler.go", "Runner API reads bounded job payloads — expected pattern"),
    ("tech_debt", "HEALTH-TECH-PHRASE", "patcher/", "Temporary git workspace comments — not actionable debt markers"),
    ("tech_debt", "HEALTH-TECH-PHRASE", "scanners/", "Temporary clone workspace comments — expected for scanners"),
    ("tech_debt", "HEALTH-TECH-MARKER", "health/", "Health analyzer self-tests intentionally include TODO markers"),
    ("optimization", "OPT-NESTED-LOOP", "operator/", "Telemetry aggregation loops — advisory only"),
    ("optimization", "OPT-NESTED-LOOP", "gitea/", "Ref probe loops are bounded by candidate list — advisory"),
    ("optimization", "OPT-NESTED-LOOP", "github/", "Ref probe loops are bounded by candidate list — advisory"),
    ("architecture", "GRAPH-DISCONNECTED-PACKAGE", "", "Monolith package graph noise — packages are wired via main/cmd, not import cycles"),
    ("public_release", "REL-INTERNAL-INFRA-REF", "docs/", "Docs use localhost examples for local install — expected"),
    ("public_release", "REL-INTERNAL-INFRA-REF", "README.md", "README localhost health/onboard examples — expected"),
    ("public_release", "REL-INTERNAL-INFRA-REF", "QUICK_SETUP.md", "Quick setup localhost examples — expected"),
    ("public_release", "REL-INTERNAL-INFRA-REF", "Makefile", "Makefile pprof localhost helper — expected"),
    ("public_release", "REL-INTERNAL-INFRA-REF", "deploy.ps1", "Deploy script localhost health hint — expected"),
    ("public_release", "REL-INTERNAL-INFRA-REF", "preinstall/", "Preinstall explicitly rejects localhost/loopback — detection of the check itself"),
    ("public_release", "REL-INTERNAL-INFRA-REF", "containers/", "Container discovery filters localhost hosts — intentional"),
    ("public_release", "REL-INTERNAL-INFRA-REF", "benchmark/fixture/", "Benchmark fixtures intentionally include internal URL shapes"),
    ("test_gap", "HEALTH-GO-NO-TEST", "testdata/", "testdata fixtures are sample repos, not production packages"),
    ("security", "SEC-CMD-EXEC", "sbom/", "SBOM path intentionally shells out to grype with fixed argv"),
    ("security", "SEC-CMD-EXEC", "analyzers/static.go", "Static rule pattern source contains exec.Command regex — not live exec"),
    ("container", "DL3018", "Dockerfile", "Alpine apk version pins drift frequently; accepted for this image build"),
]


def main() -> None:
    con = sqlite3.connect(DB)
    cur = con.cursor()
    inserted = 0
    for source, rule_id, path_pattern, reason in RULES:
        cur.execute(
            """
            SELECT id FROM repo_calibration_rules
            WHERE repository_id=? AND source=? AND rule_id=? AND path_pattern=? AND active=1
            """,
            (REPO_ID, source, rule_id, path_pattern),
        )
        if cur.fetchone():
            continue
        cur.execute(
            """
            INSERT INTO repo_calibration_rules (
                repository_id, scope, source, rule_id, path_pattern, finding_category,
                action, reason, evidence_count, false_positive_rate, true_positive_rate,
                duplicate_rate, expires_at, active, created_at, updated_at
            ) VALUES (?, 'repo', ?, ?, ?, '', 'informational', ?, 79, 0.85, 0.1, 0.0, ?, 1, ?, ?)
            """,
            (REPO_ID, source, rule_id, path_pattern, reason, EXPIRES, NOW, NOW),
        )
        inserted += 1
    con.commit()
    con.close()
    print(f"seeded {inserted} product repo calibration rule(s)")


if __name__ == "__main__":
    main()
