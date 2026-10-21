# Issue filing policy (beta)

Repository Detective files issues only when:

1. `auto_create_issues` is enabled (not dry-run / report-only modes)
2. Scan type allows filing (not pre-install audit)
3. Finding meets severity/confidence thresholds
4. Target forge matches the repository's provider and coordinates

## Gitea

Primary supported forge. Duplicate fingerprints update existing linked issues instead of creating new ones.

## GitHub

Code path exists (`issues.Manager` + `github` forge). Treat as **beta-unproven** until RC regression tests pass for your org token and repo mapping.

## GitLab

**Not implemented.** UI and docs must not imply GitLab issue filing works.

## Container scans

Issue filing disabled by default. Use report + operator triage.
