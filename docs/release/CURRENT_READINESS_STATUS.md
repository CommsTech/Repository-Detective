# Current readiness status

**Reconciled:** 2026-06-12 (first external tester complete)  
**Source commit:** `f3dcb9a` (pending post-tester commits)  
**Live revision:** `rc-381667a` (deployed 2026-06-12)

## Readiness decisions

| Level | Status | Notes |
|-------|--------|-------|
| **Marketing ready** | **NO** | Wiki blocked; external VM install; need ≥2 external testers |
| **Private beta ready** | **YES** | First external tester complete; calibration before #2 |
| **Controlled demo ready** | **YES** | |
| **External tester #1** | **COMPLETE** | `ext-operator-jrice` / `Wifi_Collector` scan `85a8ab62e76da076` |

## Product dogfood (live DB)

| Metric | Value |
|--------|-------|
| Active-present open (product repo) | **0** |
| High/critical active (product) | **0** |
| Gitea open mapped issues (product) | **0** |

## First external tester (2026-06-12)

| Item | Status |
|------|--------|
| Tester assigned | `ext-operator-jrice` |
| Repo | `commstech/Wifi_Collector` |
| Scan | `85a8ab62e76da076` report-only |
| Issues / PRs created | **0 / 0** |
| Feedback | Received via Gitea templates |
| Outcome | Safe scan; **calibration sprint before tester #2** |

## Stabilization (prior)

| Item | Status |
|------|--------|
| Full `go test ./...` | **PASS** |
| Structured issue body live | **PASS** (`rc-381667a`) |
| Gitea issue templates | **15** via API |
| Internal rehearsal | `operator-cohort-1` / PCAP `512145e55d4488ea` |

## Remaining blockers (marketing)

1. Gitea wiki HTTP 500
2. External VM clean install proof
3. At least **2** clean external tester cycles (1 of 2 complete)
4. Optional logged-in template picker screenshot
5. False-positive calibration from tester #1 before broader expansion

## Next step

**Focused calibration/docs sprint** (hardcoded-secret heuristic, graph noise, SBOM syft messaging), then onboard **external tester #2**.

## Do not repeat

- All-repo scanning
- Issue filing on beta tester repos
- Marketing launch without wiki + VM proof + 2+ external testers
