# Configuration reference

**Repository Detective** — Inspect. Analyze. Improve.

Configuration merges three layers (highest wins where applicable):

```text
1. Environment variables (REPOSITORY_DETECTIVE_* only)
2. config/config.yaml (non-secret settings)
3. Built-in defaults in config/config.yaml.example
```

Secrets (**API key**, forge tokens, AI keys, webhook secret) belong in **`.env`** only — not committed.

---

## Files

| File | Purpose |
|------|---------|
| `.env` | Secrets and env overrides (from `.env.example`) |
| `config/config.yaml` | Operator settings (from `config/config.yaml.example`) |
| `config/config.yaml.example` | Documented defaults including beta profile |

Docker Compose loads `.env` via `env_file`.

---

## Minimum variables (.env)

```bash
REPOSITORY_DETECTIVE_API_KEY=
REPOSITORY_DETECTIVE_GITEA_URL=https://git.example.com
REPOSITORY_DETECTIVE_GITEA_TOKEN=
REPOSITORY_DETECTIVE_WEBHOOK_SECRET=
REPOSITORY_DETECTIVE_PUBLIC_URL=          # required for webhooks from external Gitea
REPOSITORY_DETECTIVE_SKIP_STARTUP_CHECKS=true
```

At least one forge token: Gitea and/or GitHub (see `.env.example`).

---

## API authentication

**Preferred header:**

```http
X-Repository-Detective-API-Key: <same as REPOSITORY_DETECTIVE_API_KEY>
```

Legacy `X-Repository-Detective-API-Key` and `?api_key=` (UI homelab only) still accepted.

### Local admin auth (optional slice 1)

Default: `auth_mode: api_key_only` (no behavior change).

| Key | Default | Notes |
|-----|---------|-------|
| `auth_mode` | `api_key_only` | Set `local` for session login on UI |
| `session_cookie_name` | `rd_session` | HttpOnly signed cookie |
| `session_secret` | `""` | **Required** when `auth_mode=local` |
| `session_ttl_hours` | `12` | Session lifetime |
| `csrf_enabled` | `true` | UI form POST protection |
| `local_admin_bootstrap_enabled` | `true` | First-owner setup at `/ui/bootstrap` |

Env: `REPOSITORY_DETECTIVE_AUTH_MODE`, `REPOSITORY_DETECTIVE_SESSION_SECRET`, `REPOSITORY_DETECTIVE_SESSION_TTL_HOURS`, `REPOSITORY_DETECTIVE_CSRF_ENABLED` (legacy `REPOSITORY_DETECTIVE_*` aliases supported).

Full guide: [AUTH_LOCAL.md](AUTH_LOCAL.md).

---

## Key YAML settings

| Key | Beta default | Notes |
|-----|--------------|-------|
| `scan_profile` | `standard` | [SCAN_PROFILES.md](SCAN_PROFILES.md) — Light / Standard / Deep / Custom |
| `enable_llm_auditors` | `false` | Deterministic-first |
| `remediation_pr_enabled` | `false` | Safe PRs off until operator enables |
| `evidence_closure_close_issues` | `false` | Comments only |
| `preinstall_audit_enabled` | `true` | Pre-install audit on-ramp (report-only; no issue filing) |
| `ai_startup_test_enabled` | `false` | No paid probe on boot |
| `database_path` | `./data/repository-detective.db` | Legacy filename intentional |
| `label_compat_mode` | `new_only` | Writes `repository-detective/*` labels |
| `auth_mode` | `api_key_only` | `local` enables UI session login |

Full example: `config/config.yaml.example`.

---

## Per-repository overrides

Stored in SQLite `repo_settings` — override global policy per repo from UI or API. See [POLICY.md](POLICY.md).

---

## Scan profiles

Set globally or per repo. Recommended day-to-day: **`standard`**.

---

## Precedence rules

| Situation | Winner |
|-----------|--------|
| `REPOSITORY_DETECTIVE_PORT` and `REPOSITORY_DETECTIVE_PORT` both set | `REPOSITORY_DETECTIVE_*` |
| YAML `port` and env port | Env |
| Repo setting vs global | Repo when set |

---

## Config matrix (audit)

Beta-critical keys with defaults and edition notes: [FEATURE_COMPLETENESS_AUDIT.md](FEATURE_COMPLETENESS_AUDIT.md#part-c--config-matrix-beta-critical-keys).

## Related docs

- [POLICY.md](POLICY.md) — reporting and gates
- [SCAN_PROFILES.md](SCAN_PROFILES.md)
- [AI_PROVIDERS.md](AI_PROVIDERS.md)
- [BETA_READINESS.md](BETA_READINESS.md)
- [BRANDING_MIGRATION.md](BRANDING_MIGRATION.md)

---

See also [Home](Home).
