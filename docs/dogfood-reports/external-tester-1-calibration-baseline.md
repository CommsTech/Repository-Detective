# External tester #1 — calibration baseline

**Scan ID:** `85a8ab62e76da076`  
**Repository:** `commstech/Wifi_Collector` (repo id 10)  
**Tester:** `ext-operator-jrice`  
**Date:** 2026-06-12  
**Live revision:** `rc-381667a`

---

## Summary

| Metric | Value |
|--------|-------|
| Raw findings (API list) | 136 |
| Pipeline findings | 123 |
| Critical | 0 |
| High | 1 |
| Medium | 4 |
| Low | 121 |
| Info | 10 |
| Issues created | 0 |
| PRs created | 0 |

## Severity breakdown

| Severity | Count |
|----------|-------|
| high | 1 |
| medium | 4 |
| low | 121 |
| info | 10 |

## Rule breakdown

| Rule | Count | Assessment |
|------|-------|------------|
| `QUAL-DEBUG` | 99 | Noisy — debug prints in collector script |
| `GRAPH-ORPHAN-FILE` | 13 | Informational graph noise |
| `HEALTH-TECH-PHRASE` | 9 | Tech-debt informational |
| `OPT-NESTED-LOOP` | 3 | Advisory perf |
| `HEALTH-TECH-MARKER` | 3 | Tech-debt (1 medium) |
| `REL-INTERNAL-INFRA-REF` | 2 | Homelab example CIDR — downgrade |
| `HEALTH-COMMENT-BLOCK` | 2 | Tech-debt info |
| `GRAPH-SUSPICIOUS-ISLAND` | 2 | Graph info |
| `HEALTH-LARGE-FILE` | 1 | Large python script — low not medium |
| `SEC-HARDCODED-SECRET` | 1 | **False positive** — status message |
| `HEALTH-DEPRECATED` | 1 | Tech-debt info |

## Scanner / tool breakdown

| Source | Count |
|--------|-------|
| static | 105 |
| tech_debt | 15 |
| graph | 15 |
| maintainability | 1 |

### Runtime gaps (not findings)

| Tool | Status |
|------|--------|
| trivy | binary_missing |
| grype | binary_missing |
| gitleaks | binary_missing |
| semgrep | binary_missing |
| ruff | binary_missing |
| gitleaks-history | failed (git clone) |
| syft / SBOM | sbom_tool_missing (requirements.txt present) |

## High finding detail

| Field | Value |
|-------|-------|
| Fingerprint | `rd-7d677a8f45b47b2b` |
| Rule | `SEC-HARDCODED-SECRET` |
| Path | `wifi_collector.py` (~line 6198) |
| Matched line | `password = "Decryption failed"` |
| Why flagged | Static regex: `password = "…"` with 8+ char literal |
| Real secret? | **No** — error/status string returned when decryption fails |
| gitleaks would flag? | Unlikely (not in slim image); pattern is human-readable phrase |
| Tester feedback | False positive — alarming high before reading detail |
| Classification | false_positive |
| Planned action | Skip placeholder/status literals; require entropy/token shape for high |

## Medium findings detail

| Fingerprint | Rule | Path | Why flagged | Classification | Planned action |
|-------------|------|------|-------------|----------------|----------------|
| `rd-0d0e51dc420fed9a` | `REL-INTERNAL-INFRA-REF` | `install.py` | `f.write("# 192.168.1.0/24\n")` example config | useful_info / FP in homelab | Skip example CIDR writes |
| `rd-5307e7d9252af115` | `REL-INTERNAL-INFRA-REF` | `LEGAL.md` | `192.168.1.0/24` in legal/docs text | useful_info | Downgrade .md homelab refs to info |
| `rd-6fcc0632682cbd01` | `HEALTH-LARGE-FILE` | `wifi_collector.py` | ~6k line monolithic collector | useful_info | low severity for operational `.py` |
| `rd-ae56e8161ef3f88c` | `HEALTH-TECH-MARKER` | (tech_debt) | TODO/FIXME-style marker | useful_info | Keep — informational |

## Info / graph noise

| Category | Count | Notes |
|----------|-------|-------|
| `GRAPH-ORPHAN-FILE` | 13 | Map structure — group in scan summary |
| `GRAPH-SUSPICIOUS-ISLAND` | 2 | Informational |
| Other info | 10 | Mixed health/graph |

## SBOM state

| Field | Value |
|-------|-------|
| Status | `sbom_tool_missing` |
| Detail | syft not installed |
| Manifest | `requirements.txt` detected in repo profile |
| Honest? | Yes — does not claim clean SBOM |

## Triage table (representative)

| Fingerprint | Rule | Sev | Conf | Path | Scanner | Why flagged | Tester feedback | Classification | Planned action |
|-------------|------|-----|------|------|---------|-------------|-----------------|----------------|----------------|
| `rd-7d677a8f45b47b2b` | SEC-HARDCODED-SECRET | high | 0.87 | wifi_collector.py | static | password status message | FP | false_positive | Rule fix — skip placeholders |
| `rd-0d0e51dc420fed9a` | REL-INTERNAL-INFRA-REF | medium | 0.80 | install.py | static | example CIDR write | acceptable | global_rule_fix_candidate | Skip example lines |
| `rd-5307e7d9252af115` | REL-INTERNAL-INFRA-REF | medium | 0.50 | LEGAL.md | static | docs CIDR | acceptable | global_rule_fix_candidate | Homelab .md → info |
| `rd-6fcc0632682cbd01` | HEALTH-LARGE-FILE | medium | 0.95 | wifi_collector.py | maintainability | huge script | context-dependent | global_rule_fix_candidate | low for .py scripts |
| `rd-00ad40d31efe3` | QUAL-DEBUG | low | 0.80 | (multiple) | static | print/debug | noisy | duplicate_or_groupable | UI group ≥3 |
| `rd-3eff61541dc8b` | GRAPH-ORPHAN-FILE | info | 0.33 | (multiple) | graph | orphan heuristic | noisy | duplicate_or_groupable | UI group ≥3 |
| — | binary_missing | — | — | — | trivy/grype/… | tool absent | confusing | scanner_missing_status | Docs + scan UI |
| — | sbom_tool_missing | — | — | — | sbom | syft absent | confusing | docs_gap | SBOM messaging |

## Sprint goals

1. Remove high `SEC-HARDCODED-SECRET` false positive on rescan
2. Downgrade/skip homelab infra refs in docs/install examples
3. `HEALTH-LARGE-FILE` → low for large operational Python
4. Group graph/QUAL-DEBUG noise in scan summary UI
5. Clarify scanner-missing and SBOM tool-missing messaging
