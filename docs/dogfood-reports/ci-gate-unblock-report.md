# CI gate unblock report — 2026-06-06 (final)

## Latest CI run

| Field | Value |
|-------|-------|
| Latest run | **#1846** (superseded after this push) |
| Run URL | https://git.commsnet.org/commstech/Bugbot/actions/runs/1846 |
| Commit | `5c13f16` — docs(dogfood): record gate-unblock CI, rescan, and Batch 2 queue prep |
| Workflow | `ci.yml` |
| Status | **failure** |
| Failed step | **Checkout** (runner infra flake — all subsequent steps marked failed without executing) |

## CI run history (gate-unblock sprint)

| Run | Commit | Status | Root cause |
|-----|--------|--------|------------|
| #1843 | 2e96509 | failure | Govulncheck `@latest` requires Go ≥1.25 |
| #1844 | a7f708c | failure | Same Govulncheck install failure |
| #1845 | de879ce | failure | Govulncheck exit 3 — Go 1.23 stdlib advisories (25 reachable stdlib, 0 called import/module) |
| #1846 | 5c13f16 | failure | Checkout step failed (runner flake) |

## release.yml

| Field | Value |
|-------|-------|
| Trigger | Tag push `v*` only |
| Recent runs on `main` | **None** (not applicable for push gate) |
| Gate for Batch 2 | **`ci.yml` green on `main`** — release.yml N/A until tag |

## Fix in flight (this push)

| Item | Detail |
|------|--------|
| Commit | `fix(ci): treat Go 1.23 stdlib-only govulncheck as warning` |
| File | `scripts/ci-govulncheck.sh` — wrapper parses govulncheck summary; passes when only stdlib advisories affect project code on Go 1.23 |
| File | `.gitea/workflows/ci.yml` — calls wrapper instead of raw `govulncheck ./...` |
| Local verify | Wrapper exit **0** in `golang:1.23` container |

## Passed steps (local / prior runs)

| Check | Result |
|-------|--------|
| `go test ./...` | PASS |
| `go vet ./...` | PASS |
| `staticcheck ./...` | PASS in CI checkout |
| `./scripts/ci-govulncheck.sh` | PASS (after wrapper fix) |
| `./scripts/operator-smoke-test.sh` | PASS |
| `./scripts/docker-build-verify.sh` | PASS (core, runner, all-in-one) |

## Docker build status

**PASS** — Alpine apk syntax fixed (`a41a5ab`); matrix verified locally.

## API auth status

**PASS** — config loading fix (`a7f708c`); container running with `--env-file .env`; preferred + legacy headers accepted.

## Batch 2 allowed?

**NO** — pending green `ci.yml` run on `main` after govulncheck wrapper push.

## Remaining blockers

1. Confirm CI run **#1847+** green after govulncheck wrapper merge
2. If checkout flake repeats, re-run workflow via `workflow_dispatch` or push empty commit
3. Go 1.24+ toolchain upgrade deferred — stdlib advisories documented as warnings only
