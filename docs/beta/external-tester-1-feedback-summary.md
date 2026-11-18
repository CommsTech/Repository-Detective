# External tester #1 — feedback summary

**Date:** 2026-06-12 (updated post-calibration)  
**Tester:** `ext-operator-jrice`  
**Scan ID (before):** `85a8ab62e76da076`  
**Scan ID (after calibration):** `eb3e7662b31d943c`  
**Repository:** `commstech/Wifi_Collector`  
**Feedback channel:** Gitea `beta_feedback` + `scanner_false_positive`

---

## Feedback status

| Item | Status |
|------|--------|
| Scan ID in feedback | **yes** |
| Gitea template used | **yes** |
| Calibration sprint | **complete** |
| Rescan verified | **yes** — see [external-tester-1-calibration-rescan.md](../dogfood-reports/external-tester-1-calibration-rescan.md) |

---

## What was useful

- Report-only worked — **0 forge issues, 0 PRs**
- Fast scan (~5s) for ~26 files
- Findings detail pages explain severity and location
- Graph map helped orient in multi-file Python repo
- Gitea template links reduce friction
- Post-calibration grouped informational summary improves first-scan readability

## What was confusing (original feedback)

- 123 findings felt overwhelming on first scan
- 1 high `SEC-HARDCODED-SECRET` alarming before detail — **fixed on rescan**
- SBOM `sbom_tool_missing` with requirements.txt — **messaging improved**
- Many `binary_missing` scanners — **documented in beta guides + scan UI**

## Top false positives (resolved / mitigated)

| Finding | Before | After calibration |
|---------|--------|-------------------|
| `SEC-HARDCODED-SECRET` | high | **removed** |
| `REL-INTERNAL-INFRA-REF` ×2 | medium | **removed** |
| `HEALTH-LARGE-FILE` | medium | **low** |
| `QUAL-DEBUG` / graph info | noisy | **grouped in UI** |

## Remaining items

| Rule | Severity | Notes |
|------|----------|-------|
| `HEALTH-TECH-MARKER` | medium | 1 actionable tech-debt marker — acceptable for beta |

## Would tester run again?

**Yes** — report-only; post-calibration experience acceptable.

## Would tester recommend?

**Yes, with caveats** — technical operators only; slim image scanner gaps documented.

---

## Classification (updated)

| Bucket | Items |
|--------|-------|
| **Must fix before next tester** | **None** — high FP resolved on rescan |
| **Should fix before broader private beta** | Optional `HEALTH-TECH-MARKER` tuning |
| **Should fix before marketing** | Wiki, VM install, syft in default image optional |
| **False-positive calibration** | **Done** for SEC-HARDCODED-SECRET, REL-INTERNAL-INFRA-REF, HEALTH-LARGE-FILE |
| **Docs gap** | **Addressed** — scanner matrix, SBOM, first-scan volume |
| **Feature request** | Category filters (future) |
| **Not in scope** | Issue filing on tester repos; AI; runner; container scan |

## Decision

**Proceed to external tester #2.**
