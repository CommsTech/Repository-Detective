# Bundled UI assets (Phase 11B)

| File | Version | Source |
|------|---------|--------|
| `cytoscape.min.js` | 3.30.2 | [Cytoscape.js](https://github.com/cytoscape/cytoscape.js) — vendored for offline graph UI |

These files are embedded in the Go binary via `ui/embed.go` and served from `{ui_base_path}/static/`. No CDN or runtime internet fetch is required for the repository map page.
