# First tester feedback summary

**Date:** 2026-06-12  
**Scan ID:** `512145e55d4488ea`  
**Repository:** `commstech/PCAP_Analyser`  
**Tester:** `operator-cohort-1` (internal rehearsal — structured feedback per beta templates; external tester still pending)

## Feedback status

| Item | Status |
|------|--------|
| Structured template (Gitea `beta_feedback`) | **received** (operator rehearsal) |
| External named tester | **pending** |

---

## Template: what was useful

- Fast scan (~4.4s) on small repo (8 files)
- Report-only worked — **0 forge issues, 0 PRs**
- Findings detail pages actionable (severity, confidence, calibration hints)
- Graph available (29 nodes) despite tiny codebase
- Executive report useful for operator triage
- Issue template links on finding/scan pages (post `311e97c`) improve feedback path

## What was confusing

- SBOM empty state on Python repo without lockfile — correct but needs one-line “add requirements.txt/poetry.lock” hint
- 12 findings on 8 files feels dense for first impression (mostly info-level graph/static noise)
- `standard_deterministic` vs `beta_standard` naming in docs
- Several scanners show `binary_missing` in scanner results (semgrep, trivy, grype, gitleaks) — expected in slim image but surprising to testers

## Top false positives (operator triage)

| Finding | Rule | Severity | Assessment |
|---------|------|----------|--------------|
| Large PowerShell scripts | `HEALTH-LARGE-FILE` | medium (×2) | **Likely acceptable** — PCAP tooling scripts are inherently large; calibrate or downgrade for PS repos |
| Graph islands / orphans | `GRAPH-*` | info (×9) | **Informational OK** — many already suppressed globally; useful for map, noisy in list |
| Nested loop in script | `OPT-NESTED-LOOP` | low (×1) | **Context-dependent** — may be real hotspot or acceptable for analysis scripts |

## Are the 2 medium findings real?

**Partially.** `HEALTH-LARGE-FILE` is technically true (file length) but not a security defect for this repo class. Recommend repo-scoped calibration or health threshold tuning for PowerShell-heavy repos — **not a product bug**.

## Are the 9 info findings useful?

**Yes, with caveats.** Graph heuristics help map standalone scripts; as issue candidates they are too noisy. Good for dashboard + map; default report-only is correct.

## Report quality

Acceptable for invited operator beta. Reconciliation panel correctly explains report-only vs forge issues. Not marketing-polished.

## Findings detail usefulness

**Good.** Severity/confidence explanations, location, and false-positive guidance are clear. Template link to Gitea `scanner_false_positive` reduces friction.

## SBOM empty-state clarity

**Needs docs/UI hint** — status `sbom_no_supported_manifest` is honest; add one sentence on supported manifests for Python.

## Graph usefulness

**Moderate positive** — 29 nodes / 23 edges helps see disconnected PowerShell scripts; not critical for this repo size.

## Would run again?

**Yes** (report-only). Tester would use Gitea templates for any FP/docs feedback.

---

## Classification

| Bucket | Items |
|--------|-------|
| **Must fix before next tester** | None |
| **Should fix before marketing** | SBOM empty-state hint for manifest-less Python; scanner `binary_missing` visibility in tester guide |
| **False-positive calibration** | `HEALTH-LARGE-FILE` for large `.ps1` analysis scripts (repo-scoped) |
| **Docs gap** | Scan profile naming; explain optional scanners not in slim image |
| **Not in scope** | Issue filing on tester repos; AI; runner; container scan |

## Blockers from first tester

**None** for cohort continuation. External named tester feedback still recommended before broad expansion.

## Recommended Gitea templates for follow-up

- False positive on large-file mediums → `scanner_false_positive`
- SBOM/docs → `docs_gap` or `beta_feedback`
