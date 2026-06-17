#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/.."
exec python3 scripts/nightly-rd-skill-loop.py --daily-mode --promote
