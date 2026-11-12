#!/usr/bin/env bash
# CI govulncheck wrapper for the Go 1.26 toolchain.
# Fails when govulncheck finds a call path in this module, including the
# standard library. Findings that exist only in required modules, and that
# this code does not call, are warnings.
set -euo pipefail

go install golang.org/x/vuln/cmd/govulncheck@v1.8.0

set +e
output=$(govulncheck ./... 2>&1)
code=$?
set -e

printf '%s\n' "$output"
summary_oneline=$(printf '%s' "$output" | tr '\n' ' ')

if [ "$code" -eq 0 ]; then
  echo "govulncheck: no vulnerabilities reported"
  exit 0
fi

if [ "$code" -ne 3 ]; then
  echo "govulncheck: tool error (exit $code)" >&2
  exit "$code"
fi

# "Your code is affected by N" is the reachable result. Match singular and
# plural. "affected by 0" is not a failure.
if printf '%s' "$summary_oneline" | grep -Eq 'Your code is affected by [1-9][0-9]* vulnerabilit(y|ies)'; then
  echo "govulncheck: reachable vulnerabilities — failing CI" >&2
  exit 3
fi

echo "govulncheck: findings are not reachable from project code — warning only"
exit 0
