# Graph and pre-install sandbox baseline

Recorded: 2026-06-02 (sprint start, pre-fix deploy)

## Repository state

| Item | Value |
|------|-------|
| Git HEAD | `da908bf` |
| Live container image | `repository-detective:all-in-one` (built from `ca56dbf` era) |
| Product repo ID | 1 (`commstech/Repository-Detective`) |
| Latest product scan ID | `b21dc57c40411f31` |

## Graph page (pre-fix)

- URL tested: `http://localhost:8081/ui/scans/b21dc57c40411f31/graph` (401 without session/API key)
- **Known UX bug:** contradictory static/loading messages on Repository Map:
  - "No graph was stored for this scan…"
  - "Loading graph summary…"
  - "Graph not available for this scan or repository"
- Root cause: template always rendered loading text; JS could show missing + API error simultaneously.

## Graph API (live, pre state-model deploy)

- `GET /api/v1/scans/b21dc57c40411f31/graph` → **200** with legacy payload (`scan_id`, `nodes`, `edges`, `metrics`) — no `state` field yet.
- DB `scan_graphs` row exists for latest scan: **3627 nodes**, **5964 edges** (graph data persisted; UI state model was wrong).

## Graph configuration

| Setting | Value |
|---------|-------|
| `enable_code_graph` (global default) | true |
| `analysis_depth` (global default) | 3 (≥ 2) |
| Graph generation | enabled for standard/deep profiles when depth ≥ 2 |

## Pre-install clone / scan isolation (pre-hardening)

| Control | Baseline |
|---------|----------|
| Dedicated temp workspace per audit | partial (temp dir, not fully documented in reports) |
| Shallow clone, no hooks | partial |
| Submodule disable | not enforced by default |
| Max repo / file / count limits | existed; per-file cap not enforced |
| Read-only workspace after clone | not enforced |
| Sandbox metadata in audit report | missing dedicated section |
| Private IP blocking | yes (URL validation) |
| Code execution during audit | no install/build/test |
| Issue / PR creation | 0 (report-only) |

## Active-present (product repo)

- Before this sprint rescan: **876** (after scan `b21dc57c40411f31`)
- Gitea open issues: **1** (#48 operator task)

## Failure messages catalogued

- Repository Map: contradictory missing/loading/unavailable strings (see above).
- Graph API auth failure could be misread as missing graph in UI.
