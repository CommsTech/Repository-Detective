#!/bin/sh
set -eu

if [ -d /app/data ] && [ "$(id -u)" -eq 0 ]; then
  chown -R repositorydetective:repositorydetective /app/data
fi

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
