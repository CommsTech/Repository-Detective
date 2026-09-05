# Private beta smoke test report

Date: 2026-06-02  
Target: Live operator instance (`127.0.0.1:8081`) + local unit tests  
Beta bundle: `dist/repository-detective-beta/` (commit `b4f1a60`)

## Summary

| Area | Result | Notes |
|------|--------|-------|
| API / health | **PASS** | Operator smoke test completed |
| UI (live container) | **PARTIAL** | Dashboard unlock works; `/ui/configure`, `/ui/learning`, static assets 404 — container image predates routes |
| UI (unit tests) | **PASS** | Configure, preinstall, favicon tests pass on current main |
| Private beta config tests | **PASS** | `TestPrivateBetaExample*` |
| Report-only validation | **PASS** (prior) | Scan `1c4db8a1a7ed8d1e`, 0 issues |
| Beta package build | **PASS** | `make beta-release` |

**Overall smoke: PASS with operator rebuild recommended for full UI parity.**

## API checks (live)

```bash
RD_BASE_URL=http://127.0.0.1:8081 ./scripts/operator-smoke-test.sh
```

| Check | Result |
|-------|--------|
| `GET /health` | healthy |
| `database_healthy` | true |
| Scanners available | 10 / 10 configured |
| `GET /api/v1/about` | PASS — product name, compatibility |
| `GET /api/v1/status` | PASS — no secret leakage |
| `GET /api/v1/dashboard/summary` | PASS |
| Legacy `X-Repository-Detective-API-Key` header | accepted |

## Feature flags (from `/health` + `/api/v1/status`)

| Flag | Value |
|------|-------|
| `remediation_pr_enabled` | false |
| `runner_delegation_enabled` | false |
| `notifications_enabled` | false |
| `evidence_closure_enabled` | true |
| `preinstall_audit_enabled` | false (live config; beta example enables true) |

## UI checks

| Route | Live | Unit test (main) |
|-------|------|------------------|
| `/ui` | 200 (unlock page) | PASS |
| `/ui/configure` | 404 | PASS |
| `/ui/preinstall` | 404 | PASS |
| `/ui/learning` | 404 | PASS |
| `/ui/static/favicon.svg` | 404 | PASS |
| Executive report | 404 (repo route) | PASS (`TestRepoReport`) |
| Risk map / graph | 404 | PASS |

**Action:** Rebuild live container from `b4f1a60+` for tester UI parity:

```bash
docker compose up -d --build
```

## Report-only scan (prior validation)

Scan `1c4db8a1a7ed8d1e` on `commstech/Repository-Detective`:

| Field | Value |
|-------|-------|
| `dry_run_report_only` | true |
| Findings | 1146 |
| Issues created | 0 |
| Open issues delta | 0 |

## Issue creation

| Check | Result |
|-------|--------|
| Report-only scans | 0 issues |
| Non-product dry-runs | 0 issues (see validation report) |

## SBOM

| Check | Result |
|-------|--------|
| Beta bundle SBOM file | Not generated (cyclonedx-gomod absent) |
| Scan-level SBOM metadata | Available in engine on current main; verify after container rebuild |

## Learning health

| Check | Result |
|-------|--------|
| `/ui/learning` live | 404 (old image) |
| Learning events in DB | 16 (prior operator review) |
| Unit/integration on main | PASS |

## Private beta config validation

```bash
go test -run TestPrivateBetaExample -count=1 .
```

Result: **PASS** — issue filing off, remediation PR off, LLM gate off, no embedded secrets.

## Beta artifact smoke

```bash
make beta-release
./scripts/check-beta-package-secrets.sh
sha256sum -c dist/repository-detective-beta/checksums.txt
```

Result: **PASS**

## Blockers for testers

1. Distribute rebuilt Docker image or binary from current main — not the stale live container alone
2. Optional: install `cyclonedx-gomod` for release SBOM

## Not run this session

- Full `docker-build-verify.sh` (~23 min) — deferred; prior sprint PASS documented
- New live report-only scan trigger — prior validation scan used
