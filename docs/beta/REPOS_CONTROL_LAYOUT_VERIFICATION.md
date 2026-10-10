# Repos control layout verification

Date: 2026-06-02  
Commit: `c9dc996`  
Live URL: `http://127.0.0.1:8081/ui/repos`

## Before (problems)

- 11-column table with `min-width: 1100px`
- Stacked full-width action buttons clipped on the right
- Very tall rows from column wrapping
- Long explainer dominated the header

## After (behavior)

| Item | Result |
|------|--------|
| Layout | 6-column fleet grid (Repository, State, Latest scan, Findings, Forge, Actions) |
| Actions | **Scan now** + **⋯** overflow menu (Enable/Disable, Details, Settings, Report) |
| Scroll | Contained in `.rd-fleet-scroll` wrapper |
| Narrow viewports | Card-style stacked rows with labeled sections |
| Help text | Short hint with full text in `title` tooltip |
| Row height | Compact inline state badges and counts |

## Live checks

| Check | Result |
|-------|--------|
| `/ui/repos` loads | 200 |
| Fleet rows | 39 |
| Scan Now buttons | 39 |
| Action menus | 39 |
| Settings in menu | YES |
| Favicon | `favicon.svg?v=2` |

## Manual scan (repo 31 / AMMBER)

| Field | Value |
|-------|-------|
| Scan ID | `b1ccf97e2d814d2e` |
| Trigger | `manual` |
| Report-only | `true` |
| Issues created | **0** |
| PRs created | **0** |

## Tests

| Command | Result |
|---------|--------|
| `go test ./...` | PASS |
| `staticcheck ./...` | PASS |
| `operator-smoke-test.sh` | PASS |
| `make beta-release` | PASS |

## Remaining UX (non-blocking)

- Hover tooltips for sync note could link to repo reconciliation panel
- Optional column sort on desktop

## Recommendation

Layout is **beta-ready** for fleet operations at common laptop/desktop widths.
