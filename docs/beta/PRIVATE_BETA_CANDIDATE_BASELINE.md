# Private beta candidate baseline

Generated: 2026-06-07

## Latest commit

`fd59e2e` — docs(beta): publish private beta go-no-go

## Product repo state

| Gate | Status |
|------|--------|
| Open issues (Gitea) | 1 (#48 operator task) |
| Active-present findings | 0 (no open external issue fingerprint in latest scan) |
| Non-product issue filing | Disabled |
| All-repo scan | Not started |
| LLM sanity gate | Disabled by default |
| Report-only dry-run | Available |

## Beta package status

| Item | Status |
|------|--------|
| `make beta-release` | PASS (commit `6a2cbfd`) |
| `make clean-beta-release` | PASS |
| Secrets check script | PASS |
| SBOM in package | Optional (cyclonedx-gomod not installed at build) |
| Binary ownership | commstech (not root) |

## Staticcheck status

| Item | Status |
|------|--------|
| Local/container | PASS with `GOFLAGS=-buildvcs=false` after cmd tool fixes |
| CI run #128 (`6a2cbfd`) | **Failed** — invalid `-buildvcs=false` flag passed to staticcheck (not a staticcheck flag) |
| Fix pending | `GOFLAGS=-buildvcs=false` on Staticcheck step; remove invalid flag |

## Benchmark fixture status

| Item | Status |
|------|--------|
| Layout | `.src` inject files + `benchmark/fixture_benchmark_test.go` |
| `go test ./benchmark/...` | PASS |

## Remaining blockers (pre-final verification)

1. ~~CI staticcheck step fix~~ — fixed in `572635b`; await green CI run #129
2. ~~staticcheck S1040 in `cmd/rd-*` tools~~ — fixed in `572635b`
3. Rebuild live container for learning-event emission on dry-runs
4. Optional: full `docker-build-verify.sh` re-run
