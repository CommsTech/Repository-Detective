# Repository map graph rescan verification

Recorded: 2026-06-08

## Deploy

| Item | Value |
|------|-------|
| Live image | `repository-detective:all-in-one` |
| Image revision label | `b5f44f6` (includes graph state model + sandbox hardening; `c4e7361` chore on git HEAD) |
| Git HEAD at deploy | `c4e7361` |

## Product rescan

| Field | Value |
|-------|-------|
| Scan ID | `1ec1a0ebe4bd660e` |
| Trigger | manual, `analysis_depth=2`, `enable_code_graph=true` |
| Scan status | `completed` |
| Graph API `state` | `available` |
| Node count | 3674 |
| Edge count | 6053 |
| Truncated | no |
| `graph_enabled` | true |
| Export JSON | HTTP 200, ~1.6 MB |

## Repository Map page

URL: `/ui/scans/1ec1a0ebe4bd660e/graph`

| Check | Result |
|-------|--------|
| Single initial state payload (`graph-initial-state`) | pass — `"state":"available"` |
| No "Loading graph summary…" | pass |
| No "Graph not available…" | pass |
| Cytoscape loaded | pass |
| Contradictory missing+loading messages | **fixed** |

## Active-present reconciliation (product repo ID 1)

| Metric | Value |
|--------|-------|
| Before rescan | 876 |
| After rescan (`1ec1a0ebe4bd660e`) | 87 |

Large drop reflects reconciliation against latest scan with graph orphan calibration; operator should review whether remaining 87 are expected.

## Gitea open issues

Baseline before sprint: **1** (#48 operator task). Not re-queried from Gitea API in this report.

## API sample

```json
{
  "state": "available",
  "scan_id": "1ec1a0ebe4bd660e",
  "repo_id": 1,
  "graph_enabled": true,
  "analysis_depth": 3,
  "node_count": 3674,
  "edge_count": 6053,
  "truncated": false
}
```
