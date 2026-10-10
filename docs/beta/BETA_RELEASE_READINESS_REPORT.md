# Beta release readiness report

Generated: 2026-06-02 (Post-learning beta gate sprint)

## Verification summary

| Check | Result |
|-------|--------|
| `go test ./...` | PASS (Docker golang:1.23-bookworm) |
| `go vet ./...` | PASS |
| staticcheck | PASS locally with `-buildvcs=false`; CI pinned v0.6.1 |
| `make beta-release` | **PASS** (user-safe staging + clean-beta-release) |
| Beta secrets check | **PASS** (`scripts/check-beta-package-secrets.sh`) |
| Docker build verify | Prior sprint PASS; re-run after merge recommended |
| Learning engine validation | PASS — see learning-engine-validation-report.md |
| Benchmark fixture | PASS — see CURSOR_BUGBOT_BENCHMARK_RESULTS.md |
| Calibration operator review | PASS — 3 repo-scoped accepts, 0 global |

## Gate status

| Gate | Status |
|------|--------|
| Open issues (product) | 1 (#48) |
| Active-present findings | 0 |
| Non-product issue filing | Disabled |
| All-repo scan | Not started |
| LLM sanity gate | Disabled by default |
| Report-only dry-run | Available |

## Packaging

| Item | Status |
|------|--------|
| `make clean-beta-release` | Removes root-owned dist via Docker fallback |
| `make beta-release` | Builds as current user; no secrets in package |
| SBOM in package | Optional when `cyclonedx-gomod` installed at build time |

## Remaining blockers

1. Rebuild live `repository-detective` container for learning API on deployed instance
2. Optional: mirror benchmark fixture to GitHub for Cursor Repository-Detective side-by-side run
3. `data/` directory owned by container user — document Docker-based operator DB updates

## Recommendation

**Private beta ready** — packaging fixed, staticcheck validated, benchmark fixture complete, calibration reviewed safely.

**Public beta:** pending live container rebuild + optional Cursor Repository-Detective mirror benchmark.

Not ready for: unlimited issue filing, all-repo scan, global auto-calibration.
