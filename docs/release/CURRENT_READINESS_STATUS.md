# Current readiness status

**Reconciled:** 2026-06-12 (external beta stabilization)  
**Source commit:** `d5548f1`  
**Live revision:** `rc-381667a` (deployed 2026-06-12)

## Readiness decisions

| Level | Status | Notes |
|-------|--------|-------|
| **Marketing ready** | **NO** | Wiki blocked; external VM install pending |
| **Private beta ready** | **YES** | Full `go test` pass; filing proof; structured body live |
| **Controlled demo ready** | **YES** | |
| **External tester #1** | **READY TO ONBOARD** | Handoff packet prepared; report-only |

## Product dogfood (live DB)

| Metric | Value |
|--------|-------|
| Active-present open | **0** |
| High/critical active | **0** |
| Gitea open mapped issues (product) | **0** |

## Stabilization sprint (2026-06-12)

| Item | Status |
|------|--------|
| SBOM test failure | **fixed** — manifest before syft |
| Full `go test ./...` | **PASS** |
| Live deploy structured issue body | **PASS** (`rc-381667a`) |
| Gitea filing proof | **PASS** (prior scratch test) |
| Issue templates | **15** via API |
| External tester handoff | **docs/beta/external-tester-1-handoff.md** |

## Remaining blockers (marketing)

1. Gitea wiki HTTP 500
2. External VM clean install proof
3. Named external tester feedback (handoff ready)
4. Optional logged-in template picker screenshot

## Next step

Onboard **first named external tester** — one repo, report-only, Gitea templates for feedback.

## Do not repeat

- All-repo scanning
- Issue filing on beta tester repos
- Marketing launch without wiki + VM proof
