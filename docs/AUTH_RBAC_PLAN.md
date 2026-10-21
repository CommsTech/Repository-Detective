# Auth / RBAC design plan

**Product:** Repository Detective — Inspect. Analyze. Improve.  
**Status:** Slice 1 implemented — local login, sessions, CSRF, bootstrap (see [AUTH_LOCAL.md](AUTH_LOCAL.md))  
**Date:** 2026-06-05 (updated)

This document describes the migration from **API-key-only homelab mode** to **authenticated multi-user** operation suitable for trusted private beta teams and, later, SaaS.

---

## 1. Current mode (today)

| Aspect | Behavior |
|--------|----------|
| **Identity** | None — all callers share one operator secret |
| **Config** | Single `api_key` in `config.yaml` / `REPOSITORY_DETECTIVE_API_KEY` |
| **API auth** | **Preferred:** `X-Repository-Detective-API-Key`. **Legacy accepted:** `X-Repository-Detective-API-Key`, `Authorization: Bearer`, or `?api_key=` query |
| **UI auth** | Same global API key passed in query string or header |
| **UI CSRF** | HMAC token derived from global API key (`internal/security/csrf.go`) — mitigates form POST abuse when key is in URL |
| **Runner auth** | Separate HMAC on `/api/v1/runner/*` — unchanged by user auth |
| **Forge auth** | Gitea/GitHub tokens are **service credentials**, not user identity |
| **Tenancy** | Single SQLite database; no org/team/repo ACLs |
| **Audit** | Application logs only; no structured security audit table |

**Accepted for private beta:** one trusted operator, homelab Gitea, API key in `.env`.

**Not acceptable for:** multi-analyst teams, customer-facing SaaS, paid self-service onboarding.

---

## 2. Target mode

### 2.1 Identity layers

```text
Human users        → session cookie (browser UI)
Automation/scripts → scoped API tokens (header only, no query string)
Runners            → existing runner HMAC (unchanged)
Forge integration  → service-level Gitea/GitHub tokens (unchanged)
```

### 2.2 Objects

| Object | Purpose |
|--------|---------|
| **User** | Login identity (email/username, password hash, optional OIDC subject) |
| **Session** | Browser session with rotating ID, expiry, IP/UA fingerprint (optional) |
| **API token** | Long-lived automation credential with scopes and optional expiry |
| **Organization** | Top-level tenant boundary (single-org default in private beta) |
| **Team** | Group of users within an org |
| **Membership** | User ↔ org/team with role |
| **Repo grant** | Optional repo-scoped permission override |
| **Audit event** | Append-only security log |

### 2.3 Auth modes (coexist during migration)

| Mode | When |
|------|------|
| `api_key_only` | **Shipped default** — global API key for UI and API |
| `local` | **Shipped slice 1** — session cookie for UI; API key unchanged for API clients |
| `session_and_tokens` | Future — per-user API tokens + scoped automation |
| `oidc_only` | Future enterprise; local password disabled |

Config flag (implemented): `auth_mode: api_key_only | local`

---

## 3. Roles

| Role | Typical holder | Scope |
|------|----------------|-------|
| **owner** | Instance creator | Full org control, billing (future), break-glass |
| **admin** | Platform operator | Users, runners, global settings, all repos |
| **maintainer** | Repo lead | Repo settings, scans, suppressions, remediation approval |
| **security_reviewer** | AppSec / auditor | View findings, approve suppressions, pre-install/disclosure views |
| **developer** | Engineer | View repos, trigger scans, comment; no policy changes |
| **read_only** | Stakeholder | Read dashboards and reports only |
| **billing_admin** | Finance (SaaS) | Billing/subscription only; no repo access |

**Private beta (phase 1):** implement **owner**, **admin**, **security_reviewer**, **read_only** first. Defer **billing_admin** until monetization phase.

---

## 4. Permissions matrix

Permissions are **strings** checked at handler level. Roles are bundles of permissions.

| Permission | owner | admin | maintainer | security_reviewer | developer | read_only | billing_admin |
|------------|:-----:|:-----:|:----------:|:-----------------:|:---------:|:---------:|:-------------:|
| `repos.view` | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | — |
| `scans.run` | ✓ | ✓ | ✓ | — | ✓ | — | — |
| `repo.settings.edit` | ✓ | ✓ | ✓ | — | — | — | — |
| `suppressions.approve` | ✓ | ✓ | ✓ | ✓ | — | — | — |
| `remediation.approve` | ✓ | ✓ | ✓ | ✓ | — | — | — |
| `remediation.pr.create` | ✓ | ✓ | ✓ | — | — | — | — |
| `notifications.manage` | ✓ | ✓ | ✓ | — | — | — | — |
| `runners.manage` | ✓ | ✓ | — | — | — | — | — |
| `users.manage` | ✓ | ✓ | — | — | — | — | — |
| `preinstall.view` | ✓ | ✓ | ✓ | ✓ | — | — | — |
| `disclosures.view` | ✓ | ✓ | ✓ | ✓ | — | — | — |
| `audit.view` | ✓ | ✓ | — | ✓ | — | — | — |
| `org.settings` | ✓ | ✓ | — | — | — | — | — |
| `billing.manage` | ✓ | — | — | — | — | — | ✓ |

