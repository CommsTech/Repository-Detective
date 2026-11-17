# Finding investigation workbench — visual and interaction acceptance

**Status:** unreleased source update (2026-11-17). Intended for authenticated staging validation before public deployment.

## Product goal and audience

The **Findings** queue and **Finding detail** page should work for maintainers who do not have security training and for security engineers investigating complicated issues. The interface must answer, in order:

1. **What happened and where?** Show severity, lifecycle status, scanner, repository, affected path, and a readable headline without requiring a scroll.
2. **What is known, and what is not known?** Show scanner match confidence as a labeled meter, **never** as a probability of successful exploitation or a safety score. Keep unknown reachability and exploitability explicit.
3. **What can I do next?** Offer the remediation-plan anchor, plan status, blockers, regression risk, approval controls, and post-change verification without implying an AI recommendation is already implemented.
4. **Where is the proof?** Give direct access to scanner evidence, fingerprints, timestamps, raw redacted metadata, and history through a keyboard-accessible disclosure.
5. **How do I challenge a result?** Keep false-positive, suppression, and rule-specific disposition actions available but separate from the primary fix path; do not equate accepted risk with a false positive.

## Changes and boundaries

- The case header establishes a hierarchy: source context and risk labels → reader-friendly summary → prominent link to remediation.
- The main assessment is a two-column layout on large screens and a stacked layout on smaller screens. Long repository names, commands, finding titles, file paths and IDs must wrap rather than collide.
- A native HTML `meter` visualizes **existing scanner match confidence** (0–100%). It is not an exploitability forecast or a composite trust score and requires no chart library.
- The primary report includes recommendations; optional capability inventory is collapsed. Plan status and blocking conditions remain visible.
- Remediation actions, CSRF fields, endpoint URLs, approval permissions, closure rules, and issue-link behavior are **unchanged** by this pass.
- The evidence and false-positive controls use native `<details>`/`<summary>`; both are usable with a keyboard and without JavaScript.
- The queue provides a compact lead-in and severity-accented rows while preserving a semantic table with horizontally scrollable columns on small screens.
- No remote fonts, graphics, trackers, or JavaScript dependencies were added. Styles are scoped to the relevant operator surfaces.

## Staging acceptance

Build with the pinned Go toolchain and run unit/integration tests. Start an authenticated **staging** control plane with at least one representative finding. Use a non-production instance: the check is read-only, but it may reveal private repository names in browser memory.

```bash
# Install Playwright only in your CI/testing environment, not the application image.
npm install --no-save playwright
RD_BASE_URL=http://127.0.0.1:8081/ui \
RD_FINDING_ID=123 \
REPOSITORY_DETECTIVE_API_KEY="$REPOSITORY_DETECTIVE_API_KEY" \
node scripts/ui-finding-workbench-check.js
```

The script checks 390, 768, 1024, 1280 and 1600 px in light/dark themes. It asserts no document-width overflow or overlapping title/action panels; a visible confidence meter; required anchors, CSRF-containing forms, a dismissed theme bootstrap, and functioning native details controls. It does **not** submit any form or create a PR.

Separately verify proposed, approved, rejected, disabled, and absent remediation plans; high and low confidence, dependency and non-dependency findings; false-positive controls; read-only users; a very long title and path; a repository with no evidence; and secret redaction. Use keyboard-only navigation, screen-reader output, 200% / 400% zoom, reduced-motion mode, and real user sessions. Test that the `Approve plan`, `Reject plan`, `Generate plan`, and `Verify closure` actions still obey server-side permissions and produce accurate error/success feedback. Never approve a real patch purely to pass a UI test.

## Evidence actually collected in this development environment

- Standard-library Go HTML template **parsing and rendering** with synthetic proposed-plan and no-plan fixtures: passed.
- Static CSS parser, JavaScript syntax and Go formatting checks: passed.
- DOM structure and presence of remediation forms, closure step, fingerprint evidence and collapsible sections in rendered HTML: passed.
- **Real browser layout/interaction screenshot acceptance: not run.** The container's Chromium browser was denied local navigation by administrator policy. Do not mistake static validation for viewport/contrast testing.
- **Full Go 1.26.11 tests, authenticated staging integration and live data performance benchmarking: not run here.** These remain release gates.

## Operator/release decisions

This is a frontend workbench improvement. It does **not** add new AI execution capability, certify runner isolation, change suppression policy, or prove a repository safe. Monitor actual case-to-remediation navigation time, false-positive disposition accuracy, UI accessibility issues and user feedback during staging. Treat scanner-completeness and commit provenance separately from visual confidence indicators.
