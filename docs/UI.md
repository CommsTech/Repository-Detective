# Operator UI (Phase 6) + scheduling (Phase 7)

Repository Detective exposes a minimal operator web UI and JSON API for viewing persisted scan history, editing per-repo settings, and configuring scheduled scans.

> **Naming:** See [NAMING.md](NAMING.md). Product name is Repository Detective; internal config still uses `BUGBOT_*`.

## Phase 8 policy enforcement

See [POLICY.md](POLICY.md). Per-repo settings now affect scans: scanners (Go trio, IaC/container scanners hadolint/checkov), workspace, AI, issue creation, and commit status gates. The UI shows effective settings and whether they are enforced on scans.

## Enable / disable

```yaml
ui_enabled: true
ui_base_path: /ui
```

Environment variables:

```text
BUGBOT_UI_ENABLED=true
BUGBOT_UI_BASE_PATH=/ui
```

Requirements:

- `database_enabled: true` — UI and control-plane API need the Phase 5 database
- `api_key` configured — UI and API routes use the same API key auth

If `ui_enabled=false`, JSON API routes under `/api/v1/` still work; HTML UI is not mounted.

## Authentication

All control-plane routes require the Bugbot API key:

- Header: `X-Bugbot-API-Key: your-key`
- Query param: `?api_key=your-key` (convenient for browser UI links)

Example:

```text
https://bugbot.example.com/ui?api_key=YOUR_KEY
```

Secrets (`gitea_token`, `ai_api_key`, database DSN, etc.) are never exposed in API or HTML responses.

## API routes

All require API key auth and return JSON.

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/v1/dashboard/summary` | Dashboard aggregates |
| GET | `/api/v1/repos` | List repositories with summary stats |
| GET | `/api/v1/repos/:id` | Repository detail |
| GET | `/api/v1/repos/:id/settings` | Stored + effective settings |
| PUT | `/api/v1/repos/:id/settings` | Update repo settings |
| GET | `/api/v1/repos/:id/scans` | Scans for repository |
| GET | `/api/v1/repos/:id/findings` | Findings for repository |
| GET | `/api/v1/scans/:scan_id` | Scan detail |
| GET | `/api/v1/scans/:scan_id/scanner-results` | Scanner results |
| GET | `/api/v1/findings` | List findings (filters: severity, category, status, source, repo_id) |
| GET | `/api/v1/findings/:id` | Finding detail |
| GET | `/api/v1/findings/:id/lifecycle` | Lifecycle events |
| GET | `/api/v1/notifications/status` | Redacted global notification config |
| POST | `/api/v1/notifications/test` | Send safe test notification |
| GET | `/api/v1/findings/:id/remediation` | Latest remediation plan |
| POST | `/api/v1/findings/:id/remediation/generate` | Generate remediation plan |
| POST | `/api/v1/remediation/:plan_id/attempt-pr` | Attempt safe remediation PR (when enabled) |
| GET | `/api/v1/remediation/:plan_id/patch-attempts` | List patch attempts |
| GET | `/api/v1/patch-attempts/:attempt_id` | Patch attempt detail |
| GET | `/api/v1/findings/:id/closure-evidence` | Closure evidence status |
| POST | `/api/v1/findings/:id/verify-closure` | Verify closure from latest scan |
| POST | `/api/v1/patch-attempts/:attempt_id/check-merge` | Poll PR merge state |

List routes support `limit` (default 50, max 200) and `offset`.

If `database_enabled=false`, API returns `503` with `"error": "database disabled"`.

### curl examples

```bash
KEY=your-api-key
BASE=https://bugbot.example.com

curl -H "X-Bugbot-API-Key: $KEY" "$BASE/api/v1/dashboard/summary"
curl -H "X-Bugbot-API-Key: $KEY" "$BASE/api/v1/repos"
curl -H "X-Bugbot-API-Key: $KEY" "$BASE/api/v1/repos/1/settings"
curl -X PUT -H "X-Bugbot-API-Key: $KEY" -H "Content-Type: application/json" \
  -d '{"policy_level":"gate_pr","workspace_mode":"archive"}' \
  "$BASE/api/v1/repos/1/settings"
