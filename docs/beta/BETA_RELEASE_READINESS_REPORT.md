# Beta release readiness report

Generated: 2026-06-07 (Beta UX + Release Gate sprint)  
Latest commit: see git log

## Verification summary

| Check | Result |
|-------|--------|
| `go test ./...` | PASS |
| `go vet ./...` | PASS |
| staticcheck | Not confirmed in CI container (IPv6/proxy); local optional |
| gosec | Baseline findings; not a sprint blocker |
| Docker build verify | **PASS** (~23 min, all 3 targets, smoke OK) |
| `make beta-release` | PASS (manual path) |
| Configure capability links | **FIXED** — deep links to `/configure#…` |
| Pre-install audit | **200** when disabled with banner |

## Feature / UX status

| Item | Status |
|------|--------|
| Configure action links | Fixed with anchor sections |
| Remediation PR configure | Shows keys, limits, token present/missing |
| Feature flag matrix | Updated with configure links |
| Ruff gating | Implemented (`profile/ruff.go`) |
| SBOM | Implemented prior sprint; verified in tests |
| Favicon / risk map | Done prior sprint |
| Cursor benchmark | Fixture plan published |

## Product baseline

| Gate | Status |
|------|--------|
| Open issues | 1 (#48) |
| Active-present findings | 0 |
| Limited issue filing | NOT approved |
| All-repo scan | NOT started |

## Recommendation

**Private beta ready** for homelab/internal testers.

Public beta: pending benchmark fixture execution + staticcheck CI gate.

## Remaining blockers

1. Run Cursor Bugbot benchmark fixture and fill metrics table
2. staticcheck in CI
3. Optional: UI editor for config (currently read-only instructions — acceptable for beta)
