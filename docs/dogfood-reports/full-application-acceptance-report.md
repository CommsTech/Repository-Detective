# Full application acceptance report

**Date:** 2026-06-10  
**Live revision:** `rc-e3e19ec`  
**Git:** `e3e19ec` + redeploy docs

## Summary

Repository Detective is **private-beta / controlled-demo ready** on live RC. **Marketing remains NOT READY.**

## Live deploy

- All-in-one image rebuilt from source (Alpine/musl — fixes hot-swap failure)
- Container healthy; version `rc-e3e19ec`
- RC routes verified: AI recommendations config, SBOM UI, actionable findings

## Key verifications

| Area | Result |
|------|--------|
| Findings 37361 | PASS — all actionable sections |
| SBOM repo/scan UI | PASS |
| AI Recommendations | PASS — disabled, provider-neutral |
| UI route crawl | PASS — 15/15 |
| Non-product scans | 2 dry-runs complete, 0 issues filed |
| Container logs | PASS |

## Open items

1. Product `active_present_open` = 21 (was 2 pre-deploy) — investigate
2. Gitea wiki HTTP 500
3. GitHub issue filing not release-proven
4. External clean install
5. SBOM download with persisted artifact
6. Screenshots / visual QA

## Marketing decision

**NOT READY** for public marketing.  
**READY** for credible private beta with invited operators.
