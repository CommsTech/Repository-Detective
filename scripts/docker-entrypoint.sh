#!/bin/sh
set -eu

# Durable scanner temp/cache under the data volume (avoids filling overlay /tmp).
# Use sticky world-writable TMPDIR so host-side `go test ./...` can traverse the
# bind mount even when the process user inside the container differs from the host uid.
mkdir -p /app/data/tmp /app/data/cache
chmod 1777 /app/data/tmp 2>/dev/null || true

if [ -d /app/data ] && [ "$(id -u)" -eq 0 ]; then
  # Own the DB/cache for the runtime user, but do not recursively re-own TMPDIR
  # scratch (that breaks host tooling walking the repo tree).
  chown repositorydetective:repositorydetective /app/data 2>/dev/null || true
  chown -R repositorydetective:repositorydetective /app/data/cache 2>/dev/null || true
  # Keep scanner caches traversable for host-side `go test ./...` on the bind mount.
  chmod -R a+rX /app/data/cache 2>/dev/null || true
  if [ -e /app/data/repository-detective.db ]; then
    chown repositorydetective:repositorydetective /app/data/repository-detective.db 2>/dev/null || true
  fi
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
  if ls /usr/local/share/ca-certificates/*.crt >/dev/null 2>/dev/null; then
    update-ca-certificates >/dev/null 2>&1 || true
  fi
fi

if [ "$(id -u)" -eq 0 ]; then
  exec su-exec repositorydetective:repositorydetective "$@"
fi

exec "$@"
