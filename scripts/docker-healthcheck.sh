#!/bin/sh
# Uses REPOSITORY_DETECTIVE_PORT for /health probes.
set -eu
port="${REPOSITORY_DETECTIVE_PORT:-8080}"
wget -q -O /dev/null "http://127.0.0.1:${port}/health" || exit 1
