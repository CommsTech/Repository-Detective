# Accessibility

Repository Detective targets **WCAG 2.2 AA–aligned improvements** and **Section 508–friendly** patterns. This is **not** a claim of formal WCAG or Section 508 certification — validation requires your environment and assistive technology matrix.

## Supported practices

| Area | Implementation |
|------|----------------|
| Language | `lang="en"` on pages |
| Landmarks | `aside` navigation, `main#main-content`, labeled chart regions |
| Skip link | “Skip to main content” — visible on keyboard focus |
| Focus | `:focus-visible` ring using `--rd-focus` (teal) on links, buttons, inputs |
| Charts | Text summary above chart grid; tables duplicate key metrics |
| Severity | Badge text (critical/high/…) — not color-only |
| Motion | `prefers-reduced-motion` reduces animations |
| Forms | Labels on settings forms (per-page); confirm dialogs via `data-confirm` |
| Tables | `th scope="col"` on operator tables |

## Keyboard navigation

- **Tab** — move through sidebar links, buttons, form controls
- **Enter** — activate links and buttons
- **Skip link** — first focusable element jumps to `#main-content`

Sidebar is sticky; long pages scroll in main content.

## Chart accessibility

Charts (Chart.js) are **supplementary**. Each dashboard includes:

1. Hero metrics (counts in text)
2. **Chart data text summary** (screen-reader-oriented block)
3. Tables (findings, scans, scanner coverage)

If Chart.js fails to load, an alert explains that numeric panels remain available.

## Color and contrast

Dark theme uses light text on navy surfaces. Severity uses labeled badges. Scanner impact uses text: `degraded`, `inactive`, `ok`.

Report contrast issues with the **Accessibility** issue template.

## Known limitations

- No dedicated high-contrast theme toggle (OS high contrast may apply)
- No global keyboard shortcut layer yet (see `docs/issues/P1-keyboard-shortcuts-nav.md`)
- Chart tooltips are mouse/hover-oriented
- Some icons in nav are decorative (`aria-hidden`) with text labels beside them
- Pre-install and onboard flows may vary in labeling depth

## Testing checklist (manual)

- [ ] Tab through `/ui/` sidebar and reach main content via skip link
- [ ] Focus ring visible on Dashboard, Findings, Health links
- [ ] Dashboard readable with charts disabled (block `chart.umd.min.js`)
- [ ] Finding detail: evidence section announced as preformatted text
- [ ] Health scanner table readable left-to-right with screen reader
- [ ] Zoom 200%: no horizontal overflow on dashboard grid

## Feedback

File issues with template **Accessibility** and label `type/accessibility`.

---

See also [Home](Home).
