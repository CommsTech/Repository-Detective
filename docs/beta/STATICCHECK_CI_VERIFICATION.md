# Staticcheck CI verification

Date: 2026-06-07

## CI history (main)

| Run | Commit | Result | Root cause |
|-----|--------|--------|------------|
| #128 | `6a2cbfd` | **failure** | `staticcheck -buildvcs=false` — invalid flag (staticcheck does not accept `-buildvcs`) |
| #127 | `f44dde0` | failure | Pre-fix benchmark fixture compile noise |
| #126 | `cec037a` | failure | Earlier CI issues |

## Fix applied

`.gitea/workflows/ci.yml`:

```yaml
- name: Staticcheck
  env:
    GOFLAGS: -buildvcs=false
  run: staticcheck ./...
```

`GOFLAGS=-buildvcs=false` disables VCS stamping during analysis (fixes Docker/non-git sandbox errors). The flag must **not** be passed directly to the `staticcheck` binary.

## Code fixes

| File | Issue | Fix |
|------|-------|-----|
| `cmd/rd-migrate/main.go` | S1040 redundant type assertion | Use `store.Open` → `QueryStore` directly |
| `cmd/rd-calibration-recompute/main.go` | S1040 redundant type assertion | Same |

## Local/container verification

```bash
docker run --rm -v "$PWD:/src" -w /src -e GOFLAGS=-buildvcs=false golang:1.23-bookworm \
  sh -c 'GOBIN=/tmp/bin go install honnef.co/go/tools/cmd/staticcheck@v0.6.1 && staticcheck ./...'
```

Result: **PASS** (after cmd fixes)

## CI pin

- staticcheck: `@v0.6.1` (matches workflow)
- Go: `1.23`

## Recommendation

Re-run CI on next push; staticcheck step should pass if no new findings are introduced.
