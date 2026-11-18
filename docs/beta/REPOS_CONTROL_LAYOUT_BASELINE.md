# Repos control layout baseline

Generated: 2026-06-02

## Route

`/ui/repos` — fleet control page

## Latest commit (pre-polish)

`cbe28fd`

## Screenshot-observed layout issues

| Issue | Impact |
|-------|--------|
| Action buttons stacked vertically in narrow column | Cramped, clipped off right edge |
| 11 table columns | Excessive wrapping, tall rows |
| `min-width: 1100px` table | Forces horizontal scroll / clipping |
| Full-width buttons in actions column | Detached from row, poor scanability |
| Long explainer paragraph | Dominates header area |
| No mobile/card fallback | Unusable on narrower viewports |

## Expected layout behavior

| Item | Target |
|------|--------|
| Columns | 6 grouped: Repository, State, Latest scan, Findings, Forge issues, Actions |
| Actions | Primary **Scan now** + compact **⋯** menu (Enable/Disable, Details, Settings, Report) |
| Row height | Compact single-line groups where possible |
| Scroll | `overflow-x: auto` on fleet container only |
| Narrow viewports | Stacked card rows with actions at bottom |
| Help text | Short inline hint; full text in tooltip |

## Functionality to preserve

- Scan Now from list (report-only default)
- Enable/disable scanning
- Reconciliation counts and sync note
- Issue filing off / report-only labels
- Details, Settings, Report links
- Search filter
- Modal manual scan flow

## Safety defaults (unchanged)

Report-only ON, issue filing OFF, no all-repo scan, backlog-control active.
