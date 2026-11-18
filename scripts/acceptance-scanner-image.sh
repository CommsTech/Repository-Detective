#!/bin/sh
# Run INSIDE a built runner/all-in-one image during CI/release acceptance.
# Does not download scanner data or execute repository-controlled code.
set -eu
missing=0
for tool in trivy grype syft gitleaks semgrep govulncheck gosec staticcheck cyclonedx-gomod golangci-lint; do
  if ! command -v "$tool" >/dev/null 2>&1; then
    echo "FAIL: scanner binary missing: $tool" >&2
    missing=$((missing + 1))
  else
    echo "OK: $tool present"
  fi
done
if [ "$missing" -gt 0 ]; then
  echo "FAIL: $missing scanner tools missing: no full-coverage release claim" >&2
  exit 1
fi
if ! command -v go >/dev/null 2>&1; then
  echo "FAIL: Go runtime not present for Go analysis scanners" >&2
  exit 1
fi
printf 'Go toolchain: '
go version
# The release pipeline must ALSO execute scanner version commands and a known
# fixture for each tool, and scan this image with refreshed vulnerability DBs.
# Presence is a minimum contract, not proof of coverage or up-to-date databases.
echo "PASS: minimum scanner presence gate (versions and fixtures still required)"
