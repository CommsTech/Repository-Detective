# Private beta release baseline

Generated: 2026-06-02

## Latest commit

`dedd0c6` — docs(beta): update private beta candidate baseline after verification

Includes staticcheck CI fix `572635b` and verification docs through `dedd0c6`.

## Product repo state

| Gate | Status |
|------|--------|
| Open issues (Gitea) | 1 (#48 operator task) |
| Active-present findings | **0** (no open external issue fingerprint in latest scan) |
| Non-product issue filing | Disabled by default |
| All-repo scan | Not started |
| LLM sanity gate | Disabled by default |
| Report-only dry-run | Available (`report_only_dry_run: true`) |

## Staticcheck status

| Item | Status |
|------|--------|
| CI fix | `572635b` — `GOFLAGS=-buildvcs=false` on Staticcheck step |
| Local/container | PASS with `GOFLAGS=-buildvcs=false` |
| S1040 in `cmd/rd-*` | Fixed in `572635b` |

## Benchmark fixture status

| Item | Status |
|------|--------|
| Layout | `.src` inject files + `benchmark/fixture_benchmark_test.go` |
| Compile noise | Fixed (no injected `.go` in fixture tree) |
| `go test ./benchmark/...` | PASS |

## Report-only validation status

Latest product scan: `1c4db8a1a7ed8d1e`

| Metric | Value |
|--------|-------|
| Findings persisted | 1146 |
| `issue_sync_status` | skipped |
| Issues created | 0 |
| Open issues before/after | 1 |

Non-product dry-runs (netmapper, commsnet_optimizer): 0 issues, 0 PRs each.

Evidence: [../dogfood-reports/private-beta-report-only-validation.md](../dogfood-reports/private-beta-report-only-validation.md)

## Git hygiene (pre-release)

| Check | Status |
|-------|--------|
| `.env` staged | No |
| Local `repository-detective` ELF staged | No |
| `dist/` artifacts staged | No (gitignored) |
| Working tree | Clean at baseline |

## Remaining known limitations

1. SBOM optional — `cyclonedx-gomod` not installed at build time unless operator adds it
2. Live container may run pre-learning image until rebuild (`docker compose up -d --build`)
3. Full `docker-build-verify.sh` (~23 min) not re-run every sprint
4. Some scanners time out on large repos in container (staticcheck, hadolint, checkov, ruff)
5. Global calibration auto-accept blocked in beta (repo-scoped only)
6. Python/Ruff gating validation on non-Go repos recommended before limited issue filing

## Private beta safety posture (unchanged)

- Report-only first
- Issue filing off by default (`auto_create_issues: false`)
- Remediation PR off by default
- Backlog-control on
- Evidence closure on
- Scanner transparency on
- Learning recommendations visible; no auto-global calibration
