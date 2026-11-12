# Current readiness status

**Reconciled:** 2026-06-12 (external tester calibration sprint)  
**Source commit:** `6bcc17a` (pending readiness commit)  
**Live revision:** `dev` (calibration binary hot-deployed) / image `rc-381667a` baseline

## Readiness decisions

| Level | Status | Notes |
|-------|--------|-------|
| **Marketing ready** | **NO** | Wiki, VM install, ≥2 external testers |
| **Private beta ready** | **YES** | Calibration verified; high FP gone on rescan |
| **Controlled demo ready** | **YES** | |
| **External tester #1** | **COMPLETE + calibrated** | Rescan `eb3e7662b31d943c` — 0 high |
| **External tester #2** | **READY TO INVITE** | After operator sends outreach |

## Product dogfood (live DB)

| Metric | Value |
|--------|--------|
| Active-present open (product) | **0** |
| High/critical actionable (product) | **0** |
| Gitea open mapped issues (product) | **0** |

## Calibration sprint (2026-06-12)

| Item | Status |
|------|--------|
| `SEC-HARDCODED-SECRET` placeholder FP | **fixed** |
| `REL-INTERNAL-INFRA-REF` homelab examples | **fixed** |
| `HEALTH-LARGE-FILE` Python scripts | **low severity** |
| Graph/info UI grouping | **shipped** |
| Scanner/SBOM beta messaging | **updated** |
| Wifi_Collector rescan | **120 findings, 0 high, 1 actionable** |

## Remaining blockers (marketing)

1. Gitea wiki HTTP 500
2. External VM clean install proof
3. Second external tester clean cycle (tester #1 done)
4. Optional logged-in template screenshot

## Next step

Onboard **external tester #2** — same report-only constraints.

## Do not repeat

- All-repo scanning
- Issue filing on beta tester repos
- Marketing launch without wiki + VM proof + 2+ external testers
