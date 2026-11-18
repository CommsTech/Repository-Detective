# Scan & repository policy workbenches — visual QA and release gate

**Status:** Working-tree UX update, 2026-11-17. Not deployed by this package. This is a presentation/workflow change, **not** new scanner coverage or certified AI remediation.

## Audience and intended task order

- **First-time maintainer:** choose a predictable repository scan profile → optionally define declarative custom rules → select alert routing → save → run a baseline scan.
- **Experienced maintainer:** jump directly to a policy section, inspect inherited overrides, or expand advanced settings without losing existing controls.
- **Scan reviewer:** identify pinned revision and runner provenance → inspect actual scanner outcomes (including missing tools and errors) → review AI guidance if enabled → inspect full diagnostic JSON as needed.
- A recorded `clean` scanner result means only that the tool found nothing in its executed scope. The coverage strip is **not** a percentage of code inspected or a trustworthy-release certificate. A scanner record with an error needs attention even if its status text is otherwise positive.

## Implementation

- `ui/templates/scan_detail.html`: report hero, section navigation, compact metrics, evidence-first scanner table, accessible scroll containers, non-decorative outcome bar, AI advisory section and collapsible raw diagnostics.
- `ui/templates/repo_settings.html`: visible starting path, direct anchors to scan profile, custom rules, notifications and advanced options, retained rule builder and CSRF-protected POST, clear save action and inherited-config disclosure.
- `ui/templates/configure.html`: links to scanner gates, issue/remediation, scanners and AI budget controls; fixes the duplicate scan-profile DOM ID.
- `ui/static/theme.css`: responsive layout and long-text clipping, light/dark surfaces, consistent secondary buttons and keyboard focus.
- `ui/static/app.js`: computes the completed/needs-inspection counts from **actual persisted scanner rows**, never interpolates untrusted text as HTML, and reinitializes after partial updates. The scan soft refresh retains scroll position and open identified disclosures, without a full page reload.
- `ui/scan_settings_workbench_test.go`: integration contract checks for scan evidence order, scanner error markers, settings anchors, POST CSRF and custom rules. Requires the repository's Go toolchain.

## Checks executed here

| Check | Result |
|---|---|
| Parse all HTML templates with standalone Go `html/template` and the actual template-function names | PASS |
| Execute scan and settings templates with synthetic data and long forge paths | PASS |
| Chromium Playwright `set_content` rendering at widths 390/768/1280/1600 for light and dark on **both pages** | PASS, 16 combinations |
| Whole-page horizontal overflow, colliding card headings/hero actions, broken section anchors, JavaScript errors | NONE detected in synthetic fixtures |
| Scanner outcome visualization initialized from existing scanner rows | PASS in synthetic render |
| JavaScript syntax, CSS parsing, Go formatting | PASS |
| Full Go suite (requires Go 1.26.11) | NOT RUN: local environment Go 1.23.2 |
| Real auth/session, backed scan/runner/Gitea, populated settings POST, load/performance, screen reader and full app flows | NOT VERIFIED |

**Synthetic preview limitation:** browser scripts were injected into static rendered test pages rather than loaded from a running authenticated app. These checks cannot prove production interaction, correct authorization, runner execution or AI remediation.

## Staging acceptance (read-only)

Use a disposable Gitea repository and a completed scan. Run the Go suite on the supported toolchain, then:

```sh
GOTOOLCHAIN=go1.26.11 go test ./...
# With playwright installed in the test environment:
RD_BASE_URL=https://your-staging-host/ui RD_SCAN_ID=your-scan-id RD_REPO_ID=123 node scripts/ui-scan-settings-check.js
```

Browser authentication must be established separately before the staging script runs. Do not put a token into loggable URLs. Audit each flow manually with keyboard-only navigation and at least one screen reader. Ensure both read-only views work when AI is disabled, scanners are missing, a scanner fails, no results exist and when long error messages occur. Independently verify the server rejects unauthorized POST changes and that saving policies does not start a scan or silently enable third-party issue filing. Confirm that a scanner error-bearing `found` record is not counted as a completed outcome.

## Performance and accessibility gates

Measure on staging with realistic 1,000+ finding records and multiple scanner outcomes: navigation time, server response p50/p95, DOM content loaded, Largest Contentful Paint, total JS transfer and memory. **No performance improvements are claimed without these measurements.** Require readable 200% zoom, correct focus after disclosure toggling, no dark-mode white flash, reduced-motion behavior and no WCAG 2.2 AA contrast/focus regressions. The long-term UI needs further cross-screen design/interaction review beyond these two workbenches.
