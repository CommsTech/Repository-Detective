# External tester #1 — handoff packet

**Status:** Assigned and sent  
**Operator:** commstech  
**Tester handle:** `ext-operator-jrice`  
**Contact:** Matrix `@jrice:commsnet.org` (operator secure channel)  
**Handoff date:** 2026-06-12  
**Feedback deadline:** **2026-06-19** (7 days from onboard)

---

## Tester assignment

| Field | Value |
|-------|-------|
| **Tester** | `ext-operator-jrice` |
| **Role** | Invited external homelab maintainer (technical operator) |
| **Contact** | Matrix `@jrice:commsnet.org` |
| **Approved repo** | `commstech/Wifi_Collector` |
| **Repo ID** | 10 |
| **Expected language** | Python (primary), YAML, Markdown |
| **Expected size** | Small (~26–28 files) |

## Sensitivity review

| Check | Result |
|-------|--------|
| Owned by tester/org | **yes** — commstech homelab; tester maintains repo |
| PHI/PII | **none expected** — WiFi collection/analysis tooling |
| Customer secrets | **none** |
| Production credentials in tree | **not expected** — operator spot-check before share |
| Third-party code without permission | **none** |
| Non-sensitive homelab tooling | **yes** |

## Approved scan type

**Report-only only** for first scan:

```text
report_only_dry_run: true
issue_policy: off (effective via dry-run)
analysis_depth: 2
enable_code_graph: true
scan_profile: standard_deterministic
```

## Features disabled (enforced)

| Feature | State |
|---------|-------|
| Issue filing | **off** |
| Remediation PR | **off** |
| AI Recommendations | **off** |
| Runner delegation | **off** |
| Container scanning | **off** |
| All-repo / bulk scanning | **not allowed** |
| Third-party disclosure | **off** |

## Required from tester

1. Run **one report-only scan** on `commstech/Wifi_Collector`.
2. Record **scan ID** in every feedback item.
3. Use Gitea issue templates on `commstech/Repository-Detective` (preferred) or doc templates below.
4. **Never** paste secrets, tokens, `.env`, PHI/PII, or customer data.

## Required feedback format

Use Gitea templates on `commstech/Repository-Detective` with scan ID in every submission:

| Use case | Template |
|----------|----------|
| General beta feedback | [beta_feedback](https://git.commsnet.org/commstech/repository-detective/issues/new?template=beta_feedback) |
| Bug / defect | [bug_report](https://git.commsnet.org/commstech/repository-detective/issues/new?template=bug_report) |
| False positive | [scanner_false_positive](https://git.commsnet.org/commstech/repository-detective/issues/new?template=scanner_false_positive) |
| Missed detection | [missed_detection](https://git.commsnet.org/commstech/repository-detective/issues/new?template=missed_detection) |

Doc fallbacks: [PRIVATE_BETA_FEEDBACK_TEMPLATE.md](PRIVATE_BETA_FEEDBACK_TEMPLATE.md), [PRIVATE_BETA_BUG_REPORT_TEMPLATE.md](PRIVATE_BETA_BUG_REPORT_TEMPLATE.md), [PRIVATE_BETA_FALSE_POSITIVE_TEMPLATE.md](PRIVATE_BETA_FALSE_POSITIVE_TEMPLATE.md).

## Documentation links

- [PRIVATE_BETA_RC_RELEASE_NOTES.md](PRIVATE_BETA_RC_RELEASE_NOTES.md)
- [PRIVATE_BETA_TEST_SCOPE.md](PRIVATE_BETA_TEST_SCOPE.md)
- [PRIVATE_BETA_OPERATOR_RUNBOOK_RC.md](PRIVATE_BETA_OPERATOR_RUNBOOK_RC.md)

## Live environment

| Item | Value |
|------|--------|
| Live revision | `rc-381667a` |
| Product commit | `f3dcb9a` (main at handoff) |
| Access | Operator-provided URL + API key on secure channel |

## Safety statement

- Do not scan repos you do not own or lack written permission to assess.
- Redact secrets from screenshots, logs, and issue bodies.
- Report-only scans must not create forge issues in your repository.
- Contact operator before enabling any advanced feature (filing, runner, container scan).

## Operator checklist

- [x] Tester identity recorded (`ext-operator-jrice`)
- [x] Repo `commstech/Wifi_Collector` approved
- [x] Report-only enforced (`report_only_dry_run: true`)
- [x] API key / access URL sent on secure channel
- [x] Handoff date logged — see [external-tester-1-outreach-sent.md](external-tester-1-outreach-sent.md)

## Reference (internal rehearsal — not this tester)

Internal cohort `operator-cohort-1` on `commstech/PCAP_Analyser`, scan `512145e55d4488ea` — see [first-tester-feedback-summary.md](first-tester-feedback-summary.md).
