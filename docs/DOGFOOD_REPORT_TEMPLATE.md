# Dogfood report template

Repository Detective — **Inspect. Analyze. Improve.**

Copy this template after your first self-scan. Save as `docs/dogfood-reports/YYYY-MM-DD-repository-detective.md` or your team's wiki.

---

## Repository

| Field | Value |
|-------|-------|
| Full name | `owner/repo` |
| Clone URL | |
| Connected via | webhook / manual analyze |
| Operator | |

## Commit scanned

| Field | Value |
|-------|-------|
| Ref | `main` / branch name |
| Commit SHA | |
| Scan ID | |
| Scan finished at | |

## Scan profile

| Field | Value |
|-------|-------|
| Global profile | `standard_deterministic` / `strict_security` / other |
| Per-repo overrides | yes / no — describe |
| AI enabled | yes / no (dogfood default: **no**) |
| Workspace mode | api / archive / auto |

## Scanner availability

From `GET /api/v1/status` → `tools[]`:

| Tool | Configured | Available | Version |
|------|------------|-----------|---------|
| git | | | |
| trivy | | | |
| grype | | | |
| gitleaks | | | |
| semgrep | | | |
| govulncheck | | | |
| gosec | | | |
| staticcheck | | | |
| hadolint | | | |
| checkov | | | |

Missing tools that mattered for this repo:

```text
(list)
```

## Scan results summary

| Metric | Count |
|--------|-------|
| Total findings | |
| Open findings | |
| Critical | |
| High | |
| Medium | |
| Low | |
| Scanner failures | |
| Issues created in Gitea | |
| Duration | |

## Critical/high findings

| Fingerprint | Source | Title | Valid? | Notes |
|-------------|--------|-------|--------|-------|
| | | | yes / no / unsure | |

Action taken (issue label, comment, deferred, false positive):

```text
```

## Health findings

Notable tech debt, reliability, maintainability, test gap, performance items:

| Category | Count | Notable examples |
|----------|-------|------------------|
| tech_debt | | |
| reliability | | |
| maintainability | | |
| test_gap | | |
| performance | | |

## Repository Map observations

Code graph enabled: yes / no

| Observation | Notes |
|-------------|-------|
| Graph generated | yes / no / timeout |
| Useful for triage | |
| Noise / missing edges | |

## Remediation candidates

| Finding | Plan generated | Approved | Safe for auto-PR | Notes |
|---------|----------------|----------|------------------|-------|
| | | | | |

Remediation PR attempted: **no** (dogfood default) / yes — if yes, link PR and outcome.

## False positives

| Fingerprint | Source | Why false positive | Suggested doc/config fix |
|-------------|--------|--------------------|--------------------------|
| | | | |

## Scanner failures

| Scanner | Error / reason | Impact on closure |
|---------|----------------|-------------------|
| | | |

## UI/API issues found

| Area | Issue | Severity | Ticket/link |
|------|-------|----------|-------------|
| UI | | | |
| API | | | |
| Webhook | | | |
| Performance | | | |

## Docs issues found

| Doc | Issue | Fix proposed |
|-----|-------|--------------|
| | | |

## Recommended fixes before wider rollout

Priority order:

1. 
2. 
3. 

Config changes:

```yaml
# paste recommended config deltas
```

## Go / no-go

| Decision | **Go** / **No-go** / **Go with conditions** |
|----------|---------------------------------------------|
| Wider Gitea repo onboarding | |
| Enable `remediation_pr_enabled` | |
| Enable `notifications_enabled` | |
| Enable runner delegation | |
| Enable LLM auditors | |

Conditions (if go with conditions):

```text
```

Sign-off:

| Role | Name | Date |
|------|------|------|
| Operator | | |

---

## Appendix — quick curl capture

```bash
# Paste redacted status snapshot
curl -s -H "X-Repository-Detective-API-Key: $KEY" http://localhost:8080/api/v1/status

# Paste dashboard summary
curl -s -H "X-Repository-Detective-API-Key: $KEY" http://localhost:8080/api/v1/dashboard/summary
```
