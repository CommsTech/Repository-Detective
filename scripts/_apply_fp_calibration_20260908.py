#!/usr/bin/env python3
"""One-shot: apply known-FP calibration to open findings (repo_id=1). Do not commit."""

from __future__ import annotations

import shutil
import sqlite3
import subprocess
from datetime import datetime, timezone
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
DB = ROOT / "data/repository-detective.db"
BAK = ROOT / "data/repository-detective.db.bak-fp-20260908"
NOW = datetime.now(timezone.utc).strftime("%Y-%m-%dT%H:%M:%SZ")
REPO_ID = 1
DOCKER_CONTAINER = "repository-detective"
DOCKER_DB = "/app/data/repository-detective.db"
DOCKER_BAK = "/app/data/repository-detective.db.bak-fp-20260908"


def open_counts(cur: sqlite3.Cursor) -> list[tuple]:
    return cur.execute(
        """
        SELECT severity, COUNT(1)
        FROM findings
        WHERE repository_id=? AND status='open'
        GROUP BY severity
        ORDER BY CASE severity
          WHEN 'critical' THEN 0 WHEN 'high' THEN 1 WHEN 'medium' THEN 2
          WHEN 'low' THEN 3 WHEN 'info' THEN 4 ELSE 5 END
        """,
        (REPO_ID,),
    ).fetchall()


def has_column(cur: sqlite3.Cursor, table: str, col: str) -> bool:
    return any(r[1] == col for r in cur.execute(f"PRAGMA table_info({table})").fetchall())


def ensure_backup() -> None:
    """Copy DB beside itself; fall back to docker when data/ is not host-writable."""
    if BAK.exists():
        print(f"backup exists: {BAK}")
        return
    try:
        shutil.copy2(DB, BAK)
        print(f"backup: {BAK}")
        return
    except PermissionError as err:
        print(f"host backup PermissionError ({err}); trying docker exec copy")
    result = subprocess.run(
        [
            "docker",
            "exec",
            DOCKER_CONTAINER,
            "cp",
            "-a",
            DOCKER_DB,
            DOCKER_BAK,
        ],
        capture_output=True,
        text=True,
        check=False,
    )
    if result.returncode != 0 or not BAK.exists():
        detail = (result.stderr or result.stdout or "").strip()
        raise SystemExit(
            f"backup failed (host + docker): exit={result.returncode} {detail}"
        )
    print(f"backup via docker: {BAK}")


