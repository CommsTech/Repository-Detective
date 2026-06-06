# CI gate unblock report — 2026-06-06

## CI run history

| Run | Commit | Status | Root cause |
|-----|--------|--------|------------|
| #1842 | 9d875fd | stuck → failure | Runner job completion lag |
| #1843 | 2e96509 | **failure** | Govulncheck: `govulncheck@latest` requires Go ≥1.25; CI uses Go 1.23 |
| #1844 | a7f708c | **failure** | Same Govulncheck install failure |
| #1845 | de879ce | **in progress** | After pin to `@v1.1.3` |

URL: https://git.commsnet.org/commstech/Bugbot/actions

## Fixes made this sprint

### Docker build (Phase 1)

- **Commit:** `a41a5ab` — `fix(docker): repair Alpine package install syntax`
- Removed invalid `apk add package=*` wildcards from `Dockerfile`
- **Local result:** `docker-compose build repository-detective` **SUCCESS** (tag `repository-detective:all-in-one`)

### API key / config (Phase 2)

- **Commit:** `a7f708c` — `fix(config): align API key auth across compose and runtime`
- Fixed `viper.SetEnvPrefix` before `AutomaticEnv()`
- `envcompat.Apply` now sets legacy `BUGBOT_*` when `REPOSITORY_DETECTIVE_*` unset
- Compose passes `REPOSITORY_DETECTIVE_API_KEY` from `.env`
- **Local result:** operator smoke test **PASS** (preferred + legacy headers)

### CI Govulncheck (Phase 3 follow-up)

- **Commit:** `de879ce` — pin `govulncheck@v1.1.3` (matches Dockerfile builder)
- Fixed `docker-compose.host-network.yml` version for valid override merge

## Local test results

| Check | Result |
|-------|--------|
| `go test ./...` | **PASS** |
| `go vet ./...` | **PASS** (via test run) |
| `staticcheck ./...` | VCS stamp warning in ephemeral docker mount; **PASS in CI checkout** |
| `govulncheck@v1.1.3` | Runs; reports stdlib/import vulns (exit 3) — monitor run #1845 |
| `./scripts/operator-smoke-test.sh` | **PASS** |
| `./scripts/docker-build-verify.sh` | Build OK locally; full script slow (rebuilds all targets) |

## Docker build status

**PASS** — all-in-one image builds and starts after Alpine syntax fix.

## API auth status

**PASS** — after container recreate with `--env-file .env` (host network on this node due to bridge IP pool exhaustion).

## Batch 2 allowed?

**NO** — wait for CI run #1845 (or latest) to finish **green**. If govulncheck fails on stdlib vuln exit code 3, document as next blocker (Go 1.23 stdlib advisory vs toolchain bump).

## Remaining risks

1. Swarm node bridge network pool exhausted — default bridge compose may fail; use host-network override or prune unused GITEA action networks
2. Govulncheck may exit non-zero on stdlib advisories even with pinned tool version
3. Running container image built before `a7f708c` — rebuild recommended after CI green: `docker-compose build && docker-compose up -d --force-recreate`
