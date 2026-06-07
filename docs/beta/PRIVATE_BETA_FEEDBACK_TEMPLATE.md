# Private beta feedback template

Copy this form into your feedback channel. Redact secrets before sending.

---

## Tester information

- **Tester name / handle:**
- **Date:**
- **Repository Detective version / commit:** (from `/api/v1/about` or bundle README)
- **Install method:** [ ] Docker Compose  [ ] Binary  [ ] Other: ___
- **Platform:** (e.g. Linux amd64, Ubuntu 24.04, Docker on macOS)

## Repository under test

- **Forge:** [ ] Gitea  [ ] GitHub
- **Repo:** `owner/name`
- **Primary language(s):**
- **Approximate size:** (files / LOC if known)

## Install experience

- **Install succeeded:** [ ] Yes  [ ] No
- **Issues during install:** (describe)
- **Time to first healthy `/health`:** ___ minutes

## First scan (report-only)

- **Scan ID:**
- **Used `report_only_dry_run: true`:** [ ] Yes  [ ] No
- **Scan duration:** ___ seconds / minutes
- **Findings count:**
- **Issues created in forge:** (must be 0 for report-only) ___

## Scanner results

| Scanner | Ran | Failed | Notes |
|---------|-----|--------|-------|
| trivy | | | |
| grype | | | |
| gitleaks | | | |
| semgrep | | | |
| staticcheck | | | |
| gosec | | | |
| hadolint | | | |
| checkov | | | |
| ruff / linters | | | |
| code graph | | | |

## Quality feedback

### False positives

List finding IDs or descriptions that appear incorrect:

1.
2.

### Missed findings

Describe vulnerabilities or quality issues you expected but did not see:

1.

### Confusing UI

Pages or flows that were unclear:

-

### Report usefulness

- **Executive report helpful:** [ ] Yes  [ ] Somewhat  [ ] No
- **Comments:**

### Calibration / learning

- **Visited `/ui/learning`:** [ ] Yes  [ ] No
- **Recommendations sensible:** [ ] Yes  [ ] Somewhat  [ ] No
- **Suggestions:**

## Logs (redacted)

```
Paste docker logs or relevant lines here.
Remove api_key, token, secret, password values.
```

## Overall

- **Would continue private beta testing:** [ ] Yes  [ ] Maybe  [ ] No
- **Blockers for wider rollout:**
- **Other comments:**

---

Thank you for testing Repository Detective private beta.
