# Branding migration (Phase 12B)

Repository Detective — **Inspect. Analyze. Improve.**

This document describes the compatibility migration from **Bugbot** (legacy/internal) to **Repository Detective** (product name).

## Migration rule

```text
Read old and new names.
Write new names by default.
Never break existing scans, issues, labels, env vars, DB records, or runner workflows.
```

## Product vs legacy

| Surface | Name |
|---------|------|
| Product / UI / docs / new issues | **Repository Detective** |
| Legacy compatibility | **Bugbot** |
| Tagline | Inspect. Analyze. Improve. |

## Environment variables

Both prefixes are supported:

```text
BUGBOT_*
REPOSITORY_DETECTIVE_*
```

| Rule | Behavior |
|------|----------|
| Precedence | `REPOSITORY_DETECTIVE_*` wins when both are set for the same key |
| Legacy only | `BUGBOT_*` still works; one-time deprecation notice at startup |
| YAML keys | Unchanged (`enable_trivy`, `api_key`, etc.) |

Examples:

```bash
REPOSITORY_DETECTIVE_API_KEY=...
REPOSITORY_DETECTIVE_GITEA_URL=...
REPOSITORY_DETECTIVE_RUNNER_SHARED_SECRET=...
```

Runner binary also accepts `REPOSITORY_DETECTIVE_CORE_URL`, `REPOSITORY_DETECTIVE_WORKSPACE`, etc.

## API authentication

Both headers are accepted (new first):

```text
X-Repository-Detective-API-Key
X-Bugbot-API-Key
```

Query parameter `api_key` still works for UI links.

## Gitea issue labels

Config: `label_compat_mode` (env: `REPOSITORY_DETECTIVE_LABEL_COMPAT_MODE` / `BUGBOT_LABEL_COMPAT_MODE`)

| Mode | Write behavior | Read behavior |
|------|----------------|---------------|
| `dual` (default) | Both `bugbot/*` and `repository-detective/*` | Recognizes both |
| `legacy_only` | `bugbot/*` only | Recognizes both |
| `new_only` | `repository-detective/*` only | Recognizes both |

Issue lookup searches issues labeled `bugbot` **or** `repository-detective`.

## Issue fingerprints

| Item | Behavior |
|------|----------|
| Body marker (new) | `Repository Detective fingerprint:` |
| Body marker (legacy) | `Bugbot fingerprint:` — still parsed |
| Fingerprint value | **`bugbot-<hex>` unchanged** — do not rename |

Fingerprint values retain the historical `bugbot-` prefix for compatibility. Changing the prefix would break deduplication and lifecycle tracking.

## API routes

Existing `/api/v1/*` routes are unchanged.

New informational endpoint:

```http
GET /api/v1/about
```

Returns product name, legacy name, tagline, and compatibility flags.

## Database

No table renames in Phase 12B. Internal SQLite path default remains `./data/bugbot.db`.

## Rollback

1. Set `label_compat_mode: legacy_only`
2. Continue using `BUGBOT_*` env vars
3. Existing Gitea issues remain readable (fingerprints and legacy labels unchanged)
4. No DB schema rollback required

## Related documents

- [NAMING.md](NAMING.md)
- [PHASE_12B_BRANDING_PLAN.md](PHASE_12B_BRANDING_PLAN.md)
- [RUNNERS.md](RUNNERS.md)
