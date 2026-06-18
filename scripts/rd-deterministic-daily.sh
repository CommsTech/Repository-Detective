#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/.."
mkdir -p reports/nightly-rd-evolution
exec python3 scripts/nightly-rd-skill-loop.py --daily-mode --promote --max-tier 1
