# First impression checklist

Updated: 2026-06-11 (blocker burn-down sprint)

## Live product walkthrough

- [x] Health page loads
- [x] Configure shows **AI recommendations** (provider-neutral)
- [x] Finding detail has summary / risk / fix / calibration sections
- [x] Repo SBOM page loads
- [x] Scan SBOM page loads
- [x] SBOM download with real artifact (CycloneDX, 895 components)
- [x] Screenshots captured (`docs/assets/screenshots/`)
- [ ] Wiki pages live on Gitea

## Trust signals

- [x] Dry-run scans create 0 issues (2 non-product repos)
- [x] AI Recommendations disabled by default
- [x] Product dogfood `active_present_open` = **0**
- [x] GitHub issue provider labeled not release-proven

## Decision

**Private beta / demo:** acceptable for invited operators.  
**Public marketing:** not yet (wiki + full clean install).