**Repo-scoped enforcement:** maintainer/developer/reviewer roles apply per repository via `repo_grants`. Org-wide roles (owner/admin) bypass repo grants.

**Default private beta mapping:**

- Bootstrap user → **owner**
- Trusted analyst → **security_reviewer**
- Developer triggering scans → **developer** on assigned repos only

---

## 5. Local admin bootstrap flow

Runs once when `users` table is empty and `auth_mode` allows bootstrap.

```text
1. Operator starts service with auth_mode=legacy_api_key (default).
2. Operator opens GET /ui/setup (or POST /api/v1/auth/bootstrap) with valid legacy API key.
3. Form collects: display name, email, password (twice), optional org name.
4. Server creates:
   - organization (default slug from hostname or "default")
   - user (owner role, password hash)
   - membership (user_id, org_id, role=owner)
   - session cookie
5. Server sets auth_mode=session_and_tokens in DB (not config file).
6. Legacy global api_key:
   - Option A (recommended): retained as break-glass only, UI shows deprecation banner
   - Option B: hashed and stored as bootstrap API token with owner scopes
7. Bootstrap endpoint disabled permanently (flag in org_settings).
```

**Safety:**

- Bootstrap requires existing legacy API key OR one-time `REPOSITORY_DETECTIVE_BOOTSTRAP_TOKEN` env (single use).
- Rate limit bootstrap attempts.
- Log `auth.bootstrap.completed` audit event.

---

## 6. Password storage requirements

| Requirement | Choice |
|-------------|--------|
| Algorithm | **Argon2id** (preferred) or bcrypt cost ≥ 12 |
| Pepper | Optional server-side pepper from env `AUTH_PASSWORD_PEPPER` |
| Minimum length | 12 characters; reject common-password list (optional Have I Been Pwned k-anonymity later) |
| Storage | Never store plaintext; store `password_hash`, `hash_version`, `updated_at` |
| Reset | Phase 2: email/token reset; Phase 1: admin reset only |
| Lockout | 5 failed attempts / 15 min per IP+username (configurable) |

---

## 7. OIDC future path

Not implemented in phase 1. Design hooks:

```yaml
oidc_enabled: false
oidc_issuer_url: ""
oidc_client_id: ""
oidc_client_secret: ""   # env only
oidc_scopes: "openid profile email"
oidc_allowed_domains: []  # e.g. ["example.com"]
oidc_auto_provision_role: read_only
```

Flow: authorization code + PKCE → map `sub` + `email` to user → session cookie. Local password remains for break-glass if `oidc_allow_local_fallback: true`.

---

## 8. Session cookie security

| Property | Value |
|----------|-------|
| Name | `rd_session` |
| HttpOnly | **true** |
| Secure | **true** when TLS or `trust_proxy_headers` + HTTPS |
| SameSite | **Lax** (Strict for admin-only paths optional) |
| Path | `/` |
| Max-Age | 24h default; sliding refresh on activity |
| Storage | Server-side session row: `session_id` (random 32 bytes), `user_id`, `created_at`, `expires_at`, `last_seen_at`, optional `ip_hash` |
| Rotation | New session ID on login and privilege elevation |
| Logout | Delete server session + clear cookie |

**No sensitive data in cookie** — only opaque session ID.

---

## 9. CSRF protection (session mode)

| Context | Mechanism |
|---------|-----------|
| Browser UI (session) | Double-submit cookie `rd_csrf` (non-HttpOnly) + hidden form field; validated on POST/PUT/DELETE |
| API tokens | No CSRF — `Authorization: Bearer` header only |
| Legacy API key | Keep existing HMAC CSRF until legacy mode removed |

Migrate UI forms from API-key-derived CSRF to session CSRF when logged in.

---

## 10. API token model

| Field | Description |
|-------|-------------|
| `id` | Public prefix e.g. `rd_live_abc123` |
| `secret_hash` | SHA-256 of full secret (shown once at creation) |
| `user_id` | Creator |
| `org_id` | Scope boundary |
| `name` | Human label ("CI scan trigger") |
| `scopes` | JSON array of permission strings |
| `expires_at` | Optional |
| `last_used_at` | Updated throttled |
| `revoked_at` | Soft revoke |

