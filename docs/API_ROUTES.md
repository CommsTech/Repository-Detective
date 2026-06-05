# API routes reference

**Product:** Repository Detective — Inspect. Analyze. Improve.  
**Base path:** `/api/v1` (unless noted)

## Authentication

| Method | Header |
|--------|--------|
| **Preferred** | `X-Repository-Detective-API-Key: <key>` |
| Legacy | `X-Bugbot-API-Key: <key>` |
| Alternative | `Authorization: Bearer <key>` |

Public routes: `GET /health`, `GET /onboard`, `GET /ui/static/*` (assets only).

Webhook: `POST /webhook` — Gitea HMAC (`X-Gitea-Signature`), not API key.

Runner worker routes: `/api/v1/runner/*` — HMAC (`X-Runner-*`), not operator API key.

---

## Health / about / status

| Method | Path | Auth | Beta | Purpose |
|--------|------|------|------|---------|
| GET | `/health` | none | ✅ | Liveness; scanner tools summary |
| GET | `/api/v1/about` | API key | ✅ | Product name, compatibility flags |
| GET | `/api/v1/status` | API key | ✅ | Runtime features, no secrets |
| POST | `/api/v1/config/reload` | API key | ✅ | Reload config from disk |

---

## Analyze / webhooks

| Method | Path | Auth | Beta | Purpose |
|--------|------|------|------|---------|
| POST | `/webhook` | webhook secret | ✅ | Gitea push/PR events |
| POST | `/api/v1/analyze` | API key | ✅ | Manual scan (Gitea or GitHub forge) |
| POST | `/api/v1/analyze/all` | API key | ✅ | Bulk scan configured forges |

---

## Onboarding

| Method | Path | Auth | Beta | Purpose |
|--------|------|------|------|---------|
| GET | `/onboard` | none | ✅ | Setup wizard UI |
| GET | `/api/v1/onboard/defaults` | API key | ✅ | Wizard defaults |
| POST | `/api/v1/onboard/test-gitea` | API key | ✅ | Test Gitea token |
| POST | `/api/v1/onboard/test-ai` | API key | ✅ | Test AI provider |
| POST | `/api/v1/onboard/repos` | API key | ✅ | List repos for token |
| POST | `/api/v1/onboard/webhooks` | API key | ✅ | Register webhooks |

---

## Dashboard / repos / scans / findings

| Method | Path | Auth | Beta | Purpose |
|--------|------|------|------|---------|
| GET | `/api/v1/dashboard/summary` | API key | ✅ | Operator dashboard JSON |
| GET | `/api/v1/repos` | API key | ✅ | List repositories |
| GET | `/api/v1/repos/:id` | API key | ✅ | Repository detail |
| GET | `/api/v1/repos/:id/settings` | API key | ✅ | Per-repo settings |
| PUT | `/api/v1/repos/:id/settings` | API key | ✅ | Update repo settings |
| GET | `/api/v1/repos/:id/scans` | API key | ✅ | Scan history |
| GET | `/api/v1/repos/:id/findings` | API key | ✅ | Findings for repo |
| GET | `/api/v1/repos/:id/graph` | API key | ✅ | Repository map JSON |
| GET | `/api/v1/repos/:id/graph/export` | API key | ✅ | Graph export |
| GET | `/api/v1/scans/:scan_id` | API key | ✅ | Scan detail |
| GET | `/api/v1/scans/:scan_id/scanner-results` | API key | ✅ | Per-scanner results |
| GET | `/api/v1/scans/:scan_id/graph` | API key | ✅ | Scan-scoped graph |
| GET | `/api/v1/scans/:scan_id/graph/export` | API key | ✅ | Export |
| GET | `/api/v1/findings` | API key | ✅ | List findings |
| GET | `/api/v1/findings/:id` | API key | ✅ | Finding detail |
| GET | `/api/v1/findings/:id/lifecycle` | API key | ✅ | Issue lifecycle |

---

## Suppressions / false positives

| Method | Path | Auth | Beta | Purpose |
|--------|------|------|------|---------|
| POST | `/api/v1/findings/:id/suppress` | API key | ✅ | Suppress finding |
| POST | `/api/v1/findings/:id/mark-false-positive` | API key | ✅ | Mark false positive |
| POST | `/api/v1/suppressions` | API key | ✅ | Create suppression rule |
| GET | `/api/v1/suppressions` | API key | ✅ | List rules |
| POST | `/api/v1/suppressions/:id/disable` | API key | ✅ | Disable rule |
| GET | `/api/v1/analytics/scan-quality` | API key | ✅ | Scan quality report |

---

## Reconciliation

| Method | Path | Auth | Beta | Purpose |
|--------|------|------|------|---------|
| GET | `/api/v1/repos/:id/reconcile-issues/preview` | API key | ✅ | Preview reconcile |
| POST | `/api/v1/repos/:id/reconcile-issues` | API key | ✅ | Apply reconcile |
| GET | `/api/v1/issues/reconciliation/:run_id` | API key | ✅ | Run status |

---

## Calibration

| Method | Path | Auth | Beta | Purpose |
|--------|------|------|------|---------|
| GET | `/api/v1/calibration/summary` | API key | ✅ | Calibration summary |
| GET | `/api/v1/calibration/recommendations` | API key | ✅ | List recommendations |
| POST | `/api/v1/calibration/recommendations/:id/accept` | API key | ✅ | Accept |
| POST | `/api/v1/calibration/recommendations/:id/reject` | API key | ✅ | Reject |
| POST | `/api/v1/calibration/recompute` | API key | ✅ | Recompute stats |

---

## Remediation

