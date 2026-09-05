#!/usr/bin/env bash
# Fail if the published git tree would ship live operator secrets or configs.
# Safe for CI and for scripts/sync-gitea-to-github.sh.
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

fail=0
log() { printf '%s\n' "$*"; }
bad() { printf 'FAIL: %s\n' "$*" >&2; fail=1; }

log "==> public release secret gate"

# 1) Forbidden paths must never be tracked
while IFS= read -r path; do
  [[ -z "$path" ]] && continue
  if git ls-files --error-unmatch "$path" >/dev/null 2>&1; then
    bad "tracked forbidden path: $path"
  fi
done <<'EOF'
.env
config/config.yaml
data/repository-detective.db
EOF

# 2) Working tree must not stage those files
if git diff --cached --name-only | grep -E '(^|/)\.env$|(^|/)config/config\.yaml$|(^|/)data/.*\.db$' >/dev/null; then
  bad "staged forbidden secret/config/db path"
fi

# 3) Live .env values must not appear in tracked content
if [[ -f .env ]]; then
  python3 - <<'PY' || fail=1
from pathlib import Path
import re, subprocess, sys

def load_env(path):
    out = {}
    for line in Path(path).read_text().splitlines():
        line = line.strip()
        if not line or line.startswith("#") or "=" not in line:
            continue
        k, v = line.split("=", 1)
        out[k.strip()] = v.strip().strip('"').strip("'")
    return out

def placeholder(v: str) -> bool:
    if not v or len(v) < 8:
        return True
    low = v.lower()
    return low.startswith(("your-", "change-", "test", "ci-", "verify", "example", "generate", "http://", "https://"))

env = load_env(".env")
keys = [k for k in env if re.search(r"(TOKEN|KEY|SECRET|PASSWORD|DSN)", k, re.I)]
tracked = [t for t in subprocess.check_output(["git", "ls-files"], text=True).splitlines() if not t.startswith("vendor/")]
leaks = []
for k in keys:
    v = env[k]
    if placeholder(v):
        continue
    for f in tracked:
        try:
            text = Path(f).read_text(errors="ignore")
        except Exception:
            continue
        if v in text:
            leaks.append((k, f))
if leaks:
    for k, f in leaks:
        print(f"FAIL: live {k} value appears in tracked file {f}", file=sys.stderr)
    sys.exit(1)
print("OK: no live .env secret values in tracked files")
PY
fi

# 4) Examples must stay placeholders (empty tokens in yaml examples)
for f in config/config.yaml.example config/private-beta.example.yaml config/runner.example.yaml; do
  [[ -f "$f" ]] || continue
  if grep -E '^(gitea_token|github_token|api_key|webhook_secret|session_secret|ai_api_key|runner_shared_secret):[[:space:]]*["'\'']?[A-Za-z0-9_\-]{12,}' "$f" >/dev/null; then
    bad "example config looks like it embeds a real secret: $f"
  fi
done

# 5) .env.example must not contain live-looking tokens
if grep -E '^REPOSITORY_DETECTIVE_(GITEA_TOKEN|GITHUB_TOKEN|API_KEY|WEBHOOK_SECRET|AI_API_KEY)=.{16,}$' .env.example \
  | grep -viE 'your-|change-me|generate-|example|placeholder' >/dev/null; then
  bad ".env.example has a non-placeholder secret-looking value"
fi

if [[ "$fail" -ne 0 ]]; then
  log "public release secret gate FAILED"
  exit 1
fi
log "PASS: public tree ships examples only (no live .env / config.yaml / secrets)"