def main() -> None:
    if not DB.exists():
        raise SystemExit(f"missing DB: {DB}")
    ensure_backup()

    con = sqlite3.connect(DB)
    con.row_factory = sqlite3.Row
    cur = con.cursor()
    file_col = "file_path" if has_column(cur, "findings", "file_path") else "file"
    has_cal_note = has_column(cur, "findings", "calibration_note")
    print(f"file_column={file_col} calibration_note={has_cal_note}")

    print("BEFORE open by severity:")
    for sev, n in open_counts(cur):
        print(f"  {sev}: {n}")

    # 1) Known-FP severity downgrade for medium/low
    rules = [
        ("gosec", ("G201", "G202"), f"{file_col} LIKE 'store/%'"),
        ("gosec", ("G104",), "1=1"),
        ("gosec", ("G204",), "1=1"),
        ("gosec", ("G304",), "1=1"),
        ("shellcheck", ("LINT-SHELL-2034",), "1=1"),
        ("golangci-lint", ("LINT-GO-typecheck",), "1=1"),
        ("static", ("OPT-NESTED-LOOP",), "1=1"),
        ("health", ("HEALTH-IGNORED-ERROR",), "1=1"),
    ]
    # Also match source aliases seen in dogfood dumps
    source_aliases = {
        "gosec": ("gosec",),
        "shellcheck": ("shellcheck",),
        "golangci-lint": ("golangci-lint", "golangci"),
        "static": ("static", "static-analysis", "optimization"),
        "health": ("health", "reliability", "maintainability"),
    }

    total_downgraded = 0
    for src_key, rule_ids, path_sql in rules:
        placeholders = ",".join("?" * len(rule_ids))
        sources = source_aliases[src_key]
        src_ph = ",".join("?" * len(sources))
        sql = f"""
            UPDATE findings
            SET severity='info',
                confidence=MIN(confidence, 0.55)
                {", calibration_note=CASE WHEN calibration_note='' OR calibration_note IS NULL THEN ? ELSE calibration_note END" if has_cal_note else ""}
            WHERE repository_id=?
              AND status='open'
              AND severity IN ('medium','low')
              AND rule_id IN ({placeholders})
              AND lower(source) IN ({src_ph})
              AND ({path_sql})
        """
        params: list = []
        if has_cal_note:
            params.append(f"Calibrated known-FP ({src_key}/{'|'.join(rule_ids)}) — informational; finding remains visible.")
        params.extend([REPO_ID, *rule_ids, *[s.lower() for s in sources]])
        cur.execute(sql, params)
        total_downgraded += cur.rowcount
        print(f"  downgraded {src_key}/{','.join(rule_ids)}: {cur.rowcount}")

    # Also catch OPT-NESTED-LOOP / HEALTH regardless of source spelling when rule_id matches
    for rule_id, note in (
        ("OPT-NESTED-LOOP", "Calibrated known-FP OPT-NESTED-LOOP — informational"),
        ("HEALTH-IGNORED-ERROR", "Calibrated known-FP HEALTH-IGNORED-ERROR — informational"),
        ("LINT-SHELL-2034", "Calibrated known-FP LINT-SHELL-2034 — informational"),
        ("LINT-GO-typecheck", "Calibrated known-FP LINT-GO-typecheck — informational"),
        ("G104", "Calibrated known-FP gosec G104 — informational"),
        ("G204", "Calibrated known-FP gosec G204 — informational"),
        ("G304", "Calibrated known-FP gosec G304 — informational"),
    ):
        sql = f"""
            UPDATE findings
            SET severity='info',
                confidence=MIN(confidence, 0.55)
                {", calibration_note=CASE WHEN calibration_note='' OR calibration_note IS NULL THEN ? ELSE calibration_note END" if has_cal_note else ""}
            WHERE repository_id=? AND status='open'
              AND severity IN ('medium','low') AND rule_id=?
        """
        params = []
        if has_cal_note:
            params.append(note)
        params.extend([REPO_ID, rule_id])
        cur.execute(sql, params)
        if cur.rowcount:
            print(f"  extra downgrade by rule_id {rule_id}: {cur.rowcount}")
            total_downgraded += cur.rowcount

    # G201/G202 store path only (any source casing)
    for rule_id in ("G201", "G202"):
        sql = f"""
            UPDATE findings
            SET severity='info',
                confidence=MIN(confidence, 0.55)
                {", calibration_note=CASE WHEN calibration_note='' OR calibration_note IS NULL THEN ? ELSE calibration_note END" if has_cal_note else ""}
            WHERE repository_id=? AND status='open'
              AND severity IN ('medium','low') AND rule_id=?
              AND {file_col} LIKE 'store/%'
        """
        params = []
        if has_cal_note:
            params.append(f"Calibrated known-FP gosec {rule_id} in store/ — informational")
        params.extend([REPO_ID, rule_id])
        cur.execute(sql, params)
        if cur.rowcount:
            print(f"  extra G201/G202 store {rule_id}: {cur.rowcount}")
            total_downgraded += cur.rowcount

    # 2) G703 high in patcher/% — prefer fingerprint suppressions
    g703 = cur.execute(
        f"""
        SELECT id, fingerprint, source, rule_id, severity, {file_col} AS file_path
        FROM findings
        WHERE repository_id=? AND status='open'
          AND rule_id='G703' AND severity='high'
          AND {file_col} LIKE 'patcher/%'
        """,
        (REPO_ID,),
    ).fetchall()
    suppressed = 0
    noted = 0
    for row in g703:
        fp = (row["fingerprint"] or "").strip()
        reason = "known-safe workspace write in patcher/ (gosec G703 calibrated)"
        if fp:
            exists = cur.execute(
                """
                SELECT id FROM finding_suppressions
                WHERE repository_id=? AND fingerprint=? AND active=1
                LIMIT 1
                """,
                (REPO_ID, fp),
            ).fetchone()
            if not exists:
                cur.execute(
                    """
                    INSERT INTO finding_suppressions (
                        repository_id, fingerprint, source, rule_id, category, severity,
                        scope, reason, created_by, expires_at, active, created_at, updated_at
                    ) VALUES (?, ?, ?, 'G703', 'security', 'high', 'repo', ?, 'fp-calibration-20260908', NULL, 1, ?, ?)
                    """,
                    (REPO_ID, fp, row["source"] or "gosec", reason, NOW, NOW),
                )
            cur.execute(
                "UPDATE findings SET status='false_positive' WHERE id=?",
                (row["id"],),
            )
            # learning note via calibration_note when available
            if has_cal_note:
                cur.execute(
                    """
                    UPDATE findings
                    SET calibration_note=CASE
                      WHEN calibration_note='' OR calibration_note IS NULL THEN ?
                      ELSE calibration_note END
                    WHERE id=?
                    """,
                    (reason, row["id"]),
                )
            suppressed += 1
        elif has_cal_note:
            cur.execute(
                """
                UPDATE findings
                SET calibration_note=CASE
                  WHEN calibration_note='' OR calibration_note IS NULL THEN ?
                  ELSE calibration_note END
                WHERE id=?
                """,
                (reason, row["id"]),
            )
            noted += 1
    print(f"G703 suppressions applied: {suppressed}; notes-only: {noted}")

    # 3) OPT-NESTED-LOOP / HEALTH on LICENSE, README.md, issues.md
    for rule_id in ("OPT-NESTED-LOOP", "HEALTH-IGNORED-ERROR"):
        sql = f"""
            UPDATE findings
            SET status='suppressed'
                {", calibration_note=CASE WHEN calibration_note='' OR calibration_note IS NULL THEN ? ELSE calibration_note END" if has_cal_note else ""}
            WHERE repository_id=? AND status='open'
              AND rule_id=?
              AND (
                {file_col} IN ('LICENSE','README.md','issues.md')
                OR {file_col} LIKE '%/LICENSE'
                OR {file_col} LIKE '%/README.md'
                OR {file_col} LIKE '%/issues.md'
              )
        """
        params = []
        if has_cal_note:
            params.append(f"Suppressed {rule_id} on docs/license path — non-code noise")
        params.extend([REPO_ID, rule_id])
        cur.execute(sql, params)
        print(f"  suppressed {rule_id} on LICENSE/README/issues.md: {cur.rowcount}")

    con.commit()

    print("AFTER open by severity:")
    after = open_counts(cur)
    for sev, n in after:
        print(f"  {sev}: {n}")
    total_open = sum(n for _, n in after)
    print(f"total_open={total_open} downgraded_ops≈{total_downgraded}")
    con.close()


if __name__ == "__main__":
    main()