**Transport:** `Authorization: Bearer rd_live_abc123.<secret>` — **never** query string.

**Legacy migration:** existing global `api_key` can be imported as one owner-scoped token with full permissions during bootstrap.

---

## 11. Audit log events

Append-only `audit_events` table. Payload JSON **redacted** (no passwords, tokens, secret evidence).

| Event | When |
|-------|------|
| `auth.login.success` / `auth.login.failure` | Login attempts |
| `auth.logout` | Session ended |
| `auth.bootstrap.completed` | First admin created |
| `auth.password.changed` | Password update |
| `auth.token.created` / `auth.token.revoked` | API token lifecycle |
| `user.invited` / `user.role_changed` / `user.disabled` | User admin |
| `repo.grant.changed` | Repo ACL update |
| `scan.triggered` | Who triggered scan (user or token id) |
| `suppression.approved` / `suppression.rejected` | Policy actions |
| `remediation.approved` / `remediation.pr.created` | Remediation workflow |
| `preinstall.started` / `preinstall.completed` | Third-party audit |
| `settings.changed` | Org/global settings |

Retention: 90 days default (configurable). Export: JSON lines for SIEM (SaaS phase).

---

## 12. Database schema proposal (migration 17+)

New tables (SQLite; additive only):

```sql
-- migration 17
CREATE TABLE organizations (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  slug TEXT NOT NULL UNIQUE,
  display_name TEXT NOT NULL,
  created_at TEXT NOT NULL,
  settings_json TEXT NOT NULL DEFAULT '{}'
);

CREATE TABLE users (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  email TEXT NOT NULL UNIQUE,
  display_name TEXT NOT NULL,
  password_hash TEXT NOT NULL DEFAULT '',
  password_hash_version INTEGER NOT NULL DEFAULT 1,
  oidc_subject TEXT UNIQUE,
  disabled INTEGER NOT NULL DEFAULT 0,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL
);

CREATE TABLE org_memberships (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  org_id INTEGER NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
  user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  role TEXT NOT NULL,
  created_at TEXT NOT NULL,
  UNIQUE(org_id, user_id)
);

CREATE TABLE teams (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  org_id INTEGER NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
  slug TEXT NOT NULL,
  display_name TEXT NOT NULL,
  UNIQUE(org_id, slug)
);

CREATE TABLE team_memberships (
  team_id INTEGER NOT NULL REFERENCES teams(id) ON DELETE CASCADE,
  user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  role TEXT NOT NULL,
  PRIMARY KEY (team_id, user_id)
);

CREATE TABLE repo_grants (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  repository_id INTEGER NOT NULL REFERENCES repositories(id) ON DELETE CASCADE,
  subject_type TEXT NOT NULL,  -- 'user' | 'team'
  subject_id INTEGER NOT NULL,
  role TEXT NOT NULL,
  created_at TEXT NOT NULL,
  UNIQUE(repository_id, subject_type, subject_id)
);

CREATE TABLE sessions (
  id TEXT PRIMARY KEY,
  user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  expires_at TEXT NOT NULL,
  created_at TEXT NOT NULL,
  last_seen_at TEXT NOT NULL,
  ip_hash TEXT NOT NULL DEFAULT ''
);

CREATE TABLE api_tokens (
  id TEXT PRIMARY KEY,
  user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  org_id INTEGER NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
  name TEXT NOT NULL,
  secret_hash TEXT NOT NULL,
  scopes_json TEXT NOT NULL DEFAULT '[]',
  expires_at TEXT,
  last_used_at TEXT,
  revoked_at TEXT,
  created_at TEXT NOT NULL
);

CREATE TABLE audit_events (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  org_id INTEGER,
  actor_user_id INTEGER,
  actor_token_id TEXT,
  event_type TEXT NOT NULL,
  resource_type TEXT NOT NULL DEFAULT '',
  resource_id TEXT NOT NULL DEFAULT '',
  payload_json TEXT NOT NULL DEFAULT '{}',
  ip_hash TEXT NOT NULL DEFAULT '',
  created_at TEXT NOT NULL
);

CREATE INDEX idx_audit_events_org_created ON audit_events(org_id, created_at DESC);
CREATE INDEX idx_sessions_user ON sessions(user_id);
CREATE INDEX idx_api_tokens_user ON api_tokens(user_id);
```

**Org settings JSON** (in `organizations.settings_json`):

```json
{
  "auth_mode": "session_and_tokens",
  "bootstrap_completed_at": "2026-06-04T00:00:00Z",
  "legacy_api_key_enabled": true
}
```

---

## 13. Migration path from single API key mode

