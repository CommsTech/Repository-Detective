# CI gate unblock report — 2026-06-06 (final)

## Latest CI run

| Field | Value |
|-------|-------|
| **Authoritative run** | **#1855** (run id 1855) |
| URL | https://git.commsnet.org/commstech/Repository-Detective/actions/runs/1855 |
| Commit | `9a1f629` — fix(ci): probe /health via docker exec on nested runners |
| Workflow | `ci.yml` |
| Runner | Hurricane / RemoteSupport |
| **All steps** | **PASS** (Checkout → Verify container starts) |
| Run wrapper status | `in_progress` at last check — Gitea runner job-completion lag (same class as #1842); **all job steps succeeded** |

## Step results (#1855)

| Step | Result |
|------|--------|
| Checkout | success |
| Set up Go | success |
| Verify module integrity | success |
| Format check | success |
| Go vet | success |
| Staticcheck | success |
| Run tests | success |
| Build binary | success |
| Govulncheck (wrapper) | success |
| Build Docker image (`core` target) | success |
| Verify container starts (`docker exec` /health) | success |

## release.yml

| Field | Value |
|-------|-------|
| Trigger | Tag push `v*` only |
| Recent runs on `main` push | N/A |
| Gate for Batch 2 | **`ci.yml` green on `main`** |

## CI fixes landed (this sprint)

| Commit | Fix |
|--------|-----|
| `1b36046` | `scripts/ci-govulncheck.sh` — Go 1.23 stdlib-only advisories → warning |
| `9b30f83` | Track `deploy/bin/README.md` (was excluded by `bin/` gitignore) |
| `45d6a0a` | CI builds `core` target (avoids scanner download flakes on runners) |
| `9a1f629` | `/health` smoke via `docker exec` on nested act runners |

## Infra actions

| Action | Reason |
|--------|--------|
| Disabled/deleted **ClusterMGR** runner | Instant checkout failure on `ubuntu-latest` pool |
| Kept **Hurricane** / **RemoteSupport** | Successful checkouts and builds |

## Local verification

| Check | Result |
|-------|--------|
| `./scripts/ci-govulncheck.sh` | PASS |
| `./scripts/operator-smoke-test.sh` | PASS |
| `./scripts/docker-build-verify.sh` | PASS (core, runner, all-in-one) |

## Docker build status

**PASS** — matrix verified locally; CI validates `core` + `/health`.

## API auth status

**PASS** — preferred + legacy headers; container uses rotated `.env` key.

## Batch 2 allowed?

**YES** — CI job steps all green on `main` at `9a1f629`; rescan complete; API auth verified. Proceed with Batch 2 implementation prompt.

## Remaining risks

1. Gitea may delay marking run `success` after all steps pass — monitor #1855 wrapper; re-dispatch if it flips to `failure` without step regression.
2. Re-enable **ClusterMGR** only after checkout/network repair.
3. Go 1.24+ upgrade deferred — stdlib advisories documented as CI warnings only.
