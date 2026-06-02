# Product naming

| Name | Usage |
|------|--------|
| **Repository Detective** | Product/platform name — docs, UI headings, reports, new issue bodies |
| **Bugbot** | Legacy/internal service name — compatibility for env vars, labels, fingerprints, DB path |
| **Tagline** | Inspect. Analyze. Improve. |

Repository Detective is the product name. Bugbot remains supported for existing deployments during the transition.

## Phase 12B (implemented)

Branding compatibility migration is **implemented**. See [BRANDING_MIGRATION.md](BRANDING_MIGRATION.md) for operator details.

| Feature | Status |
|---------|--------|
| Dual env vars (`BUGBOT_*` + `REPOSITORY_DETECTIVE_*`) | Shipped |
| Label compat modes (`dual`, `legacy_only`, `new_only`) | Shipped |
| Dual fingerprint body markers | Shipped |
| Fingerprint values (`bugbot-<hex>`) | Unchanged |
| API header alias | Shipped |
| `/api/v1/about` | Shipped |
| DB table / API path renames | Not in scope |

## Environment variables

**Prefer** `REPOSITORY_DETECTIVE_*` for new deployments.

**Legacy** `BUGBOT_*` variables remain fully supported. If both are set for the same key, `REPOSITORY_DETECTIVE_*` wins.

## Issue labels

Default write mode: **`new_only`** — new issues receive `repository-detective`, `repository-detective/*` category and lifecycle labels, `severity/*`, and `automated-review`.

Legacy `bugbot/*` labels are no longer written in `dual` mode (lookup still searches both for existing issues). Use `label_compat_mode: legacy_only` only for rollback.

Configure with `label_compat_mode` in `config.yaml` or `REPOSITORY_DETECTIVE_LABEL_COMPAT_MODE`.

## Fingerprints

New issue bodies use `Repository Detective fingerprint:` but values remain `bugbot-<hex>`.

## Scanner expansion

Deterministic-first scanner roadmap: [SCANNER_ROADMAP.md](SCANNER_ROADMAP.md).

CLI / runner binary: `repository-detective-runner` (Phase 12).
