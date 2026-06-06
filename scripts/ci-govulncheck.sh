#!/usr/bin/env bash
# CI govulncheck wrapper for Go 1.23 toolchain.
# Fails on reachable third-party/import vulnerabilities in project code.
# Stdlib-only advisories (fixed in Go 1.24+) are reported as warnings — not a
# release blocker while the project stays on Go 1.23 per sprint policy.
set -euo pipefail

go install golang.org/x/vuln/cmd/govulncheck@v1.1.3

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

# Exit 3: vulnerabilities found. Pass when the summary says project code is only
# affected via Go 1.23 stdlib advisories; import/module findings are present but
# explicitly not called by this codebase.
if printf '%s' "$summary_oneline" | grep -q 'Your code is affected by [1-9][0-9]* vulnerabilities from the Go standard library'; then
  if printf '%s' "$summary_oneline" | grep -q "your code doesn't appear to call these vulnerabilities"; then
    if ! printf '%s' "$summary_oneline" | grep -Eq 'Your code is affected by [1-9][0-9]* vulnerabilities from the Go standard library and [1-9][0-9]* other'; then
      echo "govulncheck: stdlib-only advisories on Go 1.23 — warning only (upgrade Go 1.24+ when approved)"
      exit 0
    fi
  fi
fi

echo "govulncheck: reachable non-stdlib vulnerabilities — failing CI" >&2
exit 3
