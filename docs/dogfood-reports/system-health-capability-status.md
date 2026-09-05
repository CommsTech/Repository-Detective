# System health capability status

## What was broken

Health page listed “Remediation PR disabled”, “Runner delegation disabled”, etc. without explaining whether default-off, misconfigured, or degraded.

## Files changed

- `ui/capability_status.go` — actionable capability cards
- `ui/handler.go` — inject capabilities into health page
- `ui/templates/health.html` — Platform capabilities table

## Capability behavior

| Capability | Disabled reason when off | Enabled/degraded checks |
|------------|------------------------|-------------------------|
| Remediation PR | Default-off; needs `remediation_pr_enabled` + Gitea token | Shows safety note about approval gates |
| Runner delegation | Default-off; needs secret + callback | Degraded if flag on but secret missing |
| Notifications | Needs channel config | Degraded if enabled but no webhook/Slack/Discord/Telegram |
| Pre-install audit | Needs `preinstall_audit_enabled` | Degraded if `public_url` unset |

Each row includes config keys and link to Configure or relevant page.

## Tests

- `ui/executive_report_test.go` — `TestBuildCapabilityStatusesRemediationPRDisabled`

## Manual verification

Open `/ui/health` — Platform capabilities section with reasons and config keys.

## Remaining risks

- Per-repo notification overrides not expanded on health page (link to repo settings)
