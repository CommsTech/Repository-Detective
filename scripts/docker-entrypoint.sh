#!/bin/sh
set -eu

if [ -d /app/data ] && [ "$(id -u)" -eq 0 ]; then
  chown -R repositorydetective:repositorydetective /app/data
fi

# Durable scanner temp/cache under the data volume (avoids filling overlay /tmp).
mkdir -p /app/data/tmp /app/data/cache
if [ "$(id -u)" -eq 0 ]; then
  chown -R repositorydetective:repositorydetective /app/data/tmp /app/data/cache 2>/dev/null || true
fi
export TMPDIR="${TMPDIR:-/app/data/tmp}"
export XDG_CACHE_HOME="${XDG_CACHE_HOME:-/app/data/cache}"

# Drop abandoned grype scratch left behind when scans are killed/timeout.
# Also clear legacy overlay /tmp leftovers from older images.
for scratch_root in "$TMPDIR" /tmp; do
  [ -d "$scratch_root" ] || continue
  find "$scratch_root" -maxdepth 1 -type d \( -name 'grype-scratch*' -o -name 'grype-cache*' -o -name 'getter*' \) -exec rm -rf {} + 2>/dev/null || true
done

# Trust operator-provided CA bundles (e.g. self-signed OpenClaw gateway).
if [ -d /app/certs ]; then
  for cert in /app/certs/*.crt /app/certs/*.pem; do
    [ -f "$cert" ] || continue
    cp "$cert" "/usr/local/share/ca-certificates/$(basename "$cert")"
  done
  if ls /usr/local/share/ca-certificates/*.crt >/dev/null 2>&1; then
    update-ca-certificates >/dev/null 2>&1 || true
  fi
fi

if [ "$(id -u)" -eq 0 ]; then
  exec su-exec repositorydetective:repositorydetective "$@"
fi

exec "$@"
