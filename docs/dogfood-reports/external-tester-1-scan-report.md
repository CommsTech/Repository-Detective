# External tester #1 — report-only scan

**Date:** 2026-06-12  
**Tester:** `ext-operator-jrice`  
**Operator scan trigger:** yes (operator executed approved report-only scan after outreach)  
**Live revision:** `rc-381667a`

---

## Scan record

| Field | Value |
|-------|-------|
| **Tester** | `ext-operator-jrice` |
| **Repository** | `commstech/Wifi_Collector` |
| **Repo ID** | 10 |
| **Scan ID** | `85a8ab62e76da076` |
| **Ref** | `main` |
| **Trigger** | manual (API) |
| **Mode** | `report_only_dry_run: true` |
| **Profile** | `standard_deterministic` |
| **Analysis depth** | 2 |
| **Graph** | enabled |
| **Duration** | ~4.6s |
| **Status** | completed |

## Results

| Metric | Value |
|--------|-------|
| Files analyzed | 26 |
| Findings (scan pipeline) | 123 |
| Severity (pipeline) | 0 critical, **1 high**, 4 medium, 109 low (+ info in UI rollup) |
| High/critical (actionable) | **1 high**, 0 critical |
| Overall score | 0.7 |
| Issues created | **0** ✓ |
| PRs created | **0** ✓ |
| Issue sync | skipped (report-only) |
| Forge open issues (repo) | **0** |

## Scanner coverage

| Scanner | Status |
|---------|--------|
| static / health / graph / linters | ran (deterministic pipeline) |
| trivy | binary_missing |
| grype | binary_missing |
| gitleaks | binary_missing |
| semgrep | binary_missing |
| ruff | binary_missing |
| gitleaks-history | failed (git clone for history) |
| govulncheck / gosec / staticcheck | clean (no Go) |
| hadolint | clean (no Dockerfiles) |
| checkov | clean (no IaC) |

**AI policy:** disabled  
**Remediation policy:** off  
**Issue policy:** off (effective via report-only)

## Notable finding categories (no secret values)

| Severity | Rule (sample) | Notes |
|----------|---------------|-------|
| high | `SEC-HARDCODED-SECRET` | 1 — operator triage: likely false positive; tester flagged via template |
| medium | `REL-INTERNAL-INFRA-REF` | 2 — homelab hostname/path references |
| medium | `HEALTH-LARGE-FILE` | 1 |
| medium | `HEALTH-TECH-MARKER` | 1 |
| low / info | `QUAL-DEBUG`, `GRAPH-*` | majority of count |

## Graph state

| Field | Value |
|-------|-------|
| State | **available** |
| Nodes | 104 |
| Edges | 111 |

Honest and populated for a small Python repo — useful for map view; contributes to list noise.

## SBOM state

| Field | Value |
|-------|-------|
| Status | `sbom_tool_missing` |
| Detail | syft not installed — install syft or use a Go module repository |
| Manifest detected | `requirements.txt` present in repo profile |
| Package count | 0 |
| Vuln count | 0 |

**Honest state:** SBOM generation blocked by missing syft in live image despite Python manifest — not a fake “clean” SBOM.

## Safety verification

| Check | Result |
|-------|--------|
| Forge issues created this scan | **0** |
| PRs created | **0** |
| Secrets in operator logs | none observed |
| Dry-run flag in summary | true |
| Reconciliation `forge_open_issues` | 0 |
| Report safe to share with tester | **yes** (redact any local paths if exporting raw JSON) |

## Logs

Container logs for scan `85a8ab62e76da076`: **clean** — no panics; expected `binary_missing` info lines; gitleaks-history clone failure logged once.

## Operator assessment

| Area | Assessment |
|------|------------|
| Scan completion | PASS |
| Report-only enforcement | PASS |
| Findings understandable | Partial — volume high for first impression |
| SBOM honesty | PASS (tool missing honestly reported) |
| Graph honesty | PASS |
| Safe for tester review | YES |

## Recommendation

Scan meets acceptance for external tester #1 (complete, 0 issues, 0 PRs). **Calibration recommended** before tester #2 due to 123 findings and 1 alarming high-severity hardcoded-secret heuristic on a homelab repo.
