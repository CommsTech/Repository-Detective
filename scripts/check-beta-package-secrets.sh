#!/usr/bin/env bash
# Verify beta package contains no secrets or forbidden files.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
PKG="$ROOT/dist/repository-detective-beta"

if [[ ! -d "$PKG" ]]; then
  echo "ERROR: package not found: $PKG" >&2
  exit 1
fi

fail=0
for f in repository-detective checksums.txt README_BETA.md RELEASE_NOTES.md config.example.yaml docker-compose.beta.yml; do
  if [[ ! -e "$PKG/$f" ]]; then
    echo "FAIL: missing $f"
    fail=1
  fi
done

# Scan only shipped runtime artifacts — not README/docs that mention config paths.
for target in "$PKG/repository-detective" "$PKG/config.example.yaml"; do
  if [[ ! -f "$target" ]]; then
    continue
  fi
  if strings "$target" 2>/dev/null | grep -qE 'sk-live-|ghp_[A-Za-z0-9]{20,}|BEGIN (RSA|OPENSSH) PRIVATE KEY'; then
    echo "FAIL: suspicious secret material in $target"
    fail=1
  fi
done

if [[ -f "$PKG/.env" ]]; then
  echo "FAIL: live .env must not be packaged"
  fail=1
fi
if [[ -f "$PKG/repository-detective.db" ]]; then
  echo "FAIL: database must not be packaged"
  fail=1
fi

if [[ "$fail" -eq 0 ]]; then
  echo "PASS: beta package secrets check"
  if [[ -f "$PKG/sbom-go.cdx.json" || -f "$PKG/sbom.spdx.json" ]]; then
    echo "SBOM: present"
  else
    echo "SBOM: optional (cyclonedx-gomod not installed at build time)"
  fi
  exit 0
fi
exit 1
