# Dashboard information architecture and visual acceptance

Updated: 2026-11-17. Applies to the **unreleased source tree**. This is a visual and workflow change, not a claim that runner isolation, remediation, or scanner completeness is finished.

## Audience and first decisions

Repository Detective is for independent maintainers, security engineers and teams managing multiple repos. On first visit an operator must answer, in order: **What is urgent? Are scans actually happening? Which finding needs investigation? Where is evidence and the full audit?**

The dashboard now uses five portfolio indicators (critical/high, deduplicated backlog, unhealthy repositories, active scans, verified resolved) with explicit labels and links to supporting records. It separates **finding counts** from **scanner coverage** and does not call a repository safe because it has few findings.

The first analysis row gives a 14-day scan/plan/PR trend and a severity distribution; the next row ranks finding categories using horizontal bars. Radar charts were removed from the dashboard because category quantities compare more clearly on a common axis. Advanced repository risk, scanner availability, recent scan history, and calibration are available in native keyboard-accessible disclosure elements rather than six extra walls of information on every visit.

**Trust boundary:** a completed scan, initiated PR, or resolved finding count is not equivalent to complete coverage or verified safety. A PR metric is not a verified remediation count. The original stored data and links remain authoritative.

## Interaction and responsiveness

- Dashboard content never relies solely on chart colors; numeric severity counts and descriptive text accompany graphics.
- Cards and headings can shrink to fit without forcing viewport overflow; long finding titles wrap, and wide tables scroll **inside their cards**. Visible table headings avoid visually-hidden elements escaping scroll containment.
- Breakpoints: desktop five KPI cards, intermediate three, narrow two; trend and severity charts stack below 1050px; navigation is collapsible at 768px; expanded evidence stays keyboard accessible.
- Canvas charts use existing local Chart.js. Repository risk visualization mounts only after opening the disclosure, saving work on the initial view. The category chart grows for additional categories.
- Browser rendering respects light, dark and reduced-motion preferences; no external analytics, fonts, or design service was introduced.

## Repeatable QA gate

After deploying to an **authenticated staging** instance, run:

```bash
RD_BASE_URL=http://127.0.0.1:8081/ui \
  REPOSITORY_DETECTIVE_API_KEY="$REPOSITORY_DETECTIVE_API_KEY" \
  node scripts/ui-dashboard-responsive-check.js
```

The script checks ten viewport/theme combinations (390, 768, 1024, 1280, 1600px; light and dark), page overflow, card/header collisions, chart dimensions/errors, runtime JS errors, dark/light background correctness, flash-guard removal and the lazy repository risk chart. It is **read-only** and does not start a scan or remediation. Use Playwright in the existing CI image or install Playwright in an isolated development environment; the production container does not need Node.

Test manually as well: keyboard-only from topbar to primary findings; Ctrl/⌘+K and Escape; a repository name 100+ characters long; a long rule ID; zero registered repositories; no findings; missing scanner; a live scan; 200+ findings; screen reader announcement; system theme and reduced motion; 200–400% zoom; permission-limited user accounts. Confirm charts retain valid labels when data is absent.

## Verified here vs pending

**Performed on synthetic template fixtures:** Go standard-library HTML template render, browser layout checks at ten viewport/theme combinations with Chart.js running, dark/light screenshots, table containment and expandable chart interaction. The checks do not query production data.

**Not performed here:** Go 1.26.11 full tests (toolchain unavailable); server-route authentication checks; actual GitHub/Gitea runner execution; deployed browser E2E; performance profiling on real fleet datasets; third-party accessibility audit. Run these before release promotion.
