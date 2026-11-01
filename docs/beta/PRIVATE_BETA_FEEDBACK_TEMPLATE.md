# Private beta feedback template

Copy into operator feedback channel or use Gitea template [`beta_feedback`](https://git.commsnet.org/commstech/repository-detective/issues/new?template=beta_feedback). **Redact secrets before sending.**

**Workflow:** report-only scan → capture scan ID → review findings → file Gitea issue with correct template → never paste secrets.

---

## Tester information

- **Tester name / handle:**
- **Date:**
- **Repository Detective commit:** `6d011cf` or `/api/v1/about` output
- **Live revision (if known):** e.g. `rc-e3e19ec`
- **Install method:** [ ] Docker Compose  [ ] Binary  [ ] Other: ___

## Repository under test

- **Forge:** [ ] Gitea  [ ] GitHub  [ ] Other
- **Repo:** `owner/name`
- **Scan ID:**
- **Report-only:** [ ] Yes  [ ] No — issues filed: ___

## Feedback category (check one primary)

- [ ] Installation friction
- [ ] Confusing UI
- [ ] Finding unclear
- [ ] False positive → use [PRIVATE_BETA_FALSE_POSITIVE_TEMPLATE.md](PRIVATE_BETA_FALSE_POSITIVE_TEMPLATE.md)
- [ ] Missed issue
- [ ] Scanner unavailable
- [ ] Report quality
- [ ] SBOM quality
- [ ] Graph quality
- [ ] Pre-install audit quality
- [ ] Performance
- [ ] Docs gap
- [ ] Trust/safety concern
- [ ] Other: ___

## Description

What happened? What did you expect?

## Steps to reproduce

1.
2.
3.

## Evidence (redacted)

- Screenshot filename or description (no API keys in URL if possible)
- Log excerpt (no tokens)
- Finding IDs if applicable

## Severity (tester assessment)

- [ ] Blocks my testing
- [ ] Annoying but workable
- [ ] Minor / suggestion

## Operator use only

- Triage bucket:
- Linked issue / calibration ID:
- Resolution:
