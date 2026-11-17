# Reports and navigation: operator guide (unreleased)

> **Scope:** Working-tree changes based on `repository-detective-main(7).zip`, 2026-11-17. These are not claims about the currently published container image. The report is an evidence summary, **not** a safety certification.

## Fast navigation

- Use the sidebar for routine navigation. The current page is visibly highlighted and marked using `aria-current="page"`.
- Press **Ctrl+K** (Windows/Linux) or **⌘K** (macOS), or **/** when not typing in a field, to open **Find a page**. Type part of a page name (e.g. `reports`, `findings`, `health`). Press **Enter** for the first match, use arrow keys to move among matches, or press **Esc** to close. The switcher indexes only destinations already displayed in your authorized sidebar; it does not search repository data or call an external service.
- Open **Reports** for fleet-level summary; select a repository's **View** button for its dossier. Use the in-page section links to jump directly to evidence, findings, or SBOM. On a repository report, **Review findings** opens the maintainer's triage queue.
- The switcher avoids digging through menus; normal page navigation still loads a new server-rendered page. Faster page-load times have **not** been benchmarked here.

## Read an audit report in the right order

1. **Decision & limits:** Read the recommendation, risk posture, confidence, top risks, and explicit limitations. A caution or blocked label is **not** a scanner coverage certificate.
2. **Key metrics:** Open findings by severity, remediations opened, verified resolutions and failed scans are persisted counts as of the displayed **UTC snapshot time**.
3. **Coverage:** Per-repository configuration badges show what scanners were **enabled in policy**, **not** successful executions. The latest scan outcomes table displays actual scanner status records, finding counts, and available errors. Follow the scan link for commit, workspace, details and lifecycle evidence. A recorded success does not prove all relevant sources were inspected.
4. **Technical findings:** This appendix is filtered and paged (200 per page). The printed report includes only the visible page; visit other pages for remaining findings. Counts above can represent a different scope from the filtered table.
5. **Supply chain:** The SBOM panel indicates whether an artifact exists. Missing or inaccessible artifacts are not counted as verified coverage.

### Known scope and data gaps

- The fleet repository listing is capped at the first **200** repository summary rows; when more exist, a warning is displayed. The order is descending by *open finding count*, **not** exploitability or runtime exposure. Fleet summary counts use the dashboard aggregation. To make truly exhaustive fleet ranking/export, a future release must implement proper paging or a dedicated report query.
- The fleet recent scan table is a sample (up to 20), not exhaustive historical attestation.
- Scanner results are shown for the latest recorded scan, not every historical scanner attempt. To compare older scans open the scan history.
- The current executive summary uses persisted findings and scanner result states. It cannot establish a formal release approval, guarantee safe dependencies, or assert the absence of malware or supply-chain compromise.
- A repository with no scans, no scanner-result records, incomplete latest scan or evidence older than 14 days receives a caution. The 14-day threshold is advisory for report freshness, not a product-wide enforcement policy.
- Errors from key report DB reads now return HTTP 500 instead of silently rendering incomplete zeros. They do not expose raw SQL errors in the response.

## Printing and exporting

Select **Print / Save PDF** from the report header. The browser's print view hides navigation and interactive controls while keeping the snapshot timestamp, risk summary, coverage caveats, tables and findings appendix visible. This is **browser print**, not a signed, tamper-evident PDF or complete forensic evidence archive.

## UX and performance design choices

- The repository category visualization is a semantic grid; **Chart.js and its associated runtime are no longer loaded on repository report pages**. This removes a chart dependency and provides readable printed evidence.
- The page switcher is vanilla JS, no additional network requests or third-party dependencies.
- A reduced-shadow, consistent-spacing interface uses existing brand/theme variables and the already-supported dark-mode bootstrap; no remote font or external icon service was added.
- The additional SVG navigation sprite is a local static asset, not a CDN request.
- Actual latency, memory usage and large-fleet throughput require deployment-specific measurement with representative repositories.

## Regression and acceptance checklist

| Test | Current working-tree result | Required before release |
| --- | --- | --- |
| Go template parse and synthetic fleet/repository render | Passed with Go 1.23 stdlib render harness | Execute real handler tests on Go 1.26.10 |
| Quick switcher open/filter/Esc/focus | Passed in isolated headless Chromium | Run against authenticated deployed app on desktop/mobile |
| Dark report render and section links | Passed with synthetic data | Run light/dark/system with real data and screenshots |
| JavaScript syntax / Go formatting | Passed | Run full lint, vet, unit tests |
| Report data acquisition errors | Code changed to return errors | Inject DB failures and validate 500 responses |
| Actual scan coverage and recommendation | Regression tests added, **not run** in full module | Test no-scans, incomplete, stale, success, partial, and failed scanners |
| Print output | CSS present and mock HTML rendered | Confirm actual PDF print layout and pagination |
| Resource and navigation performance | Unmeasured | Measure p50/p95 route latency, DB query times, browser LCP, CLS, memory and large-fleet report pagination |
| Live Gitea/runner/AI remediation | Not in scope, unverified | End-to-end staging acceptance before public release |

Browser runner: `RD_BASE_URL=http://127.0.0.1:8081/ui RD_VERIFY_REPO_ID=1 node scripts/ui-report-navigation-smoke.js` (needs Playwright installed in the test environment and an authorized test UI). This test is **read-only**; it does not start scans or remediation.
