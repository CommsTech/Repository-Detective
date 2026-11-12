# Local database (Phase 5)

Repository Detective can persist control-plane state in a local SQL database. **Gitea remains the source of truth for issues**; the database stores history, settings, fingerprints, scans, and lifecycle events for the operator UI and scheduled scans.

Per-repo settings in `repo_settings` are merged with global config on each scan. See [POLICY.md](POLICY.md).

## Defaults

| Setting | Default | Env var |
|---------|---------|---------|
| Enabled | `true` | `REPOSITORY_DETECTIVE_DATABASE_ENABLED` |
| Driver | `sqlite` | `REPOSITORY_DETECTIVE_DATABASE_DRIVER` |
| Path | `./data/repository-detective.db` | `REPOSITORY_DETECTIVE_DATABASE_PATH` |
| DSN | empty | `REPOSITORY_DETECTIVE_DATABASE_DSN` |

SQLite is the homelab default. The parent directory (`./data/`) is created automatically on startup.

## Disable the database

Set in `config/config.yaml`:

```yaml
database_enabled: false
```

Or:

```text
REPOSITORY_DETECTIVE_DATABASE_ENABLED=false
```

When disabled, Repository-Detective behaves as before Phase 5 — no local persistence, no schema creation.

If `database_enabled=true` and initialization fails, **startup fails**.

## Backup

For SQLite, back up the database file while Repository-Detective is stopped (or use SQLite backup API):

```bash
cp ./data/repository-detective.db ./backups/repository-detective-$(date +%F).db
```

Store backups with your usual homelab backup rotation.

## What is stored locally

| Table | Purpose |
|-------|---------|
| `repositories` | Connected forge repos seen by webhooks/manual scans |
| `repo_settings` | Per-repo policy/scanner overrides (Phase 6 UI will edit these) |
| `scans` | Scan runs (trigger, ref, status, summary) |
| `scanner_results` | Per-scanner outcomes per scan |
| `findings` | Deduplicated findings by fingerprint |
| `finding_instances` | Per-scan evidence snapshots |
| `external_issues` | Local index of Gitea issue numbers/URLs |
| `lifecycle_events` | Issue/finding lifecycle history |

## What stays in Gitea

- Issue titles, bodies, comments, labels
- Commit status checks
- PRs and merges

The local DB **indexes** forge issues; it does not replace them.

## Per-repo settings (Phase 5)

`repo_settings` rows can be saved via the store API. **Runtime scan behavior still uses global `config.yaml` / env vars** until Phase 6 wires the settings resolver into scan paths.

Use `store.ResolveRepoSettings(global, repoSettings)` to preview effective settings.

## Operator UI (Phase 6)

When `database_enabled=true` and `ui_enabled=true`, open `/ui?api_key=YOUR_KEY` to view repositories, scans, findings, and edit repo settings. See [UI.md](UI.md).

Per-repo settings are stored in `repo_settings` but **do not yet override scan behavior** — global config remains active until a later phase.

## PostgreSQL (future)

Phase 5 implements SQLite only. The `store.Store` interface and `database_driver` / `database_dsn` keys are reserved for a future PostgreSQL backend (`database_driver: postgres`).

## Rollback

1. Set `database_enabled: false` and restart — no code revert required.
2. Or revert Phase 5 commits and delete `./data/repository-detective.db` if you no longer need history.

## Migrations

Schema version is tracked in `schema_migrations`. Migrations run idempotently on startup. Current version: **6**.

### Phase 10B — per-repo health settings

Migration v3 adds nullable health columns to `repo_settings`:

| Column | Type | Meaning |
|--------|------|---------|
| `enable_health_checks` | INTEGER (nullable) | Master health toggle |
| `enable_tech_debt_checks` | INTEGER | Tech debt category |
| `enable_reliability_checks` | INTEGER | Reliability category |
| `enable_maintainability_checks` | INTEGER | Maintainability category |
| `enable_test_gap_checks` | INTEGER | Test gap category |
| `enable_performance_checks` | INTEGER | Performance category |
| `enable_ai_risk_checks` | INTEGER | AI-risk category (off by default globally) |
| `health_max_findings` | INTEGER | Cap on health findings per scan |
| `health_large_file_lines` | INTEGER | Large-file threshold |
| `health_large_function_lines` | INTEGER | Large-function threshold |
| `health_max_nesting_depth` | INTEGER | Nesting depth threshold |
| `health_max_function_params` | INTEGER | Parameter count threshold |

NULL means inherit global config. Existing rows are unchanged (inherit behavior).

### Phase 11 — code graph storage

| Table | Purpose |
|-------|---------|
| `scan_graphs` | JSON repository map per connected-repo scan |
| `audit_graphs` | JSON repository map per pre-install audit |

See [CODE_GRAPH.md](CODE_GRAPH.md).

### Phase 11B — per-repo graph settings

Migration v5 adds nullable graph columns to `repo_settings`:

| Column | Type | Meaning |
|--------|------|---------|
| `enable_code_graph` | INTEGER (nullable) | Master code graph toggle |
| `graph_max_nodes` | INTEGER | Max nodes before truncation (100–50000) |
| `graph_max_edges` | INTEGER | Max edges before truncation (100–200000) |
| `graph_timeout_seconds` | INTEGER | Graph build timeout (5–1800) |
| `graph_include_functions` | INTEGER | Include function nodes |
| `graph_include_findings` | INTEGER | Overlay finding nodes |

NULL means inherit global config. Scan `effective_settings` / policy snapshots include resolved graph settings. Pre-install audits continue to use global graph config only.

### Phase 12 — runner delegation storage

Migration v6 adds `runner_jobs`, `runner_artifacts`, and `runner_nonces`. See [RUNNERS.md](RUNNERS.md).

### Phase 9 tables (pre-install audit)

| Table | Purpose |
|-------|---------|
| `audit_requests` | Third-party URL audit jobs (status, risk score, recommendation) |
| `audit_findings` | Findings from pre-install audits (separate from connected-repo `findings`) |
| `disclosure_reports` | Copy/paste markdown report drafts |

See [PREINSTALL_AUDIT.md](PREINSTALL_AUDIT.md).
