# Configuration reference

**Repository Detective** — Inspect. Analyze. Improve.

Configuration merges three layers (highest wins where applicable):

```text
1. Environment variables (REPOSITORY_DETECTIVE_* preferred; BUGBOT_* legacy)
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

Legacy `X-Bugbot-API-Key` and `?api_key=` (UI homelab only) still accepted.

---

## Key YAML settings

| Key | Beta default | Notes |
|-----|--------------|-------|
| `scan_profile` | `beta_standard` | [SCAN_PROFILES.md](SCAN_PROFILES.md) |
| `enable_llm_auditors` | `false` | Deterministic-first |
| `qdrant_enabled` | `false` | Local/redacted only when enabled |
| `remediation_pr_enabled` | `false` | Safe PRs off until operator enables |
| `evidence_closure_close_issues` | `false` | Comments only |
| `preinstall_audit_enabled` | `false` | Enable on-demand for third-party audits |
| `ai_startup_test_enabled` | `false` | No paid probe on boot |
| `database_path` | `./data/bugbot.db` | Legacy filename intentional |
| `label_compat_mode` | `new_only` | Writes `repository-detective/*` labels |

Full example: `config/config.yaml.example`.

---

## Per-repository overrides

Stored in SQLite `repo_settings` — override global policy per repo from UI or API. See [POLICY.md](POLICY.md).

---

## Scan profiles

Set globally or per repo. Private beta: **`beta_standard`**.

---

## Precedence rules

| Situation | Winner |
|-----------|--------|
| `REPOSITORY_DETECTIVE_PORT` and `BUGBOT_PORT` both set | `REPOSITORY_DETECTIVE_*` |
| YAML `port` and env port | Env |
| Repo setting vs global | Repo when set |

---

## Related docs

- [POLICY.md](POLICY.md) — reporting and gates
- [SCAN_PROFILES.md](SCAN_PROFILES.md)
- [AI_PROVIDERS.md](AI_PROVIDERS.md)
- [BETA_READINESS.md](BETA_READINESS.md)
- [BRANDING_MIGRATION.md](BRANDING_MIGRATION.md)
