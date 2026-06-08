# Documentation upgrade report

**Date:** 2026-06-02

## Guides created (`docs/guides/`)

| Guide | Screenshot placeholders |
|-------|-------------------------|
| INSTALL_STEP_BY_STEP.md | dashboard, configure |
| FIRST_REPO_SCAN.md | dashboard, repo-detail |
| MANUAL_SCAN_NOW.md | scan-now-modal |
| REPORT_ONLY_MODE.md | configure |
| ENABLE_ISSUE_FILING.md | configure |
| PREINSTALL_AUDIT_GUIDE.md | preinstall |
| REPO_SETTINGS_AND_POLICIES.md | repo-detail |
| ISSUE_FINDING_RECONCILIATION.md | reconciliation |
| SBOM_AND_DEPENDENCY_SCANNING.md | scanner-status |
| SECRET_SCANNING_AND_GIT_HISTORY.md | configure-secret-scan |
| LEARNING_AND_CALIBRATION.md | learning |
| TROUBLESHOOTING.md | dashboard |

## Screenshot capture

- Script: `scripts/capture-doc-screenshots.sh`
- Target folder: `docs/assets/screenshots/`
- Manifest: `docs/assets/screenshots/README.md`
- **Screenshots not auto-captured** in this environment (no headless Chrome on host). Placeholders referenced in guides.

## Wiki sync

23 pages under `docs/wiki/` ready for `publish-gitea-wiki.sh` (see gitea-wiki-publish-report.md).

## Reconciliation documentation

`ISSUE_FINDING_RECONCILIATION.md` explains findings without forge issues vs mapped issues and stale `external_issues` repair.
