# Notifications (Repository Detective)

Repository Detective — **Inspect. Analyze. Improve.**

Opt-in notifications alert operators to important scan, gate, runner, and pre-install audit events. Messages are concise, sanitized, and never include secrets or raw evidence.

> **No email** in this phase. **No per-repo channel credentials** — Telegram/Slack/Discord/webhook tokens are global only.

## Enable globally

```yaml
notifications_enabled: false
notification_min_severity: high
notification_cooldown_seconds: 300

telegram_enabled: false
telegram_bot_token: ""
telegram_chat_id: ""

slack_enabled: false
slack_webhook_url: ""

discord_enabled: false
discord_webhook_url: ""

webhook_notifications_enabled: false
webhook_notification_url: ""
webhook_notification_secret: ""
```

Environment variables (prefer `REPOSITORY_DETECTIVE_*`; legacy `REPOSITORY_DETECTIVE_*` works via envcompat):

```text
REPOSITORY_DETECTIVE_NOTIFICATIONS_ENABLED
REPOSITORY_DETECTIVE_NOTIFICATION_MIN_SEVERITY
REPOSITORY_DETECTIVE_NOTIFICATION_COOLDOWN_SECONDS
REPOSITORY_DETECTIVE_TELEGRAM_ENABLED
REPOSITORY_DETECTIVE_TELEGRAM_BOT_TOKEN
REPOSITORY_DETECTIVE_TELEGRAM_CHAT_ID
REPOSITORY_DETECTIVE_SLACK_ENABLED
REPOSITORY_DETECTIVE_SLACK_WEBHOOK_URL
REPOSITORY_DETECTIVE_DISCORD_ENABLED
REPOSITORY_DETECTIVE_DISCORD_WEBHOOK_URL
REPOSITORY_DETECTIVE_WEBHOOK_NOTIFICATIONS_ENABLED
REPOSITORY_DETECTIVE_WEBHOOK_NOTIFICATION_URL
REPOSITORY_DETECTIVE_WEBHOOK_NOTIFICATION_SECRET
```

Tokens and webhook URLs are **never logged** or returned by the API/UI (shown as `configured` when set).

## Channels

| Channel | Transport |
|---------|-----------|
| `telegram` | Telegram Bot API `sendMessage` |
| `slack` | Incoming webhook JSON `{ "text": "..." }` |
| `discord` | Webhook JSON `{ "content": "..." }` |
| `webhook` | Signed JSON POST with optional `X-Repository-Detective-Signature` HMAC |

## Per-repo overrides

Nullable fields in `repo_settings` (NULL = inherit global):

| Field | Purpose |
|-------|---------|
| `notifications_enabled` | Disable notifications for one repo while global is on |
| `notification_min_severity` | Raise/lower severity threshold |
| `notification_events` | Comma-separated event filter |
| `notification_cooldown_seconds` | Per-repo cooldown override |

Configure via UI **Notifications** section or `GET/PUT /api/v1/repos/:id/settings`.

## Event types

| Event | Default enabled |
|-------|-----------------|
| `critical_finding` | yes |
| `high_finding` | yes |
| `pr_gate_failed` | yes |
| `scan_failed` | yes |
| `runner_job_failed` | yes |
| `preinstall_do_not_install` | yes |
| `scan_completed_with_findings` | no |
| `scheduled_scan_failed` | no |
| `runner_job_expired` | no |
| `preinstall_caution` | no |
| `disclosure_report_generated` | no |
| `fix_pr_merged` | no |
| `closure_verified` | no |
| `closure_blocked` | no |
| `remediation_still_present` | no |

See [EVIDENCE_CLOSURE.md](EVIDENCE_CLOSURE.md) for closure notification triggers.

Low-severity findings do **not** notify by default.

## Message safety

Notifications may include: repo name, scan ID, finding counts, highest severity, category summary, gate outcome, and UI link when `public_url` is set.

Never included: raw secrets, evidence snippets, scanner JSON, tokens, private disclosure details, or exploit instructions.

Security wording uses neutral phrasing (e.g. “High severity finding detected”) unless conclusively proven otherwise.

## Rate limiting

Cooldown applies per repository + event type (default 300 seconds). Duplicate bursts within the cooldown window are suppressed.

## API

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/v1/notifications/status` | Redacted global notification config |
| POST | `/api/v1/notifications/test` | Send safe test message to enabled channels |

Requires API key auth (same as other control-plane routes).

```bash
curl -X POST -H "X-Repository-Detective-API-Key: $REPOSITORY_DETECTIVE_API_KEY" "$BASE/api/v1/notifications/test"
```

## Integration points

Notifications emit from **core only** (not runner workers):

- Scan finish (failed / high+critical findings)
- PR security gate failure
- Runner job failure / expiry
- Pre-install audit caution / do-not-install
- Disclosure report generation

## Troubleshooting

| Symptom | Check |
|---------|-------|
| No messages | `notifications_enabled`, channel enabled + credentials, repo not disabled |
| Too many alerts | Raise `notification_min_severity`, tighten `notification_events`, increase cooldown |
| Test fails 503 | Global disabled or no channel configured |
| Test fails 502 | Channel HTTP error — verify token/webhook URL |

## Rollback

1. Set `notifications_enabled: false` (or `REPOSITORY_DETECTIVE_NOTIFICATIONS_ENABLED=false`).
2. Restart the core service.
3. Per-repo overrides remain in DB but have no effect while global is off.

To remove credentials from config, clear token/webhook fields and restart. No migration rollback required.
