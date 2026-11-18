# First tester outreach — invited operator beta

**Audience:** Technical operators (not public marketing)  
**Tone:** Invited operator beta — **not** production-ready SaaS launch

---

## Subject line (email / chat)

Repository Detective — invited operator beta (report-only)

---

## Message body

Hi,

You're invited to the **invited operator beta** of **Repository Detective** — *Inspect. Analyze. Improve.*

This is a **private beta for technical operators** who want self-hosted, report-only repository assessment. It is **not** a production-ready public product announcement.

### What you get

- Deterministic multi-scanner analysis on **one repo you own**
- Actionable findings with severity, confidence, and detail pages
- SBOM visibility (when manifests exist)
- Repository map / graph insight
- Safe pre-install audits (report-only, separate workflow)
- Structured feedback channel

### What is required

- **One owned repo** — small to medium, non-sensitive (no PHI/PII/customer secrets)
- **Report-only first scan** — no issue filing on your first run
- Feedback using Gitea issue templates ([`.gitea/ISSUE_TEMPLATE/`](https://git.commsnet.org/commstech/repository-detective/src/branch/main/.gitea/ISSUE_TEMPLATE/)) with **scan ID** included
- Redact secrets from logs and screenshots

### What is off by default

- Issue filing in your forge
- Remediation PRs
- AI Recommendations
- Runner delegation / container host scanning
- All-repo bulk scanning

Advanced features require **explicit operator approval**.

### Attachments / links

1. [PRIVATE_BETA_RC_RELEASE_NOTES.md](PRIVATE_BETA_RC_RELEASE_NOTES.md)
2. [PRIVATE_BETA_TEST_SCOPE.md](PRIVATE_BETA_TEST_SCOPE.md)
3. [PRIVATE_BETA_FEEDBACK_TEMPLATE.md](PRIVATE_BETA_FEEDBACK_TEMPLATE.md)
4. [PRIVATE_BETA_BUG_REPORT_TEMPLATE.md](PRIVATE_BETA_BUG_REPORT_TEMPLATE.md)
5. [PRIVATE_BETA_FALSE_POSITIVE_TEMPLATE.md](PRIVATE_BETA_FALSE_POSITIVE_TEMPLATE.md)
6. Operator runbook (on request): [PRIVATE_BETA_OPERATOR_RUNBOOK_RC.md](PRIVATE_BETA_OPERATOR_RUNBOOK_RC.md)

### First scan checklist

1. Install beta bundle (operator-provided)
2. Set API key and forge token locally — **never commit `.env`**
3. Connect **one** repository
4. Run scan with `"report_only_dry_run": true`
5. Confirm **0 issues** created in Gitea/GitHub
6. Review findings, graph, SBOM page
7. Submit feedback template with scan ID

### Support

Contact operator via [OPERATOR_CHANNEL].

---

## OpenClaw / internal messaging instruction

```text
Prepare invited-operator-beta messaging ONLY.
Do NOT write public launch, SaaS, or production-ready marketing copy.

Position Repository Detective as:
- private beta for technical operators
- report-only repo assessment first
- actionable findings + SBOM visibility + graph insight
- safe pre-install audits
- self-hosted on Gitea (GitHub secondary, not release-proven for filing)

Avoid: "production-ready", "enterprise-grade", "replace your security team", public CTAs.
```

## Operator send checklist

- [ ] Tester selected and sensitivity reviewed
- [ ] Beta package checksum verified
- [ ] API key issued (secure channel)
- [ ] Repo connected or tester instructed to connect one repo
- [ ] Report-only enforced in config or scan request
- [ ] Feedback deadline communicated
- [ ] No marketing URLs or public posts
