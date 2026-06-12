# External tester #1 — handoff packet

**Status:** Ready to send  
**Operator:** commstech  
**Tester handle:** _TBD — assign before send_  
**Feedback deadline:** _TBD (suggest 7 days from onboard)_

---

## Scope

| Item | Value |
|------|--------|
| **One approved repo** | Tester-owned, non-sensitive (operator confirms name before connect) |
| **Scan mode** | **Report-only** (`report_only_dry_run: true` or repo `issue_policy=off`) |
| **Live revision** | `rc-381667a` |
| **Product commit** | `d5548f1` (main) |

## Features disabled (default)

- Issue filing on tester repo
- Remediation PR
- AI Recommendations
- Runner delegation
- Container scanning (production hosts)
- All-repo / bulk scanning
- Third-party disclosure submission

## Required from tester

1. Run **one report-only scan** on the approved repo.
2. Record **scan ID** in every feedback item.
3. Use Gitea issue templates on `commstech/Bugbot` (preferred) or doc templates below.
4. **Never** paste secrets, tokens, `.env`, PHI/PII, or customer data.

## Feedback templates (Gitea)

| Use case | Template |
|----------|----------|
| General beta feedback | [beta_feedback](https://git.commsnet.org/commstech/Bugbot/issues/new?template=beta_feedback) |
| Bug / defect | [bug_report](https://git.commsnet.org/commstech/Bugbot/issues/new?template=bug_report) |
| False positive | [scanner_false_positive](https://git.commsnet.org/commstech/Bugbot/issues/new?template=scanner_false_positive) |

## Documentation links

- [PRIVATE_BETA_RC_RELEASE_NOTES.md](PRIVATE_BETA_RC_RELEASE_NOTES.md)
- [PRIVATE_BETA_TEST_SCOPE.md](PRIVATE_BETA_TEST_SCOPE.md)
- [PRIVATE_BETA_FEEDBACK_TEMPLATE.md](PRIVATE_BETA_FEEDBACK_TEMPLATE.md)
- [PRIVATE_BETA_BUG_REPORT_TEMPLATE.md](PRIVATE_BETA_BUG_REPORT_TEMPLATE.md)
- [PRIVATE_BETA_FALSE_POSITIVE_TEMPLATE.md](PRIVATE_BETA_FALSE_POSITIVE_TEMPLATE.md)
- [PRIVATE_BETA_OPERATOR_RUNBOOK_RC.md](PRIVATE_BETA_OPERATOR_RUNBOOK_RC.md)

## Safety statement

- Do not scan repos you do not own or lack written permission to assess.
- Redact secrets from screenshots, logs, and issue bodies.
- Report-only scans must not create forge issues in your repository.
- Contact operator before enabling any advanced feature (filing, runner, container scan).

## Operator checklist before send

- [ ] Tester identity recorded
- [ ] Repo `owner/name` approved
- [ ] Report-only enforced in RD repo settings
- [ ] API key / access URL sent on secure channel
- [ ] Handoff date logged in `first-tester-feedback-summary.md`

## Reference rehearsal

Internal rehearsal completed on `commstech/PCAP_Analyser`, scan `512145e55d4488ea` — see [first-tester-feedback-summary.md](first-tester-feedback-summary.md).
