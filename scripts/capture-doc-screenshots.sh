#!/usr/bin/env bash
# Capture documentation screenshots from a running Repository Detective UI.
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
OUT="${SCREENSHOT_DIR:-$ROOT/docs/assets/screenshots}"
BASE="${RD_SCREENSHOT_BASE:-http://127.0.0.1:8081/ui}"
API_KEY="${REPOSITORY_DETECTIVE_API_KEY}"

mkdir -p "$OUT"

log() { printf '==> %s\n' "$*"; }

if ! command -v chromium-browser >/dev/null 2>&1 && ! command -v google-chrome >/dev/null 2>&1 && ! command -v chromium >/dev/null 2>&1; then
  log "No headless Chrome found. Install chromium or capture manually."
  log "Save PNGs to: $OUT"
  log "Required files listed in docs/assets/screenshots/README.md"
  exit 0
fi

BROWSER=""
for c in chromium-browser chromium google-chrome; do
  if command -v "$c" >/dev/null 2>&1; then
    BROWSER="$c"
    break
  fi
done

capture() {
  local name="$1"
  local path="$2"
  local url="${BASE}${path}"
  if [[ -n "$API_KEY" && "$url" != *api_key=* ]]; then
    url="${url}?api_key=${API_KEY}"
  fi
  log "capture $name -> $url"
  "$BROWSER" --headless --disable-gpu --window-size=1440,900 \
    --screenshot="$OUT/${name}.png" "$url" 2>/dev/null || \
    warn "failed: $name"
}

capture dashboard "/"
capture repos-list "/repos"
capture repo-detail "/repos/1"
capture configure "/configure"
capture preinstall "/preinstall"
capture learning "/learning"
capture scan-now-modal "/repos/1/scan"

log "Screenshots written to $OUT"
log "Review for secrets and redact repo names before publishing."
