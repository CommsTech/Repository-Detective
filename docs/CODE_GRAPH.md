# Repository Map / Code Graph (Phase 11 / 11B)

Repository Detective — **Inspect. Analyze. Improve.**

The **Repository Map** is an evidence-backed code graph that shows how a repository is connected: directories, files, packages, imports, functions (where practical), test relationships, and finding overlays.

Unlike a basic knowledge graph built from forge metadata alone, this map is **AST- and import-aware** (Go uses `go/parser`), overlays scan findings, and detects possibly disconnected code.

## What the graph shows

| Layer | Relationships |
|-------|----------------|
| Structural | repo → directories → files |
| Imports | file → package, file → external dependency |
| Symbols | file → function (Go; optional) |
| Findings | finding → file (severity/category coloring) |
| Analysis | orphan files, disconnected packages, suspicious islands |

## Language support (Phase 11)

| Language | Support |
|----------|---------|
| Go | Full import + function parsing via `go/parser` |
| JavaScript/TypeScript | Basic `import` / `require` |
| Python | Basic `import` / `from … import` |
| Other | Directory tree + generic import heuristics |

## Disconnected-code findings

Deterministic graph analysis produces cautious findings (source: `graph`):

| Rule ID | Category | Wording |
|---------|----------|---------|
| `GRAPH-ORPHAN-FILE` | maintainability | Possible disconnected code path |
| `GRAPH-ORPHAN-FUNCTION` | maintainability | Function may be unused |
| `GRAPH-DISCONNECTED-PACKAGE` | architecture | Potentially disconnected package |
| `GRAPH-SUSPICIOUS-ISLAND` | architecture | Isolated cluster with findings |

Findings never claim code was abandoned or AI-written — they recommend human review.

**False-positive expectations:** Orphan/disconnected signals are conservative heuristics. Reflection, dynamic imports, code generation, build tags, and plugin architectures may produce false positives. Treat as review hints, not proof of dead code.

**Beta default (`beta_standard`):** All graph rule IDs above are **report-only** — visible on the dashboard and Repository Map, but not opened as Gitea issues by default. Use global suppressions (see `docs/dogfood-reports/closeout-suppressions.sql`) for legacy issues created before calibration.

## When graphs are generated

| Context | Trigger |
|---------|---------|
| Connected repo scan | `analysis_depth >= 2` and effective `enable_code_graph: true` |
| Pre-install `standard` / `deep` | Global graph config |
| Pre-install `quick` | Skipped |

Graph generation does not execute repo code, install dependencies, or call LLM.

## Configuration

### Global defaults (`config.yaml` / env)

```yaml
enable_code_graph: true
graph_max_nodes: 5000
graph_max_edges: 15000
graph_timeout_seconds: 120
graph_include_functions: true
graph_include_findings: true
```

Env: `REPOSITORY_DETECTIVE_ENABLE_CODE_GRAPH`, `REPOSITORY_DETECTIVE_GRAPH_*`.

### Per-repo overrides (Phase 11B)

Nullable fields on `repo_settings` inherit global when unset:

| Field | Validation |
|-------|------------|
| `enable_code_graph` | bool |
| `graph_max_nodes` | 100–50000 |
| `graph_max_edges` | 100–200000 |
| `graph_timeout_seconds` | 5–1800 |
| `graph_include_functions` | bool |
| `graph_include_findings` | bool |

Configure via `PUT /api/v1/repos/:id/settings` or `/ui/repos/:id/settings`. Scan policy snapshots include resolved graph settings.

Pre-install audits **always** use global graph config (not per-repo overrides).

## UI / API

| Route | Description |
|-------|-------------|
| `GET /ui/repos/:id/graph` | Latest repo map (interactive, offline UI) |
| `GET /ui/scans/:scan_id/graph` | Scan-specific map |
| `GET /api/v1/repos/:id/graph` | JSON graph (latest scan) |
| `GET /api/v1/scans/:scan_id/graph` | JSON graph for scan |
| `GET /api/v1/repos/:id/graph/export` | Download graph JSON (attachment) |
| `GET /api/v1/scans/:scan_id/graph/export` | Download scan graph JSON (attachment) |

Interactive features: pan/zoom, layout modes, node-type filter, disconnected-only filter, findings overlay toggle, entrypoint highlight, node detail panel, stats (nodes/edges/orphans/findings), copy summary, JSON export.

### Offline UI assets (Phase 11B)

Cytoscape.js **3.30.2** is vendored under `ui/static/` and embedded in the Go binary. The graph page does not fetch scripts from a CDN at runtime. See `ui/static/ASSETS.md`.

## Persistence

Tables `scan_graphs` and `audit_graphs` store JSON payloads with node/edge counts. Large repos may truncate with package-level aggregation when limits are exceeded.

## Limitations

- No full semantic call graph for JS/Python yet
- Function orphan detection is conservative (may false-positive on reflection/dynamic calls)
- Pre-install uses global graph config only (no per-repo graph overrides)
- JSON export only — PNG/SVG export not implemented
- Truncated graphs may hide distant dependencies

## Notifications

High/critical graph-related findings follow the same notification rules as other scan findings when notifications are enabled. See [NOTIFICATIONS.md](NOTIFICATIONS.md).

## Rollback

Set `enable_code_graph: false` globally and/or per repo, then restart. Existing graph rows are harmless historical data. To revert migration v5, restore a pre-11B database backup — columns are additive and nullable.
