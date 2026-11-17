# Repos control page verification

Date: 2026-06-02  
Commit: `9a8a649`  
Live URL: `http://127.0.0.1:8081/ui/repos`

## Deploy

| Item | Value |
|------|-------|
| Image rebuilt | YES |
| Image revision | `9a8a649` |
| Container | `repository-detective` |

## Route verification

| Check | Result |
|-------|--------|
| `GET /ui/repos` | 200 |
| Fleet control heading | Present |
| Scan Now per repo (`data-scan-open`) | 39 buttons |
| Enable/disable controls | 39 forms |
| Reconciliation explainer | Present |
| Repo 31 (AMMBER) visible | YES |
| Favicon | `favicon.svg?v=2` |

## Manual scan from repo list (repo 31)

Report-only scan via API (same engine as list modal):

| Field | Value |
|-------|-------|
| Scan ID | `8f2b5dbdf0e98310` |
| Trigger type | `manual` |
| Report-only | `true` |
| Issues created | **0** |
| PRs created | **0** |

## Enable/disable scanning (repo 31)

| Action | Result |
|--------|--------|
| `POST /api/v1/repos/31/disable-scanning` | `effective.enabled=false` |
| `POST /api/v1/repos/31/enable-scanning` | `effective.enabled=true` |
| Settings persist | YES |
| Learning event recorded | YES |

## Reconciliation on list (AMMBER)

List row shows scan findings vs mapped forge issues, issue sync status, report-only label, and sync-note badge when counts differ (AMMBER-style: many findings, few mapped issues).

## Tests

| Command | Result |
|---------|--------|
| `go test ./...` | PASS |
| `staticcheck ./...` | PASS |
| `operator-smoke-test.sh` | PASS |
| `make beta-release` | PASS |

## Remaining UX (non-blocking)

- Horizontal scroll on wide fleet table (by design for dense ops view)
- Browser click-through on list Scan Now modal (HTML/JS verified; API scan confirmed)

## Recommendation

**Ready for tester fleet operations** from `/ui/repos`.
