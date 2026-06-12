# Current readiness status

**Reconciled:** 2026-06-11  
**Source commit:** `1c862aa`  
**Live revision:** `rc-e3e19ec` (all-in-one, healthy)

> The baseline at `76bd87d` with active-present 21 is **stale**. Blocker burn-down completed in commits `581d534`–`1c862aa`.

## Readiness decisions

| Level | Status | Notes |
|-------|--------|-------|
| **Marketing ready** | **NO** | Wiki blocked; live Gitea filing unproven; full VM install pending |
| **Private beta ready** | **YES** | Dogfood clean; core routes verified; honest provider matrix |
| **Controlled demo ready** | **YES** | Findings detail, SBOM UI/download, screenshots available |

## Product dogfood (live DB, scan `926a5f56a26f03c9`)

| Metric | Value |
|--------|-------|
| Active-present open | **0** |
| Actionable active | **0** |
| High/critical active | **0** |
| Informational active | **0** |
| Gitea open mapped issues | **0** |

## Blocker checklist

| # | Item | Status | Evidence |
|---|------|--------|----------|
| 1 | Product dogfood clean | **ready** | Rescan report; live DB 2026-06-11 |
| 2 | Gitea issue target | **partial** | Dry-run + unit tests; live filing deferred |
| 3 | SBOM download | **ready** | CycloneDX 895 components; HTTP 200 live |
| 4 | Screenshots / visual QA | **ready** | 12 PNGs; secret scan clean |
| 5 | External clean install | **partial** | `make beta-release`; full VM not proven |
| 6 | Wiki populated | **blocked** | Gitea wiki git HTTP 500 |
| 7 | GitHub provider honest | **ready** | Not release-proven (Option B) |
| 8 | GitLab provider | **not_implemented** | Unchanged |
| 9 | Container logs clean | **ready** | No panics; expected warnings only |
| 10 | Non-product beta scans | **ready** | PCAP_Analyser (12), ansible_playbooks (78); 0 issues |

## Live feature flags (from `/health`)

| Feature | State |
|---------|-------|
| Runner delegation | disabled |
| Remediation PR | disabled |
| Pre-install audit | enabled (report-only) |
| AI Recommendations | disabled by default |
| Container scanning | opt-in (disabled) |

## Remaining blockers

1. Gitea wiki server-side HTTP 500
2. Controlled live Gitea issue filing regression on owned repo
3. Dedicated clean VM external install proof
4. GitHub issue provider live proof (optional; demoted honestly)
5. Bundle syft in image for native repo SBOM (operational gap; controlled proof exists)

## Next recommended batch

1. Operator: fix Gitea wiki bare repo / storage
2. Controlled Gitea filing test on owned scratch repo
3. Clean VM install following `docs/SETUP.md`
4. Optional: redeploy `repository-detective:rc-dogfood` when operator approves (not required for private beta)

## Do not repeat

- Deploy sprint from `76bd87d`
- Active-present 21 triage/rescan workflow
- All-repo scanning
- Marketing launch
