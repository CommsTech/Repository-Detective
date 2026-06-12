# External tester #1 — feedback summary

**Date:** 2026-06-12  
**Tester:** `ext-operator-jrice`  
**Scan ID:** `85a8ab62e76da076`  
**Repository:** `commstech/Wifi_Collector`  
**Feedback channel:** Gitea `beta_feedback` + `scanner_false_positive` (structured templates)

---

## Feedback status

| Item | Status |
|------|--------|
| Scan ID in feedback | **yes** |
| Gitea template used | **yes** |
| Secrets in feedback | **none** |

---

## What was useful

- Report-only worked — **0 forge issues, 0 PRs** on tester repo
- Fast scan (~4.6s) for ~26 files
- Findings detail pages explain severity and location
- Graph map (104 nodes / 111 edges) helped see Python module structure
- Executive report useful for triage
- Gitea template links reduce friction vs free-form email
- Reconciliation panel correctly explains report-only vs forge issues

## What was confusing

- **123 findings** on a small homelab repo felt overwhelming for a first scan
- **1 high** `SEC-HARDCODED-SECRET` was alarming before reading detail — turned out likely benign config pattern (false positive)
- SBOM page says `sbom_tool_missing` while repo has `requirements.txt` — unclear why SBOM didn’t run
- Many scanners show `binary_missing` (trivy, grype, gitleaks, semgrep) — expected in slim image but surprising without upfront docs
- `standard_deterministic` vs `beta_standard` naming in docs
- Volume of low/graph findings drowns actionable mediums

## Top false positives

| Finding | Rule | Severity | Tester assessment |
|---------|------|----------|-------------------|
| Hardcoded secret heuristic | `SEC-HARDCODED-SECRET` | high | **False positive** — homelab config string, not a live credential |
| Internal infra references | `REL-INTERNAL-INFRA-REF` | medium (×2) | **Likely acceptable** — expected homelab hostnames |
| Large file | `HEALTH-LARGE-FILE` | medium | **Context-dependent** — collection script size |
| Tech debt marker | `HEALTH-TECH-MARKER` | medium | **Informational** |
| Graph orphans / islands | `GRAPH-*` | info/low | **Noisy** — useful on map, not as findings |

## Missed detections

None reported for this scan scope. Tester did not expect container or history scanning (disabled / failed clone).

## Docs gaps

- Explain which optional scanners are absent in slim Docker image
- SBOM: clarify syft requirement when `requirements.txt` exists
- First-scan guidance: expect high finding count on graph-heavy profiles; how to filter
- Report-only workflow screenshot optional

## UI / report quality

**Acceptable** for invited beta. List density is the main UX concern. Detail pages are clear. Not marketing-polished.

## SBOM clarity

**Needs improvement.** Honest `sbom_tool_missing` status is good; missing one-line “install syft to enable Python SBOM” hurts trust slightly when manifest exists.

## Graph usefulness

**Moderate positive** — helped orient in multi-file Python repo; would prefer graph noise suppressed from default findings list.

## Would tester run it again?

**Yes** — report-only on owned homelab repos, after filtering/noise expectations are set.

## Would tester recommend to another technical operator?

**Yes, with caveats** — recommend only to technical operators who understand homelab noise, report-only default, and optional scanner gaps. Not ready for non-technical users.

---

## Classification

| Bucket | Items |
|--------|-------|
| **Must fix before next tester** | None (scan completed safely; 0 issues filed) |
| **Should fix before broader private beta** | Calibrate/downgrade `SEC-HARDCODED-SECRET` for benign homelab patterns; reduce graph/info noise in default list; SBOM messaging when manifest exists but syft missing |
| **Should fix before marketing** | Scanner `binary_missing` visibility in tester guide; first-scan volume expectations; wiki/docs parity |
| **False-positive calibration** | `SEC-HARDCODED-SECRET` (high); `REL-INTERNAL-INFRA-REF` for homelab repos; graph heuristics default suppression |
| **Docs gap** | Slim-image scanner matrix; SBOM syft requirement; profile naming |
| **Feature request** | Category filters / “actionable only” default view |
| **Not in scope** | Issue filing on tester repos; AI; runner; container scan; all-repo scan |

---

## Decision input

| Signal | Result |
|--------|--------|
| Scan/report safety | **PASS** |
| Must-fix blockers | **0** |
| False-positive noise | **Elevated** — calibration sprint recommended before tester #2 |
| Install/docs confusion | **Minor** — docs tweaks, not blocking |
| Expand to tester #2 | **After focused calibration/docs sprint** (not immediately) |

## Recommended Gitea follow-ups

- `scanner_false_positive` for `SEC-HARDCODED-SECRET` on scan `85a8ab62e76da076`
- `beta_feedback` for SBOM syft messaging
- `docs_gap` for slim-image scanner availability table
