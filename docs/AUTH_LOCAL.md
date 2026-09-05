# Local admin authentication (slice 1)

**Product:** Repository Detective — Inspect. Analyze. Improve.  
**Status:** Implemented (slice 1)  
**Date:** 2026-06-05

Slice 1 adds **optional local admin login** with secure browser sessions while keeping the existing **API-key** model for automation and private beta compatibility.

---

## Recommended new installs (RD-010)

For **new** Community installs, prefer:

```yaml
auth_mode: local
session_secret: "<long random>"
database_enabled: true
csrf_enabled: true
local_admin_bootstrap_enabled: true
```

Then open `/ui/bootstrap` to create the first owner account.

**Existing** deployments keep working on `api_key_only` until you opt in. The runtime default remains `api_key_only` so upgrades never lock operators out.

Automation (scripts, MCP, CI) continues to use `X-Repository-Detective-API-Key` in both modes.

## Auth modes

| Mode | Config | UI | API |
|------|--------|----|-----|
| `api_key_only` | **Default** | Global API key (query/header) | API key headers |
| `local` | Requires `session_secret` | Session cookie login | API key headers unchanged |

Rollback: set `auth_mode: api_key_only` and restart. Session tables remain but are unused.

---

## Configuration

```yaml
auth_mode: api_key_only          # api_key_only | local
session_cookie_name: rd_session
session_secret: ""               # required when auth_mode=local
session_ttl_hours: 12
csrf_enabled: true
local_admin_bootstrap_enabled: true
```

### Environment variables

| Key | Legacy alias |
|-----|----------------|
| `REPOSITORY_DETECTIVE_AUTH_MODE` | `REPOSITORY_DETECTIVE_AUTH_MODE` |
| `REPOSITORY_DETECTIVE_SESSION_SECRET` | `REPOSITORY_DETECTIVE_SESSION_SECRET` |
| `REPOSITORY_DETECTIVE_SESSION_TTL_HOURS` | `REPOSITORY_DETECTIVE_SESSION_TTL_HOURS` |
| `REPOSITORY_DETECTIVE_CSRF_ENABLED` | `REPOSITORY_DETECTIVE_CSRF_ENABLED` |

`REPOSITORY_DETECTIVE_*` wins when both prefixes are set (see `internal/config/envcompat`).

### Startup validation

- `auth_mode=local` **fails startup** if `session_secret` is empty.
- `auth_mode=local` requires `database_enabled: true`.
- Default remains `api_key_only` for existing deployments.

---

## Bootstrap flow

When `auth_mode=local`, `local_admin_bootstrap_enabled=true`, and the `users` table is empty:

1. Open `GET /ui/bootstrap`
2. Create the first **owner** account (strong password required)
3. Receive a signed session cookie and redirect to the dashboard
4. Bootstrap is **permanently disabled** once any user exists

Routes:

| Method | Path | Auth |
|--------|------|------|
| GET | `/ui/bootstrap` | Public (only when no users) |
| POST | `/ui/bootstrap` | Public + CSRF |
| GET | `/ui/login` | Public |
| POST | `/ui/login` | Public + CSRF |
| POST | `/ui/logout` | Session + CSRF |

Login errors use **generic messaging** (no email-existence leak).

---

## Sessions

- Signed session ID cookie (`session_secret` + HMAC)
- **HttpOnly**, **SameSite=Lax**
- **Secure** when `public_url` starts with `https://`
- Expires per `session_ttl_hours`
- Logout deletes the server-side session row and clears the cookie
- New session ID on each login (rotation)

Database tables (migration 17): `users`, `sessions`, `auth_audit_events`.

### Roles (slice 1)

| Role | Purpose |
|------|---------|
| `owner` | Bootstrap account; full control (RBAC enforcement in later slices) |
| `admin` | Operator admin (created manually in DB today) |
| `read_only` | Read-only (enforcement in later slices) |

Passwords: bcrypt cost 12, minimum 12 characters with letters and numbers.

---

## CSRF

| Context | CSRF required? |
|---------|----------------|
| UI form POST in `local` mode | Yes (when `csrf_enabled: true`) |
| UI form POST in `api_key_only` mode | Yes (API-key-derived token) |
| API JSON with `X-Repository-Detective-API-Key` or `X-Repository-Detective-API-Key` | **No** |

Protected UI POSTs include settings, suppressions, remediation, patch PR, notifications test, reconciliation, pre-install actions.

---

## API compatibility

Automation and scripts **do not change**:

- **Preferred:** `X-Repository-Detective-API-Key`
- **Legacy:** `X-Repository-Detective-API-Key`, `Authorization: Bearer`, `?api_key=`

Local auth affects **browser UI routes only**.

---

## Quick enable (homelab)

```bash
# .env
REPOSITORY_DETECTIVE_AUTH_MODE=local
REPOSITORY_DETECTIVE_SESSION_SECRET=$(openssl rand -hex 32)
```

Restart, open `/ui/bootstrap`, create owner, sign in. Keep `REPOSITORY_DETECTIVE_API_KEY` for CI and curl.

---

## Known limitations (slice 1)

- No per-route RBAC enforcement yet (roles stored; checks in slice 2)
- No OIDC, API tokens per user, or tenant isolation
- No billing or license enforcement
- Admin user management UI not yet implemented (bootstrap + DB only)
- Break-glass global API key still works in `local` mode for API clients

**Next slice:** role checks on handlers, admin user CRUD, session listing/revocation, optional API token table.
