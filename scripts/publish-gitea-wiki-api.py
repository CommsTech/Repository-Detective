#!/usr/bin/env python3
"""Publish docs/wiki/*.md to the Gitea wiki via REST API (works when .wiki.git push returns 500)."""
from __future__ import annotations

import argparse
import base64
import os
import sys
import urllib.error
import urllib.request
from pathlib import Path


def env(*keys: str, default: str = "") -> str:
    for k in keys:
        v = os.environ.get(k, "").strip()
        if v:
            return v
    return default


def api_request(method: str, url: str, token: str, payload: dict | None = None) -> tuple[int, dict | list | str]:
    import json

    data = None
    headers = {
        "Authorization": f"token {token}",
        "Accept": "application/json",
    }
    if payload is not None:
        data = json.dumps(payload).encode()
        headers["Content-Type"] = "application/json"
    req = urllib.request.Request(url, data=data, headers=headers, method=method)
    try:
        with urllib.request.urlopen(req, timeout=60) as resp:
            body = resp.read().decode()
            code = resp.status
    except urllib.error.HTTPError as e:
        body = e.read().decode()
        code = e.code
    try:
        parsed = __import__("json").loads(body) if body else {}
    except Exception:
        parsed = body
    return code, parsed


def main() -> int:
    ap = argparse.ArgumentParser(description=__doc__)
    ap.add_argument("--dry-run", action="store_true")
    args = ap.parse_args()

    root = Path(__file__).resolve().parents[1]
    source = Path(env("WIKI_SOURCE_DIR", default=str(root / "docs" / "wiki")))
    gitea = env("REPOSITORY_DETECTIVE_GITEA_URL", "GITEA_URL", default="https://git.commsnet.org").rstrip("/")
    owner = env("REPOSITORY_DETECTIVE_GITEA_OWNER", default="commstech")
    # Live Gitea canonical name is lowercase.
    repo = env("REPOSITORY_DETECTIVE_GITEA_REPO", default="repository-detective")
    token = env("REPOSITORY_DETECTIVE_GITEA_TOKEN", "BUGBOT_GITEA_TOKEN")
    if not token and not args.dry_run:
        print("REPOSITORY_DETECTIVE_GITEA_TOKEN required", file=sys.stderr)
        return 2

    base = f"{gitea}/api/v1/repos/{owner}/{repo}"
    pages = sorted(source.glob("*.md"))
    if not pages:
        print(f"no markdown in {source}", file=sys.stderr)
        return 1

    # Existing page titles
    existing = set()
    if not args.dry_run:
        code, listed = api_request("GET", f"{base}/wiki/pages", token)
        if code == 200 and isinstance(listed, list):
            existing = {str(p.get("title") or p.get("sub_url") or "") for p in listed}
        else:
            print(f"warn: list pages HTTP {code}: {listed}", file=sys.stderr)

    ok = 0
    fail = 0
    for path in pages:
        title = path.stem  # Home, QUICK_START, …
        content = path.read_text(encoding="utf-8")
        b64 = base64.b64encode(content.encode("utf-8")).decode("ascii")
        payload = {
            "title": title,
            "content_base64": b64,
            "message": f"Sync {title} from docs/wiki",
        }
        if args.dry_run:
            print(f"DRY {title} ({len(content)} bytes)")
            ok += 1
            continue

        if title in existing or title.replace("-", " ") in existing:
            code, resp = api_request("PATCH", f"{base}/wiki/page/{title}", token, payload)
            action = "update"
        else:
            code, resp = api_request("POST", f"{base}/wiki/new", token, payload)
            action = "create"
            if code in (400, 409) and "already exists" in str(resp).lower():
                code, resp = api_request("PATCH", f"{base}/wiki/page/{title}", token, payload)
                action = "update-after-exists"

        if 200 <= code < 300:
            print(f"OK {action} {title} HTTP {code}")
            ok += 1
        else:
            print(f"FAIL {action} {title} HTTP {code}: {resp}", file=sys.stderr)
            fail += 1

    print(f"done ok={ok} fail={fail} wiki=https://git.commsnet.org/{owner}/{repo}/wiki")
    return 0 if fail == 0 else 1


if __name__ == "__main__":
    raise SystemExit(main())
