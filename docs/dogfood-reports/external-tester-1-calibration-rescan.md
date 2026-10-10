# External tester #1 — calibration rescan

**Date:** 2026-06-12  
**Repository:** `commstech/Wifi_Collector`  
**Tester:** `ext-operator-jrice`  
**Before scan:** `85a8ab62e76da076` (pre-calibration, `rc-381667a`)  
**After scan:** `eb3e7662b31d943c` (post-calibration, live `dev` binary)  
**Mode:** `report_only_dry_run: true`

---

## Acceptance

| Check | Result |
|-------|--------|
| Scan completed | **PASS** |
| High false positive gone | **PASS** — 0 high in pipeline (was 1) |
| `SEC-HARDCODED-SECRET` | **absent** on rescan |
| Issues created | **0** |
| PRs created | **0** |
| Product dogfood | **0** active-present high/critical on product repo |
| SBOM messaging | **PASS** — `sbom_tool_missing` with syft guidance in scan UI |
| Scanner-missing messaging | **PASS** — scan detail explains binary_missing |

---

## Before / after comparison

| Metric | Before (`85a8ab62…`) | After (`eb3e7662…`) | Delta |
|--------|----------------------|---------------------|-------|
| Total findings (pipeline) | 123 | 120 | −3 |
| High | 1 | **0** | −1 ✓ |
| Medium (pipeline log) | 4 | **1** | −3 ✓ |
| Low | 109 | 96 | −13 |
| Actionable (reconciliation) | 5 | **1** | −4 ✓ |
| Informational (reconciliation) | 118 | **119** | +1 |
| Issues filed | 0 | 0 | — |
| PRs | 0 | 0 | — |
| Duration | ~4.6s | ~5.4s | similar |

## Rule-level changes (this scan only)

| Rule | Before | After | Notes |
|------|--------|-------|-------|
| `SEC-HARDCODED-SECRET` | 1 high | **0** | Placeholder `password = "Decryption failed"` skipped |
| `REL-INTERNAL-INFRA-REF` | 2 medium | **0** | Example CIDR in install.py + LEGAL.md skipped |
| `HEALTH-LARGE-FILE` | 1 medium | 1 **low** | Large Python script calibrated |
| `HEALTH-TECH-MARKER` | 1 medium | 1 medium | Remaining actionable item |
| `QUAL-DEBUG` | 99 low | 81 low + 14 info | Homelab debug downgrade |
| `GRAPH-ORPHAN-FILE` | 13 info | 7 info | Slight reduction; UI groups ≥3 |

## Remaining actionable finding

| Rule | Severity | Path | Assessment |
|------|----------|------|------------|
| `HEALTH-TECH-MARKER` | medium | `wifi_collector.py` | Tech-debt marker — review TODO/FIXME; not a security defect |

## False positives remaining

| Item | Status |
|------|--------|
| `SEC-HARDCODED-SECRET` | **resolved** |
| `REL-INTERNAL-INFRA-REF` homelab examples | **resolved** |
| `HEALTH-LARGE-FILE` severity | **mitigated** (low) |
| `QUAL-DEBUG` volume | **mitigated** via UI grouping + homelab downgrade; still numerous |
| Graph info findings | **mitigated** via grouped summary on scan page |

## SBOM / scanner state (unchanged runtime)

| Item | Status |
|------|--------|
| SBOM | `sbom_tool_missing` — requirements.txt present, syft absent |
| trivy/grype/gitleaks/semgrep/ruff | binary_missing (documented) |
| gitleaks-history | clone failed |

## Tester-facing clarity

- Scan detail shows **actionable vs grouped informational** counts
- SBOM card explains syft requirement for Python manifests
- Scanner table footnote: binary_missing ≠ clean repo

## Operator decision

**Proceed to external tester #2** — high FP eliminated, actionable count 1, volume acceptable with grouping. Optional: tune `HEALTH-TECH-MARKER` before marketing (not blocking private beta #2).
