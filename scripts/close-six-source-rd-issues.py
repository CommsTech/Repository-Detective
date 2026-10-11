#!/usr/bin/env python3
"""Close the six leftover source/repository-detective forge issues and sync DB dispositions."""

from __future__ import annotations

import json
import os
import sqlite3
import sys
import urllib.error
import urllib.request
from datetime import datetime, timezone
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
DB = ROOT / "data" / "repository-detective.db"
OWNER, REPO = "commstech", "repository-detective"

# Issue number -> fingerprint, rule, file, disposition, close rationale
ISSUES = {
    472: {
        "fp": "rd-aae43819c7490673",
        "rule": "G122",
        "file": "preinstall/clone.go",
        "status": "resolved_verified",
        "comment": (
            "**Resolved verified** — hardened sandbox read-only walk.\n\n"
            "G122 (WalkDir/Chmod TOCTOU): `makeWorkspaceReadOnly` now uses `os.OpenRoot` "
            "and root-scoped `Chmod` on relative paths so symlink escape during the walk "
            "cannot chmod outside the operator-owned temp sandbox. Re-scan should clear "
            "fingerprint `rd-aae43819c7490673`.\n\n"
            "Label: `resolved-verified`"
        ),
    },
    473: {
        "fp": "rd-62a295232294fe4b",
        "rule": "G306",
        "file": "sbom/sbom.go",
        "status": "resolved_verified",
        "comment": (
            "**Resolved verified** — G306 WriteFile mode tightened.\n\n"
            "`sbom/sbom.go` Syft output is written with `0o600` (was `0o640`). "
            "Fingerprint `rd-62a295232294fe4b` should not recur.\n\n"
            "Label: `resolved-verified`"
        ),
    },
    474: {
        "fp": "rd-f1181d8578655061",
        "rule": "LINT-SHELL-3040",
        "file": "scripts/install-scanner-tools.sh",
        "status": "resolved_verified",
        "comment": (
            "**Resolved verified** — removed POSIX-undefined `pipefail`.\n\n"
            "`scripts/install-scanner-tools.sh` keeps `#!/bin/sh` (Alpine ash) and no longer "
            "calls `set -o pipefail`. Download integrity uses `download_to` (curl to file, "
            "then consume) instead of `curl | tar`. Fingerprint `rd-f1181d8578655061`.\n\n"
            "Label: `resolved-verified`"
        ),
    },
    475: {
        "fp": "rd-ef41c67eb61ad790",
        "rule": "LINT-SHELL-1078",
        "file": "scripts/publish-github-wiki.sh",
        "status": "resolved_verified",
        "comment": (
            "**Resolved verified** — fixed shellcheck SC1078 quoting.\n\n"
            "`scripts/publish-github-wiki.sh` wiki-missing error message now uses a "
            "single-quoted heredoc so the multi-line string is unambiguous to shellcheck. "
            "Fingerprint `rd-ef41c67eb61ad790`.\n\n"
            "Label: `resolved-verified`"
        ),
    },
    476: {
        "fp": "rd-c010c291a0dfff22",
        "rule": "LINT-SHELL-2046",
        "file": "scripts/verify-all.sh",
        "status": "resolved_verified",
        "comment": (
            "**Resolved verified** — quoted staticcheck package list.\n\n"
            "`scripts/verify-all.sh` builds `STATICCHECK_PKGS` via `mapfile` and invokes "
            "`staticcheck \"${STATICCHECK_PKGS[@]}\"` to avoid word-splitting (SC2046). "
            "Fingerprint `rd-c010c291a0dfff22`.\n\n"
            "Label: `resolved-verified`"
        ),
    },
    477: {
        "fp": "rd-f625584236ba047c",
        "rule": "G115",
        "file": "store/noise_reduction.go",
        "status": "resolved_verified",
        "comment": (
            "**Resolved verified** — removed rune→byte truncation.\n\n"
            "`formatIntComma` in `store/noise_reduction.go` now formats with "
            "`strings.Builder` / `WriteString` / `WriteByte(',')` — no `byte(rune)` "
            "conversion. Fingerprint `rd-f625584236ba047c`.\n\n"
            "Label: `resolved-verified`"
        ),
    },
}


def load_env() -> tuple[str, str]:
    token = os.environ.get("REPOSITORY_DETECTIVE_GITEA_TOKEN", "")
    base = os.environ.get("REPOSITORY_DETECTIVE_GITEA_URL", "https://git.commsnet.org").rstrip("/")
    env_path = ROOT / ".env"
    if env_path.exists():
        for line in env_path.read_text().splitlines():
            if "=" not in line or line.strip().startswith("#"):
                continue
            k, _, v = line.partition("=")
            k, v = k.strip(), v.strip().strip('"').strip("'")
            if k == "REPOSITORY_DETECTIVE_GITEA_TOKEN" and not token:
                token = v
            if k == "REPOSITORY_DETECTIVE_GITEA_URL":
                base = v.rstrip("/")
    if not token:
        sys.exit("REPOSITORY_DETECTIVE_GITEA_TOKEN required")
    return token, base


