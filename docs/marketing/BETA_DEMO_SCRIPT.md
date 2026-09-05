# Beta demo script (controlled, non-marketing)

Duration: ~20 minutes. Audience: technical evaluator. **Report-only.**

## 1. Health and scanners (2 min)

- Open `/health` — show database healthy, tools_summary (explain missing optional scanners)

## 2. Scan a repository (5 min)

- Trigger manual scan on owned repo
- Show scan detail: persistence, graph state, scanner coverage

## 3. Repository Map (3 min)

- Open graph view for completed scan
- Highlight findings linked to nodes

## 4. Pre-install audit (3 min)

- Run report-only pre-install audit on public repo URL
- Show shareable report; no auto-disclosure

## 5. Container image scan (3 min)

- Show Container Images panel (disabled by default)
- Explain runner + `container-scan` label; no core Docker socket
- Demo discover + enqueue (if runner available)

## 6. Finding reconciliation (2 min)

- Actionable vs informational counts
- Why no issue filed (backlog control / report-only)

## 7. Learning / calibration (2 min)

- `/ui/learning` — recommendations queue, false-positive rate

## 8. Advanced (optional)

- Runner delegation: **off by default** — mention as advanced
- Remediation PR: **disabled / gated**

## Close

- Marketing not started; private beta
- Next: wiki fix, non-product cohort, container scan demo on runner host
