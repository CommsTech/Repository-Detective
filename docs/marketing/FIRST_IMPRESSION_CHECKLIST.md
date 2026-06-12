# First impression checklist

Updated: 2026-06-12 (external tester #1 complete)

## Cohort 1 rehearsal (internal)

- [x] Report-only scan on `commstech/PCAP_Analyser` — 0 issues filed
- [x] Operator feedback recorded (`operator-cohort-1`)

## External tester #1

- [x] Named external tester assigned (`ext-operator-jrice`)
- [x] Outreach sent — [external-tester-1-outreach-sent.md](../beta/external-tester-1-outreach-sent.md)
- [x] Report-only scan `85a8ab62e76da076` on `commstech/Wifi_Collector` — 0 issues, 0 PRs
- [x] Structured feedback received — [external-tester-1-feedback-summary.md](../beta/external-tester-1-feedback-summary.md)
- [ ] Calibration sprint (FP noise, SBOM messaging) before tester #2
- [ ] Second external tester invited

## Live product walkthrough

- [x] Health page loads
- [x] Configure shows **AI recommendations** (provider-neutral)
- [x] Finding detail has summary / risk / fix / calibration sections
- [x] Repo SBOM page loads
- [x] Scan SBOM page loads
- [x] SBOM download with real artifact (product repo)
- [x] Screenshots captured (`docs/assets/screenshots/`)
- [ ] Wiki pages live on Gitea

## Trust signals

- [x] Dry-run scans create 0 issues (beta repos + external tester #1)
- [x] Live structured issue body deployed (`rc-381667a`)
- [x] AI Recommendations disabled by default
- [x] Product dogfood `active_present_open` = **0**
- [x] GitHub issue provider labeled not release-proven
- [x] Gitea feedback templates used by external tester

## Decision

**Private beta:** YES — first external tester complete; pause expansion for calibration.  
**Controlled demo:** YES.  
**Public marketing:** NOT READY (wiki + VM install + 2+ external testers + calibration).
