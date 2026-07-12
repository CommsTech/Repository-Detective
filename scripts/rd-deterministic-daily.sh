#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/.."
mkdir -p reports/nightly-rd-evolution
# Prefer host Go when available (faster + survives Docker daemon outages).
if [[ -z "${GO:-}" ]]; then
  for candidate in \
    "${HOME}/.local/go/bin/go" \
    /usr/local/go/bin/go \
    "$(command -v go 2>/dev/null || true)"; do
    if [[ -n "${candidate}" && -x "${candidate}" ]]; then
      export GO="${candidate}"
      break
    fi
  done
fi
exec python3 scripts/nightly-rd-skill-loop.py --daily-mode --promote --max-tier 1