```

## UI pages

| Path | Description |
|------|-------------|
| `/ui` | Dashboard |
| `/ui/repos` | Repository list |
| `/ui/repos/:id` | Repository detail |
| `/ui/repos/:id/settings` | Edit repo settings |
| `/ui/scans/:scan_id` | Scan detail |
| `/ui/scans/:scan_id/graph` | Interactive repository map for scan |
| `/ui/repos/:id/graph` | Latest repository map for repo (local Cytoscape assets, no CDN) |
| `/ui/findings` | Findings list with filters (severity, category, source, status) |
| `/ui/findings/:id` | Finding detail (redacted evidence) |
| `/ui/preinstall` | Pre-install audit — paste third-party repo URL |
| `/ui/preinstall/audits/:audit_id` | Audit results, findings, disclosure drafts |

See [PREINSTALL_AUDIT.md](PREINSTALL_AUDIT.md) for pre-install audit behavior.

## Dashboard metrics (operator signal)

The dashboard separates **actionable backlog** from **raw detector noise** and **platform readiness warnings**:

| Metric | Meaning |
|--------|---------|
| **Open unique findings** | Deduplicated fingerprints in `findings` with status `open` — primary triage queue |
| **New (7d)** | Open findings first seen in the last 7 days |
| **Regressions (7d)** | Open findings first seen earlier but re-detected in the last 7 days |
| **Verified resolved** | Findings with evidence-based closure |
| **Critical / high open** | Actionable severity backlog (shown first) |
| **Low severity backlog** | Rolled up count — not promoted to hero metrics |
| **Raw hits in scans (7d)** | Sum of `issues_found` from completed scans — secondary, includes duplicates |
| **Raw instances (7d)** | Rows in `finding_instances` — not the same as unique findings |
| **Failed scans** | Repository scans with `status=failed`, grouped by reason bucket |
| **Scanner platform warnings** | Per-scanner readiness (configured / installed / affected repos) — **not** counted as findings |

`scanner_tools_missing_count` in the API now counts **distinct scanner names** with `binary_missing` status, not every repeated scan event. Raw missing-tool event totals are exposed separately as `platform.raw_missing_events`.

Remediation candidates explain why auto-remediation may be zero (planner disabled, no safe plans, gates, etc.).

## Finding categories (Phase 10)

Findings and dashboard counts include health check categories alongside security categories:

| Category | Description |
|----------|-------------|
| `tech_debt` | TODO/FIXME/HACK markers, workaround comments |
| `reliability` | Ignored errors, panics, missing HTTP timeouts |
| `maintainability` | Large files/functions, deep nesting |
| `code_quality` | Parameter count and similar structural signals |
| `test_gap` | Missing tests for packages/modules |
| `performance` | Regex-in-loop, long sleeps |
| `ai_generated_risk` | Possible low-context code risk (optional, off by default) |

Filter via UI query params or `GET /api/v1/findings?category=tech_debt`. Pre-install audit pages show category and source columns for all finding types.

## Repo settings — health checks (Phase 10B)

The settings page and API expose per-repo health check toggles and thresholds. Empty/inherit values use global `config.yaml` / `BUGBOT_*` defaults. AI-generated-risk is clearly marked as a cautious heuristic that does not claim AI authorship.

## Repo settings — code graph (Phase 11B)

Per-repo graph toggles and limits (`enable_code_graph`, `graph_max_nodes`, `graph_max_edges`, `graph_timeout_seconds`, `graph_include_functions`, `graph_include_findings`) inherit global defaults when left empty. The graph page uses bundled static assets under `{ui_base_path}/static/` and works offline (no CDN).

Graph export: use **Download JSON** on the graph page or `GET /api/v1/scans/:scan_id/graph/export` / `GET /api/v1/repos/:id/graph/export`.

## Runner jobs (Phase 12)

Dashboard shows runner job counts by status when delegation is in use. Scan detail shows linked runner job ID/status. Operator API: `GET /api/v1/runner/jobs`. See [RUNNERS.md](RUNNERS.md).

## Settings: stored vs active

Per-repo settings are saved in SQLite and shown as stored + effective (merged with global defaults and scan profiles).

The repo settings page leads with a **Scan profile** dropdown, effective summary badges (Security, Go, IaC, Health, Graph, AI, Runner), a **Notifications** section (per-repo overrides; global channel status read-only), and a collapsible **Advanced settings** section for individual toggles. Changing advanced toggles switches the repo to `custom`. See [SCAN_PROFILES.md](SCAN_PROFILES.md).

See [NOTIFICATIONS.md](NOTIFICATIONS.md) for channel setup, event types, and test endpoint.

The API settings response includes `scan_profile`, `profile_modified`, `profile_source`, `effective_profile_summary`, `notification_global`, and `effective_notifications`.

The API and UI include a notice on every settings response/page.

## Validation

- **scan_profile:** `fast`, `standard_deterministic`, `strict_security`, `maintainer_deep`, `preinstall_cautious`, `custom`
- **policy_level:** `monitor_only`, `issue_only`, `gate_pr`, `suggest_fix`, `auto_pr_with_approval`, `auto_pr_low_risk`
- **workspace_mode:** `api`, `archive`, `auto`
- **analysis_depth:** `1`, `2`, `3`
- **severity_gate:** `critical`, `high`, `medium`, `low`, `info`
- **issue_policy:** `off`, `fingerprint`, `all`
- **remediation_policy:** `off`, `suggest`, `approval`, `auto_low_risk`
- **runner_policy:** `core`, `gitea_actions`, `auto`
- **ai_policy:** `allowed`, `disabled`
- **confidence_gate:** 0.0–1.0
- **notification_min_severity:** `critical`, `high`, `medium`, `low`, `info`
- **notification_events:** comma-separated list from [NOTIFICATIONS.md](NOTIFICATIONS.md)
- **notification_cooldown_seconds:** 0–86400

## Safe remediation lifecycle (dashboard)

The dashboard shows counts for each stage of the safe loop:

```text
detect → issue → plan → approve → patch PR → merge → rescan → verified closure
```

Stages include open findings, remediation candidates, approved plans, PRs opened/merged, pending rescan, closure blocked, still present, and resolved verified. Each row includes an operator-facing label (e.g. **Waiting for rescan**, **Verified resolved**).

Finding detail pages show a **Lifecycle stage** banner derived from plan, patch attempt, and closure evidence state.

## Operator readiness (dashboard + status API)

The dashboard includes scanner binary availability (configured / on PATH / version) and feature flags. Same data is available as JSON:

| Endpoint | Auth | Description |
|----------|------|-------------|
| `GET /health` | None | Liveness; tool summary when ready |
| `GET /api/v1/status` | API key | Full readiness JSON |
| `GET /api/v1/about` | None | Product name, version, compatibility |

No tokens, DSNs, or webhook secrets are exposed. See [OPERATOR_READINESS.md](OPERATOR_READINESS.md).

## Rollback

1. Set `ui_enabled: false` to hide HTML UI
2. Set `database_enabled: false` to disable all persistence