| Method | Path | Auth | Beta | Purpose |
|--------|------|------|------|---------|
| GET | `/api/v1/findings/:id/remediation` | API key | ✅ | Get plan |
| POST | `/api/v1/findings/:id/remediation/generate` | API key | ✅ | Generate plan |
| GET | `/api/v1/remediation/:plan_id` | API key | ✅ | Plan detail |
| POST | `/api/v1/remediation/:plan_id/approve` | API key | ✅ | Approve |
| POST | `/api/v1/remediation/:plan_id/reject` | API key | ✅ | Reject |
| POST | `/api/v1/remediation/:plan_id/attempt-pr` | API key | ⚠️ | Create PR — **off by default** (`remediation_pr_enabled: false`) |
| GET | `/api/v1/remediation/:plan_id/patch-attempts` | API key | ⚠️ | Patch attempts |
| GET | `/api/v1/patch-attempts/:attempt_id` | API key | ⚠️ | Attempt detail |

---

## Evidence closure

| Method | Path | Auth | Beta | Purpose |
|--------|------|------|------|---------|
| GET | `/api/v1/findings/:id/closure-evidence` | API key | ✅ | Closure evidence |
| POST | `/api/v1/findings/:id/verify-closure` | API key | ✅ | Verify after fix |
| POST | `/api/v1/findings/:id/record-direct-remediation` | API key | ✅ | Record merge SHA |
| POST | `/api/v1/patch-attempts/:attempt_id/check-merge` | API key | ✅ | Check merge state |

---

## Pre-install audit

Requires `preinstall_audit_enabled: true` (default **false** in beta).

| Method | Path | Auth | Beta | Purpose |
|--------|------|------|------|---------|
| POST | `/api/v1/preinstall/audits` | API key | ✅ | Start audit |
| GET | `/api/v1/preinstall/audits` | API key | ✅ | List audits |
| GET | `/api/v1/preinstall/audits/:audit_id` | API key | ✅ | Audit detail |
| GET | `/api/v1/preinstall/audits/:audit_id/findings` | API key | ✅ | Audit findings |
| GET | `/api/v1/preinstall/audits/:audit_id/reports` | API key | ✅ | Reports |
| GET | `/api/v1/preinstall/reports/:report_id` | API key | ✅ | Report body |
| POST | `/api/v1/preinstall/reports/:report_id/mark-reviewed` | API key | ✅ | Mark reviewed |

---

## AI

| Method | Path | Auth | Beta | Purpose |
|--------|------|------|------|---------|
| GET | `/api/v1/ai/status` | API key | ✅ | Provider status (no secrets) |
| POST | `/api/v1/ai/test-connection` | API key | ✅ | Manual connection test |

---

## Notifications

Requires `notifications_enabled: true` (default **false**).

| Method | Path | Auth | Beta | Purpose |
|--------|------|------|------|---------|
| POST | `/api/v1/notifications/test` | API key | ✅ | Test webhook |
| GET | `/api/v1/notifications/status` | API key | ✅ | Notification config status |

---

## Runner delegation

Requires `runner_delegation_enabled: true` (default **false**).

**Operator** (`/api/v1/runner/*`, API key):

| Method | Path | Purpose |
|--------|------|---------|
| GET | `/api/v1/runner/jobs` | List jobs |
| GET | `/api/v1/runner/jobs/:job_id` | Job detail |
| POST | `/api/v1/runner/jobs/:job_id/cancel` | Cancel job |

**Worker** (`/api/v1/runner/*`, HMAC):

| Method | Path | Purpose |
|--------|------|---------|
| POST | `/api/v1/runner/jobs/claim` | Claim job |
| GET | `/api/v1/runner/jobs/:job_id/spec` | Job spec |
| POST | `/api/v1/runner/jobs/:job_id/result` | Submit result |

---

## UI routes (`/ui`)

Browser UI auth depends on `auth_mode`:

| Mode | UI auth |
|------|---------|
| `api_key_only` (default) | Global API key in query/header |
| `local` | Signed session cookie; API key not required in browser |

POST forms use CSRF in both modes. JSON API routes always use API key headers (no CSRF).

### Auth routes (`auth_mode=local`)

| Method | Path | Auth | Purpose |
|--------|------|------|---------|
| GET | `/ui/bootstrap` | Public (no users yet) | First owner setup |
| POST | `/ui/bootstrap` | Public + CSRF | Create owner |
| GET | `/ui/login` | Public | Sign-in page |
| POST | `/ui/login` | Public + CSRF | Sign in |
| POST | `/ui/logout` | Session + CSRF | Sign out |

See [AUTH_LOCAL.md](AUTH_LOCAL.md).

### Operator pages

| Path | Beta | Purpose |
|------|------|---------|
| `/ui` | ✅ | Dashboard |
| `/ui/repos` | ✅ | Repositories |
| `/ui/repos/:id` | ✅ | Repo detail |
| `/ui/repos/:id/settings` | ✅ | Repo settings |
| `/ui/repos/:id/graph` | ✅ | Repository map |
| `/ui/repos/:id/reconcile` | ✅ | Reconciliation |
| `/ui/scans` | ✅ | Scans |
| `/ui/findings` | ✅ | Findings |
| `/ui/findings/:id` | ✅ | Finding + remediation + closure |
| `/ui/preinstall` | ⚠️ | When `preinstall_audit_enabled` |
| `/ui/health` | ✅ | System health |

---

## Related

- [CONFIGURATION.md](CONFIGURATION.md)
- [ONBOARDING.md](ONBOARDING.md)
- [FEATURE_COMPLETENESS_AUDIT.md](FEATURE_COMPLETENESS_AUDIT.md)
