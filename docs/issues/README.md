# Prepared Gitea backlog (closeout sprint)

These files are **issue-ready markdown** for `commstech/Repository-Detective`. They were **not** auto-created on Gitea (no unattended API bulk create).

Create manually or extend `scripts/gitea-backlog-setup.sh` after reviewing for duplicates against [ISSUE_BACKLOG.md](../ISSUE_BACKLOG.md).

| File | Priority | Sprint |
|------|----------|--------|
| [P2-wiki-publish-automation.md](P2-wiki-publish-automation.md) | P2 | 6 |
| [P1-log-redaction-all-scanners.md](P1-log-redaction-all-scanners.md) | P1 | 3 |
| [P1-keyboard-shortcuts-nav.md](P1-keyboard-shortcuts-nav.md) | P1 | 2 |
| [P2-optimization-checks.md](P2-optimization-checks.md) | P2 | 5 |
| [P3-doc-detective-integration.md](P3-doc-detective-integration.md) | P3 | 5 |

## Shipped in closeout (do not re-open)

- Dashboard chart sizing and empty states — see `ui/static/dashboard-charts.js`, `theme.css`
- Scanner coverage degraded vs optional — see `operator/tool_display.go`, `docs/SCANNER_HEALTH.md`
- Privacy documentation — `docs/PRIVACY_AND_DATA_PROTECTION.md`
- Issue templates — `.gitea/ISSUE_TEMPLATE/`
- Evidence redaction on store — `store/recorder.go` uses `issues.SanitizeSecretEvidence`
