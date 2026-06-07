# Private beta go / no-go

Date: 2026-06-07

## Product repo status

| Item | Status |
|------|--------|
| Open issues | 1 (#48 operator task) |
| Active-present findings | **0** |
| Non-product issue filing | Disabled |
| All-repo scan | Not started |

## CI / staticcheck

| Item | Status |
|------|--------|
| CI run #128 (`6a2cbfd`) | Failed — invalid staticcheck flag (fixed in working tree) |
| Local staticcheck | PASS with `GOFLAGS=-buildvcs=false` |
| `go test ./...` | PASS |

## Docker build

| Item | Status |
|------|--------|
| Full `docker-build-verify.sh` | Not re-run this verification (prior sprint PASS ~23 min) |
| Operator smoke test | PASS |
| Live container | Running; rebuild recommended for learning API |

## Beta package

| Item | Status |
|------|--------|
| `make beta-release` | PASS |
| Secrets check | PASS |
| SBOM in package | Optional (tool not installed at build) |

## Feature flags & UX

| Item | Status |
|------|--------|
| LLM sanity gate | Disabled by default |
| Report-only dry-run | Verified |
| Backlog-control | Active |
| Configure / pre-install pages | Shipped prior sprints |
| Learning engine | Implemented; UI needs container rebuild |

## Calibration policy

| Item | Status |
|------|--------|
| Repo-scoped accepts | 3 rules (90-day expiry) |
| Global accepts | 0 |
| HIGH/CRITICAL auto-downgrade | Blocked |

## Validation evidence

- [BETA_PACKAGE_VERIFICATION.md](BETA_PACKAGE_VERIFICATION.md)
- [STATICCHECK_CI_VERIFICATION.md](STATICCHECK_CI_VERIFICATION.md)
- [CURSOR_BUGBOT_BENCHMARK_RESULTS.md](CURSOR_BUGBOT_BENCHMARK_RESULTS.md)
- [../dogfood-reports/private-beta-report-only-validation.md](../dogfood-reports/private-beta-report-only-validation.md)

## Unsupported for beta

- All-repo scan
- Non-product issue filing
- Global auto-calibration
- Mandatory LLM
- Public anonymous access

## Operator instructions

1. `make beta-release` → distribute `dist/repository-detective-beta/`
2. Copy `config.example.yaml`; set secrets via env (never commit)
3. Use `report_only_dry_run: true` for new repos until approved
4. Review calibration at `/ui/learning`
5. Rebuild container: `docker compose up -d --build`

## Recommendation

| Level | Decision |
|-------|----------|
| **Private beta** | **READY** — product clean, package builds, report-only validated, benchmark documented |
| **Public beta** | **NOT READY** — CI must go green on staticcheck fix, container rebuild, optional SBOM in package, support/docs polish |
| **No-go** | No |

## Remaining before public beta

1. Push staticcheck CI fix; confirm green CI run
2. Rebuild all-in-one image with learning engine
3. Optional: cyclonedx SBOM in beta package
4. Optional: Python repo dry-run for Ruff gating validation
