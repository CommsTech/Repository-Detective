# Gitea Commit Status / Checks

Repository-Detective can optionally post Gitea commit statuses on push and pull request scans. This gives immediate PR feedback without changing issue creation behavior.

## Enable

```yaml
enable_gitea_status: true
gitea_status_context: repository-detective/security-scan
gitea_status_fail_on: high
gitea_status_warn_on: medium
gitea_status_include_scanner_failures: true
public_url: https://repository-detective.example.com
```

Environment equivalents:

```bash
REPOSITORY_DETECTIVE_ENABLE_GITEA_STATUS=true
REPOSITORY_DETECTIVE_GITEA_STATUS_CONTEXT=repository-detective/security-scan
REPOSITORY_DETECTIVE_GITEA_STATUS_FAIL_ON=high
REPOSITORY_DETECTIVE_GITEA_STATUS_WARN_ON=medium
REPOSITORY_DETECTIVE_GITEA_STATUS_INCLUDE_SCANNER_FAILURES=true
REPOSITORY_DETECTIVE_PUBLIC_URL=https://repository-detective.example.com
```

Default is **disabled** (`enable_gitea_status: false`).

## Lifecycle

1. **Pending** — posted when a scan starts (push uses `after` SHA; PR uses `head.sha` when available).
2. **Final** — posted when the scan completes. Descriptions use **policy outcomes**, not “security passed”:
   - `POLICY_MET` → Gitea `success` — required analysis completed; configured conditions not violated
   - `ACTION_REQUIRED` → `failure` (Enforce) or `warning` (Warn) — owner policy conditions violated
   - `EVALUATION_INCOMPLETE` → `error` / non-blocking warn — required scanners incomplete
   - `OBSERVATION_ONLY` → `success` — Observe mode; never blocks workflow

Severity thresholds (`gitea_status_fail_on` / `warn_on`) still feed the policy conditions. Required scanner failures block `POLICY_MET` even when optional tools are missing.

If no commit SHA is available (branch-only PR without `head.sha`), status reporting is skipped with a log message.

Pull requests also receive a **single compact summary comment** (counts + policy outcome + link to canonical issues). Repository Detective does **not** create one inline review comment per finding.

## API

Repository-Detective uses the Gitea commit status endpoint:

```text
POST /api/v1/repos/{owner}/{repo}/statuses/{sha}
```

Status creation failures are logged and do not fail the analysis.

## Scanner failure policy

Bad scanner statuses: `failed`, `timed_out`, `parse_failed`

Non-bad (optional tools): `disabled`, `binary_missing`, `clean`, `found`

`binary_missing` is treated as non-bad because external scanners are optional and several are disabled by default.

## Example deterministic gate

```yaml
workspace_mode: auto
analysis_depth: 2
enable_llm_auditors: false
enable_gitea_status: true

enable_trivy: true
enable_grype: true
enable_gitleaks: true
enable_semgrep: true
enable_linters: true
```

This turns Repository-Detective into a CI-style quality gate on PRs while still opening issues separately when configured.
