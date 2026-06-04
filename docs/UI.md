# Repository Detective UI

**Repository Detective — Inspect. Analyze. Improve.**

This document describes theme behavior, browser integration, and practical accessibility notes for the web dashboard (`/ui`).

## Theme support

The dashboard supports three theme modes:

| Mode | Behavior |
|------|----------|
| `system` | Follows the browser/OS `prefers-color-scheme` setting (**default**) |
| `light` | Fixed light palette with Repository Detective brand colors |
| `dark` | Fixed dark palette (original dashboard look) |

### Persistence

- Theme preference is stored in browser `localStorage` under the key `rd-theme`.
- No login or user database is required.
- Clearing site data resets the theme to **system**.

### Avoiding flash on load

A small inline script in the page `<head>` runs before stylesheets and sets `data-theme` on `<html>` from `localStorage`. This prevents a visible flash when returning visitors prefer light mode.

Interactive theme controls load via `theme.js` (deferred).

### Theme toggle

- Location: top bar (header), next to the Live status pill.
- Options: **System**, **Light**, **Dark**.
- Accessible name: **Theme** (visually hidden label + `aria-pressed` on each option).
- Keyboard: Tab to the control group; use **Arrow Left/Right** to move between options.

### Graph and severity colors

Severity badges and graph node colors use distinct hues in both themes so critical/high/medium/low findings remain distinguishable without relying on color alone (text labels are always present on badges).

## Browser system theme

When **System** is selected:

- Light OS/browser theme → light dashboard surfaces, dark text, light graph canvas.
- Dark OS/browser theme → existing dark dashboard appearance.

The `<meta name="theme-color">` tag updates to match the resolved theme for mobile browser chrome.

## Accessibility baseline

Repository Detective targets practical usability, not formal WCAG certification.

### Implemented

- **Skip link** — “Skip to main content” on all layout pages.
- **Semantic structure** — one primary `<h1>` per page in the top bar; section headings use `<h2>`/`<h3>` where applicable.
- **Focus states** — `:focus-visible` rings on links, buttons, form controls, and nav items.
- **Form labels** — inputs tied to labels (`for`/`id`) on graph controls and settings forms.
- **Status badges** — severity and scan status badges include visible text, not color alone.
- **Tables** — `<th>` headers on data tables in dashboard, findings, scans, and health views.
- **Graph fallback** — text summary (`#graph-summary`) with node/edge/orphan/finding counts; status region for load errors.
- **Reduced motion** — animations and transitions respect `prefers-reduced-motion: reduce`.

### Known limits

- The Cytoscape graph is partially visual; use the text summary and node detail panel for screen-reader-friendly stats.
- Some legacy inline pages (pre-install wizard) use minimal styling.

## Issue reconciliation

Repository detail pages include **Reconcile existing issues** when `issue_reconciliation_enabled: true`. Preview shows proposed actions without modifying Gitea.

## Calibration summary

Dashboard can surface calibration metrics via `/api/v1/calibration/summary` (noisy rules, proposed recommendations). Accept/reject from API or future admin UI.

## Related docs

- [SCAN_PROFILES.md](SCAN_PROFILES.md) — scan presets and reporting defaults
- [POLICY.md](POLICY.md) — issue creation and severity gates
- [CODE_GRAPH.md](CODE_GRAPH.md) — graph findings and suppression
