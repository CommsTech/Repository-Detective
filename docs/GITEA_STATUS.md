# Gitea Commit Status / Checks

Bugbot can optionally post Gitea commit statuses on push and pull request scans. This gives immediate PR feedback without changing issue creation behavior.

## Enable

```yaml
enable_gitea_status: true
gitea_status_context: bugbot/security-scan
gitea_status_fail_on: high
gitea_status_warn_on: medium
gitea_status_include_scanner_failures: true
public_url: https://bugbot.example.com
```

Environment equivalents:

```bash
BUGBOT_ENABLE_GITEA_STATUS=true
BUGBOT_GITEA_STATUS_CONTEXT=bugbot/security-scan
BUGBOT_GITEA_STATUS_FAIL_ON=high
BUGBOT_GITEA_STATUS_WARN_ON=medium
BUGBOT_GITEA_STATUS_INCLUDE_SCANNER_FAILURES=true
BUGBOT_PUBLIC_URL=https://bugbot.example.com
```

Default is **disabled** (`enable_gitea_status: false`).

## Lifecycle

1. **Pending** — posted when a scan starts (push uses `after` SHA; PR uses `head.sha` when available).
2. **Final** — posted when the scan completes:
   - `failure` if any issue severity is at or above `gitea_status_fail_on` (default `high`)
   - `warning` (mapped to Gitea `failure` with warning description) if any issue is at or above `gitea_status_warn_on` (default `medium`)
   - `error` if enabled scanners fail execution (`failed`, `timed_out`, `parse_failed`) and `gitea_status_include_scanner_failures=true`
   - `success` when no qualifying findings and no relevant scanner failures

If no commit SHA is available (branch-only PR without `head.sha`), status reporting is skipped with a log message.

## API

Bugbot uses the Gitea commit status endpoint:

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

This turns Bugbot into a CI-style quality gate on PRs while still opening issues separately when configured.