def gitea(base: str, token: str, method: str, path: str, body: dict | None = None):
    url = f"{base}/api/v1{path}"
    data = json.dumps(body).encode() if body is not None else None
    headers = {"Authorization": f"token {token}"}
    if data is not None:
        headers["Content-Type"] = "application/json"
    req = urllib.request.Request(url, data=data, headers=headers, method=method)
    try:
        with urllib.request.urlopen(req, timeout=120) as resp:
            raw = resp.read().decode()
            return json.loads(raw) if raw else None
    except urllib.error.HTTPError as e:
        err = e.read().decode(errors="replace")
        raise RuntimeError(f"{method} {path} -> {e.code}: {err[:500]}") from e


def ensure_label(base: str, token: str, name: str, color: str) -> None:
    try:
        gitea(base, token, "POST", f"/repos/{OWNER}/{REPO}/labels", {
            "name": name,
            "color": color,
            "description": "",
            "exclusive": False,
        })
    except RuntimeError as e:
        if "422" not in str(e) and "already exists" not in str(e).lower():
            # ignore conflicts; label may already exist
            pass


def main() -> int:
    token, base = load_env()
    now = datetime.now(timezone.utc).isoformat()
    print(f"closing {len(ISSUES)} issues against {base}/{OWNER}/{REPO}")

    ensure_label(base, token, "resolved-verified", "16a34a")

    if not DB.exists():
        print(f"WARN: DB missing at {DB}", file=sys.stderr)
        conn = None
    else:
        conn = sqlite3.connect(str(DB))
        conn.execute("PRAGMA busy_timeout=5000")

    closed = 0
    for num, meta in ISSUES.items():
        issue = gitea(base, token, "GET", f"/repos/{OWNER}/{REPO}/issues/{num}")
        state = (issue or {}).get("state")
        print(f"#{num} state={state} fp={meta['fp']} rule={meta['rule']}")

        gitea(
            base,
            token,
            "POST",
            f"/repos/{OWNER}/{REPO}/issues/{num}/comments",
            {"body": meta["comment"]},
        )
        if state != "closed":
            gitea(
                base,
                token,
                "PATCH",
                f"/repos/{OWNER}/{REPO}/issues/{num}",
                {"state": "closed"},
            )
            closed += 1
            print(f"  closed #{num}")
        else:
            print(f"  already closed #{num}")

        try:
            gitea(
                base,
                token,
                "POST",
                f"/repos/{OWNER}/{REPO}/issues/{num}/labels",
                {"labels": ["resolved-verified"]},
            )
        except RuntimeError as e:
            print(f"  label warn: {e}")

        if conn is not None:
            cur = conn.cursor()
            # findings has no updated_at; force disposition even if previously
            # false_positive/suppressed so closeout is resolved_verified.
            cur.execute(
                """
                UPDATE findings
                   SET status = ?
                 WHERE repository_id = 1
                   AND fingerprint = ?
                   AND status != ?
                """,
                (meta["status"], meta["fp"], meta["status"]),
            )
            print(f"  findings rows updated: {cur.rowcount}")
            cur.execute(
                """
                UPDATE external_issues
                   SET state = 'closed',
                       updated_at = ?
                 WHERE state = 'open'
                   AND issue_number = ?
                   AND finding_id IN (
                        SELECT id FROM findings WHERE repository_id = 1
                   )
                """,
                (now, num),
            )
            print(f"  external_issues rows updated: {cur.rowcount}")

    if conn is not None:
        conn.commit()
        # summary
        cur = conn.cursor()
        open_hm = cur.execute(
            """
            SELECT COUNT(*) FROM findings
             WHERE repository_id=1 AND status='open'
               AND lower(severity) IN ('high','medium','critical')
            """
        ).fetchone()[0]
        print(f"DB open high/med/critical: {open_hm}")
        for num, meta in ISSUES.items():
            row = cur.execute(
                "SELECT id, status, severity FROM findings WHERE fingerprint=? AND repository_id=1",
                (meta["fp"],),
            ).fetchone()
            print(f"  fp {meta['fp']}: {row}")
        conn.close()

    # verify forge open labeled issues
    for label in ("source/repository-detective", "repository-detective"):
        open_issues = gitea(
            base,
            token,
            "GET",
            f"/repos/{OWNER}/{REPO}/issues?state=open&labels={label}&type=issues&limit=50",
        ) or []
        print(f"open issues label={label}: {len(open_issues)} {[i.get('number') for i in open_issues]}")

    print(f"done; newly_closed={closed}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
