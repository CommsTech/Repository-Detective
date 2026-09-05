# Private beta release go / no-go

Date: 2026-06-02  
Baseline commit: `b4f1a60` (packaging sprint)

## Decision

| Level | Recommendation |
|-------|----------------|
| No-go | No |
| Internal beta only | Superseded |
| **Private beta ready** | **YES** |
| Public beta ready | **NO** |

**Recommendation: private beta ready, public beta not ready.**

## Product repo status

| Item | Status |
|------|--------|
| Open issues | 1 (#48 operator task) |
| Active-present findings | **0** |
| Latest validation scan | `1c4db8a1a7ed8d1e` |
| Issues created during validation | 0 |

## CI / staticcheck

| Item | Status |
|------|--------|
| Fix commit | `572635b` |
| Local staticcheck | PASS (`GOFLAGS=-buildvcs=false`) |
| CI green on latest push | Pending confirmation after packaging commits |

## Docker

| Item | Status |
|------|--------|
| `docker-compose.beta.yml` | Shipped with safe env overrides |
| Full `docker-build-verify.sh` | Not re-run this sprint (~23 min) |
| Live operator container | Running; **rebuild required** for learning/configure UI |

## Beta package

| Item | Status |
|------|--------|
| `make beta-release` | PASS |
| `config/private-beta.example.yaml` | Shipped |
| Checksums | PASS |
| Secrets check | PASS |
| SBOM in bundle | Optional (tool not installed) |

## Safety controls

| Control | Status |
|---------|--------|
| Report-only dry-run | Verified |
| Non-product issue filing | Disabled by default |
| All-repo scan | Not started |
| LLM sanity gate | Disabled by default |
| Remediation PR | Disabled by default |
| Backlog control | Enabled in beta config |
| Evidence closure | Enabled |
| Learning engine | Implemented; UI needs image rebuild |

## Validation evidence

- [PRIVATE_BETA_RELEASE_BASELINE.md](PRIVATE_BETA_RELEASE_BASELINE.md)
- [PRIVATE_BETA_PACKAGE_VERIFICATION.md](PRIVATE_BETA_PACKAGE_VERIFICATION.md)
- [PRIVATE_BETA_SMOKE_TEST_REPORT.md](PRIVATE_BETA_SMOKE_TEST_REPORT.md)
- [PRIVATE_BETA_TESTER_GUIDE.md](PRIVATE_BETA_TESTER_GUIDE.md)
- [PRIVATE_BETA_OPERATOR_RUNBOOK.md](PRIVATE_BETA_OPERATOR_RUNBOOK.md)
- [../dogfood-reports/private-beta-report-only-validation.md](../dogfood-reports/private-beta-report-only-validation.md)

## Known limitations

1. SBOM optional in beta bundle unless cyclonedx-gomod installed at build
2. Live homelab container may lag main — distribute fresh bundle + rebuild instructions
3. Scanner timeouts on large monorepos
4. Global calibration auto-accept blocked (by design)
5. Python repo Ruff gating needs broader dry-run before limited filing

## Operator approval requirements

Before expanding beyond report-only:

- [ ] Operator reviews feedback from first tester cohort
- [ ] Container rebuilt from packaging sprint commit
- [ ] CI green on `main`
- [ ] Explicit written approval for per-repo issue filing
- [ ] Active-present remains 0 on product repo

## Rollback plan

1. Stop service / remove beta container
2. Restore previous binary or image tag
3. Restore `data/repository-detective.db` backup if schema migration issues
4. Revoke tester API keys and forge tokens if compromised
5. Re-distribute prior bundle checksum if bad build detected

See [PRIVATE_BETA_OPERATOR_RUNBOOK.md](PRIVATE_BETA_OPERATOR_RUNBOOK.md).

## Remaining blockers (public beta only)

1. CI green confirmation on latest `main`
2. Cyclonedx SBOM in release bundle (recommended)
3. Full docker-build-verify re-run
4. Support/docs polish and broader platform matrix
5. Learning UI verified on rebuilt production image

## Next recommended batch

1. Distribute `dist/repository-detective-beta/` to first testers with [PRIVATE_BETA_TESTER_GUIDE.md](PRIVATE_BETA_TESTER_GUIDE.md)
2. Rebuild operator homelab image from `b4f1a60+`
3. Collect feedback via [PRIVATE_BETA_FEEDBACK_TEMPLATE.md](PRIVATE_BETA_FEEDBACK_TEMPLATE.md)
4. Optional: Python repo report-only dry-run for Ruff validation
5. After cohort feedback: gated decision on limited issue filing for one non-product repo
