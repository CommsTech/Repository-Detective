# Stabilization UI report

## What was broken

1. **Dark-mode flash** — pages briefly rendered light background before theme CSS loaded.
2. **Setup wizard nav** — always shown even after Gitea/API key/repos configured.
3. **Repository map** — disconnected package findings could reference `_test.go` paths; graph theme applied after paint.

## Files changed

- `ui/templates/layout.html` — pre-paint theme background + setup-complete nav
- `graph/orphans.go` — filter test files from disconnected package clusters
- `ui/static/theme.css` — print + nav secondary styles

## Tests

- `go test ./graph/...` — `TestTestFileNotOrphan` passes
- `go test ./ui/...` — executive + capability tests

## Before / after

| Area | Before | After |
|------|--------|-------|
| Dark mode | White flash on navigation | Inline `backgroundColor` + `colorScheme` before first paint |
| Nav | Always “Setup wizard” | “Configure” + diagnostics link when setup complete |
| Graph orphans | `_test.go` in File field possible | Non-test files only in disconnected package findings |

## Manual verification

1. Set dark mode → navigate Dashboard → Scans → Findings — no white flash
2. Configured install → sidebar shows Configure, not primary Setup wizard
3. Open scan graph with stored data — renders or shows clear missing/truncated message

## CI / scan

- CI run: pending post-push
- Scan ID: pending rescan

## Remaining risks

- Cytoscape rendering on very large graphs still subject to node limits
- Host-network compose override documented separately

## Next batch

Batch 2 blocked until CI green.
