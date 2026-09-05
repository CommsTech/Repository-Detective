#!/usr/bin/env bash
# Full local verification — mirrors CI plus security checks.
set -euo pipefail
cd "$(dirname "$0")/.."

echo "==> go mod tidy check"
go mod tidy
git diff --exit-code go.mod go.sum

echo "==> gofmt (tracked Go files)"
FILES=$(git ls-files '*.go' | grep -v '^vendor/' || true)
test -n "$FILES"
test -z "$(echo "$FILES" | xargs -n 200 gofmt -s -l)"

echo "==> go vet"
go vet ./...

echo "==> staticcheck (install if missing)"
if ! command -v staticcheck >/dev/null 2>&1; then
  go install honnef.co/go/tools/cmd/staticcheck@latest
fi
export PATH="$(go env GOPATH)/bin:$PATH"
staticcheck $(go list ./... | grep -v /vendor)

echo "==> tests"
go test ./... -count=1

echo "==> build"
go build -ldflags "-s -w" -o bin/repository-detective .

echo "==> govulncheck"
go run golang.org/x/vuln/cmd/govulncheck@latest ./...

echo "OK — all verification steps passed"
