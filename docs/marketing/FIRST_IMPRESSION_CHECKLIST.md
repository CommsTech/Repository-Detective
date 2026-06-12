# First impression checklist

Updated: 2026-06-12 (external beta stabilization)

## Cohort 1 rehearsal

- [x] Report-only scan on `commstech/PCAP_Analyser` — 0 issues filed
- [x] Operator feedback recorded
- [x] Full `go test ./...` PASS (SBOM fix)
- [x] Live deploy `rc-381667a` — structured issue body verified
- [x] External tester handoff packet prepared
- [ ] Named external tester onboarded + feedback received
- [ ] Second tester invited (after feedback triage)

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

- [x] Dry-run scans create 0 issues (beta repos + scratch dry-run)
- [x] Live structured issue body deployed (`rc-381667a`)
- [x] AI Recommendations disabled by default
- [x] Product dogfood `active_present_open` = **0**
- [x] GitHub issue provider labeled not release-proven

## Decision

**Private beta expansion:** YES — invited cohort with report-only first.  
**Controlled demo:** YES.  
**Public marketing:** NOT READY (wiki + full clean install + external named tester).
