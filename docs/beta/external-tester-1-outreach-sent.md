# External tester #1 — outreach sent

**Date sent:** 2026-06-12  
**Channel:** Matrix DM `@jrice:commsnet.org` (secure)  
**Operator:** commstech  
**Tester:** `ext-operator-jrice`  
**Handoff packet:** [external-tester-1-handoff.md](external-tester-1-handoff.md)

---

## Message sent (redacted — no secrets)

Subject: Repository Detective — invited operator private beta

---

Hi — you're invited to the **Repository Detective private operator beta**.

**Important:** This is an **invited operator beta**, not a production-ready public release. Do **not** treat this as marketing-ready software or share broadly outside the beta agreement.

### Scope (strict)

- **One owned repo only:** `commstech/Wifi_Collector`
- **First scan: report-only** — findings in RD only; **no Gitea/GitHub issues** filed in your repo
- **No PRs**, no Remediation PR, no AI Recommendations, no runner delegation, no container scanning unless we explicitly approve later
- **Do not scan** repos with secrets, PHI/PII, customer data, or third-party code you don't have permission to assess

### What to do

1. Read [PRIVATE_BETA_RC_RELEASE_NOTES.md](PRIVATE_BETA_RC_RELEASE_NOTES.md) and [PRIVATE_BETA_TEST_SCOPE.md](PRIVATE_BETA_TEST_SCOPE.md)
2. Use the access URL and API key sent on this secure channel (not in email/wiki)
3. Run **one report-only scan** on `commstech/Wifi_Collector` with `report_only_dry_run: true`
4. Record the **scan ID** — required in all feedback
5. Submit feedback via Gitea templates on `commstech/Repository-Detective`:
   - General: [beta_feedback](https://git.commsnet.org/commstech/repository-detective/issues/new?template=beta_feedback)
   - Bug: [bug_report](https://git.commsnet.org/commstech/repository-detective/issues/new?template=bug_report)
   - False positive: [scanner_false_positive](https://git.commsnet.org/commstech/repository-detective/issues/new?template=scanner_false_positive)
   - Missed detection: [missed_detection](https://git.commsnet.org/commstech/repository-detective/issues/new?template=missed_detection)

Doc fallbacks if needed: [PRIVATE_BETA_FEEDBACK_TEMPLATE.md](PRIVATE_BETA_FEEDBACK_TEMPLATE.md), [PRIVATE_BETA_BUG_REPORT_TEMPLATE.md](PRIVATE_BETA_BUG_REPORT_TEMPLATE.md), [PRIVATE_BETA_FALSE_POSITIVE_TEMPLATE.md](PRIVATE_BETA_FALSE_POSITIVE_TEMPLATE.md).

### Safety

- Never paste secrets, tokens, `.env` contents, or customer data into issues or chat
- Redact screenshots and logs
- Contact me before enabling issue filing or any advanced feature

**Feedback due:** 2026-06-19  
**Live revision:** `rc-381667a`

Thanks — your structured feedback (with scan ID) directly shapes whether we expand the private beta.

---

## Attachments / links included

- [external-tester-1-handoff.md](external-tester-1-handoff.md)
- [PRIVATE_BETA_RC_RELEASE_NOTES.md](PRIVATE_BETA_RC_RELEASE_NOTES.md)
- [PRIVATE_BETA_TEST_SCOPE.md](PRIVATE_BETA_TEST_SCOPE.md)
- [PRIVATE_BETA_OPERATOR_RUNBOOK_RC.md](PRIVATE_BETA_OPERATOR_RUNBOOK_RC.md)

## Operator follow-up

- [x] Outreach sent 2026-06-12
- [x] Operator-initiated report-only scan recorded — see [external-tester-1-scan-report.md](../dogfood-reports/external-tester-1-scan-report.md)
- [x] Feedback collected — see [external-tester-1-feedback-summary.md](external-tester-1-feedback-summary.md)