| Step | Action | Risk |
|------|--------|------|
| 1 | Ship migration 17 tables; default `auth_mode=legacy_api_key` | None — additive |
| 2 | Add bootstrap UI + login pages; no behavior change until used | Low |
| 3 | Operator bootstraps owner account | Low |
| 4 | Dual auth middleware: accept legacy key **or** session **or** scoped token | Medium — test all routes |
| 5 | UI: login page default; legacy `?api_key=` shows deprecation notice | Low |
| 6 | Issue per-user/per-automation API tokens; document header-only | Low |
| 7 | Enforce repo grants for non-admin roles | Medium |
| 8 | Optional: disable legacy global key (`legacy_api_key_enabled: false`) | High — lockout if no tokens |

**Rollback:** set `auth_mode=legacy_api_key` in org settings; disable session middleware; global key continues to work. Session tables remain but unused.

---

## 14. Private beta vs SaaS requirements

| Requirement | Private beta | SaaS |
|-------------|:------------:|:----:|
| Local login (1–5 users) | ✓ phase 1 | ✓ |
| Session cookies + CSRF | ✓ phase 1 | ✓ |
| API tokens with scopes | ✓ phase 2 | ✓ |
| Repo-level RBAC | ✓ phase 2 | ✓ |
| Org/team model | Minimal (single org) | Full multi-org |
| OIDC / SSO | Optional | Required |
| Tenant isolation (DB) | Single DB acceptable | Separate schema or DB per tenant |
| Billing integration | No | Yes |
| Audit log export | Optional | Required |
| Rate limits per user | Nice | Required |
| Password reset email | Admin-only OK | Self-service required |

**Private beta impact:** Auth/RBAC can ship **without blocking** current beta week if `auth_mode` stays `legacy_api_key` until beta ends. Enable sessions only when ready to onboard a second analyst.

---

## 15. Rollback plan

1. **Config:** `auth_mode: legacy_api_key` (or org setting override).
2. **Code flag:** `AUTH_ENABLED=false` env skips session middleware entirely (implementation detail).
3. **Database:** Migrations 17+ are additive; no downgrade required — unused tables are harmless.
4. **UI:** Login routes hidden when auth disabled; existing API-key UI paths unchanged.
5. **Documentation:** Operator reverts to `.env` global API key; revoke issued API tokens if compromised.

---

## 16. Recommended implementation phases

| Phase | Deliverable | Est. scope |
|-------|-------------|------------|
| **0** | This design doc + schema review | Done |
| **1** | Migration 17, password hash util, bootstrap + login + logout, session middleware, login UI | **Done** (2026-06-05) |
| **2** | API token CRUD, dual auth middleware, deprecate query-string key in UI | Small |
| **3** | Permission checks on scan/settings/suppression/remediation handlers | Medium |
| **4** | Repo grants + team model + admin user UI | Medium |
| **5** | Audit log write path + viewer UI | Small |
| **6** | OIDC provider (optional) | Medium |
| **7** | Tenant isolation + billing hooks | Large (SaaS) |

**Do not implement phases 6–7 until private beta week feedback is collected.**

---

## 17. Risks

| Risk | Mitigation |
|------|------------|
| Lockout during migration | Keep legacy API key until bootstrap + token issued; break-glass env token |
| Session fixation | Rotate session ID on login |
| CSRF bypass on mixed auth | Clear rules: session → double-submit; token → header only |
| SQLite concurrency under multi-user | Acceptable for private beta; WAL mode; SaaS may need Postgres |
| Permission sprawl | Start with 14 permissions; avoid per-scanner ACLs |
| Query-string API key leakage | Deprecate in UI; log warning; remove in phase 2 |
| Audit log PII | Hash IPs; redact payloads; no forge tokens in events |

---

## 18. Files to touch (implementation reference — not now)

| Area | Files |
|------|-------|
| Migrations | `store/migrations.go` |
| Auth store | `store/auth_*.go` (new) |
| Middleware | `internal/auth/middleware.go` (new), `main.go` |
| Handlers | `api/auth_handler.go`, `ui/login_handlers.go` (new) |
| Templates | `ui/templates/login.html`, `setup.html` |
| Security | extend `internal/security/csrf.go` for session CSRF |
| Config | `auth_mode`, session TTL in `config.yaml.example` |

---

## Related docs

- [BETA_READINESS.md](BETA_READINESS.md)
- [SECURITY_HARDENING.md](SECURITY_HARDENING.md)
- [MONETIZATION_READINESS.md](MONETIZATION_READINESS.md)
- [EDITIONS.md](EDITIONS.md)
- [BRANDING_COMPATIBILITY_AUDIT.md](BRANDING_COMPATIBILITY_AUDIT.md)
- [PRIVACY.md](PRIVACY.md)
