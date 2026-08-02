# Unknown / eligible unmapped finding audit

Generated: 2026-07-12T14:52:57Z

## Overnight fleet context

After the first scheduled fleet window (03:30–04:25 UTC), expect stale scans near zero,
mapped issues to rise only when filing policies allow, and `unknown` to stay at 0.

## Reclassification summary

- Focus set (eligible_to_file / unknown / fixture-flagged): **19**
- True `eligible_to_file` (canary-safe heuristic): **0**
- Likely fixture FP (paths/tests): **19**
- Remaining literal `unknown`: **0**

Former `unknown` findings were reclassified primarily as:
- `already_mapped` — closed forge issue already exists (re-open/reconcile, do not re-file)
- `below_threshold` / `report_only` / `duplicate` — policy or dedup
- `eligible_to_file` + `likely_fixture_fp` — test fixtures, benchmark paths, or docs quoting secrets/eval

### By repo

| Repo | Count | Eligible | Fixture FP | Severities | Top rules |
|------|-------|----------|------------|------------|-----------|
| commstech/Repository-Detective | 17 | 0 | 17 | {'high': 17} | GITLEAKS-/tmp/rd-scan-264885933/redact/secrets_test.go:aws-access-token:13, GITLEAKS-/tmp/rd-archive-763177047/redact/secrets_test.go:aws-access-token:13, GITLEAKS-/tmp/rd-archive-2145926563/redact/secrets_test.go:aws-access-token:13 |
| commstech/Luna-Assist | 2 | 0 | 2 | {'high': 1, 'critical': 1} | GOV-PIPELINE-SECRET-ECHO, SEC-EVAL |

### By rule ID

| Rule | Count | Severities | Repos |
|------|-------|------------|-------|
| `GITLEAKS-/tmp/rd-scan-264885933/redact/secrets_test.go:aws-access-token:13` | 1 | {'high': 1} | commstech/Repository-Detective |
| `GITLEAKS-/tmp/rd-archive-763177047/redact/secrets_test.go:aws-access-token:13` | 1 | {'high': 1} | commstech/Repository-Detective |
| `GITLEAKS-/tmp/rd-archive-2145926563/redact/secrets_test.go:aws-access-token:13` | 1 | {'high': 1} | commstech/Repository-Detective |
| `GITLEAKS-/tmp/rd-archive-3117434377/redact/secrets_test.go:aws-access-token:13` | 1 | {'high': 1} | commstech/Repository-Detective |
| `GITLEAKS-/tmp/rd-archive-3175949230/redact/secrets_test.go:aws-access-token:13` | 1 | {'high': 1} | commstech/Repository-Detective |
| `GITLEAKS-/tmp/rd-archive-2516315045/redact/secrets_test.go:aws-access-token:13` | 1 | {'high': 1} | commstech/Repository-Detective |
| `GITLEAKS-/tmp/rd-archive-1905352843/redact/secrets_test.go:aws-access-token:13` | 1 | {'high': 1} | commstech/Repository-Detective |
| `GITLEAKS-/tmp/rd-archive-904953334/redact/secrets_test.go:aws-access-token:13` | 1 | {'high': 1} | commstech/Repository-Detective |
| `GITLEAKS-/tmp/rd-archive-549578119/redact/secrets_test.go:aws-access-token:13` | 1 | {'high': 1} | commstech/Repository-Detective |
| `GITLEAKS-/tmp/rd-archive-737799090/redact/secrets_test.go:aws-access-token:13` | 1 | {'high': 1} | commstech/Repository-Detective |
| `GITLEAKS-/tmp/rd-archive-1859889500/redact/secrets_test.go:aws-access-token:13` | 1 | {'high': 1} | commstech/Repository-Detective |
| `GITLEAKS-/tmp/rd-archive-66244356/redact/secrets_test.go:aws-access-token:13` | 1 | {'high': 1} | commstech/Repository-Detective |
| `GITLEAKS-/tmp/rd-archive-756331478/redact/secrets_test.go:aws-access-token:13` | 1 | {'high': 1} | commstech/Repository-Detective |
| `GITLEAKS-/tmp/rd-archive-3878725477/redact/secrets_test.go:aws-access-token:13` | 1 | {'high': 1} | commstech/Repository-Detective |
| `GITLEAKS-/tmp/rd-archive-1957051215/redact/secrets_test.go:aws-access-token:13` | 1 | {'high': 1} | commstech/Repository-Detective |
| `TRIVY-CVE-2025-66471` | 1 | {'high': 1} | commstech/Repository-Detective |
| `GITLEAKS-/tmp/rd-archive-871140987/redact/secrets_test.go:aws-access-token:13` | 1 | {'high': 1} | commstech/Repository-Detective |
| `GOV-PIPELINE-SECRET-ECHO` | 1 | {'high': 1} | commstech/Luna-Assist |
| `SEC-EVAL` | 1 | {'critical': 1} | commstech/Luna-Assist |

### Detail rows

| ID | Repo | Sev | Conf | Rule | Path | Reason | Notes |
|----|------|-----|------|------|------|--------|-------|
| 10674 | commstech/Repository-Detective | high | 0.75 | `GITLEAKS-/tmp/rd-scan-264885933/redact/secrets_test.go:aws-access-token:13` | `redact/secrets_test.go:13` | eligible_to_file | likely_fixture_fp |
| 12356 | commstech/Repository-Detective | high | 0.75 | `GITLEAKS-/tmp/rd-archive-763177047/redact/secrets_test.go:aws-access-token:13` | `redact/secrets_test.go:13` | eligible_to_file | likely_fixture_fp |
| 13392 | commstech/Repository-Detective | high | 0.75 | `GITLEAKS-/tmp/rd-archive-2145926563/redact/secrets_test.go:aws-access-token:13` | `redact/secrets_test.go:13` | eligible_to_file | likely_fixture_fp |
| 14429 | commstech/Repository-Detective | high | 0.75 | `GITLEAKS-/tmp/rd-archive-3117434377/redact/secrets_test.go:aws-access-token:13` | `redact/secrets_test.go:13` | eligible_to_file | likely_fixture_fp |
| 15466 | commstech/Repository-Detective | high | 0.75 | `GITLEAKS-/tmp/rd-archive-3175949230/redact/secrets_test.go:aws-access-token:13` | `redact/secrets_test.go:13` | eligible_to_file | likely_fixture_fp |
| 16508 | commstech/Repository-Detective | high | 0.75 | `GITLEAKS-/tmp/rd-archive-2516315045/redact/secrets_test.go:aws-access-token:13` | `redact/secrets_test.go:13` | eligible_to_file | likely_fixture_fp |
| 17596 | commstech/Repository-Detective | high | 0.75 | `GITLEAKS-/tmp/rd-archive-1905352843/redact/secrets_test.go:aws-access-token:13` | `redact/secrets_test.go:13` | eligible_to_file | likely_fixture_fp |
| 18685 | commstech/Repository-Detective | high | 0.75 | `GITLEAKS-/tmp/rd-archive-904953334/redact/secrets_test.go:aws-access-token:13` | `redact/secrets_test.go:13` | eligible_to_file | likely_fixture_fp |
| 19787 | commstech/Repository-Detective | high | 0.75 | `GITLEAKS-/tmp/rd-archive-549578119/redact/secrets_test.go:aws-access-token:13` | `redact/secrets_test.go:13` | eligible_to_file | likely_fixture_fp |
| 20899 | commstech/Repository-Detective | high | 0.75 | `GITLEAKS-/tmp/rd-archive-737799090/redact/secrets_test.go:aws-access-token:13` | `redact/secrets_test.go:13` | eligible_to_file | likely_fixture_fp |
| 21996 | commstech/Repository-Detective | high | 0.75 | `GITLEAKS-/tmp/rd-archive-1859889500/redact/secrets_test.go:aws-access-token:13` | `redact/secrets_test.go:13` | eligible_to_file | likely_fixture_fp |
| 23065 | commstech/Repository-Detective | high | 0.75 | `GITLEAKS-/tmp/rd-archive-66244356/redact/secrets_test.go:aws-access-token:13` | `redact/secrets_test.go:13` | eligible_to_file | likely_fixture_fp |
| 24125 | commstech/Repository-Detective | high | 0.75 | `GITLEAKS-/tmp/rd-archive-756331478/redact/secrets_test.go:aws-access-token:13` | `redact/secrets_test.go:13` | eligible_to_file | likely_fixture_fp |
| 25199 | commstech/Repository-Detective | high | 0.75 | `GITLEAKS-/tmp/rd-archive-3878725477/redact/secrets_test.go:aws-access-token:13` | `redact/secrets_test.go:13` | eligible_to_file | likely_fixture_fp |
| 26286 | commstech/Repository-Detective | high | 0.75 | `GITLEAKS-/tmp/rd-archive-1957051215/redact/secrets_test.go:aws-access-token:13` | `redact/secrets_test.go:13` | eligible_to_file | likely_fixture_fp |
| 27217 | commstech/Repository-Detective | high | 0.98 | `TRIVY-CVE-2025-66471` | `benchmark/fixture/requirements.txt:0` | eligible_to_file | likely_fixture_fp |
| 27767 | commstech/Repository-Detective | high | 0.75 | `GITLEAKS-/tmp/rd-archive-871140987/redact/secrets_test.go:aws-access-token:13` | `redact/secrets_test.go:13` | eligible_to_file | likely_fixture_fp |
| 40562 | commstech/Luna-Assist | critical | 0.7 | `SEC-EVAL` | `archive/session_summaries/comprehensive_code_review_report.md:97` | eligible_to_file | likely_fixture_fp |
| 40454 | commstech/Luna-Assist | high | 0.5 | `GOV-PIPELINE-SECRET-ECHO` | `FIX_SESSION_SUMMARY_2025-10-13.md:367` | eligible_to_file | likely_fixture_fp |

## Canary gate

**No safe canary repo.** Remaining eligible findings are HIGH/CRITICAL security (secrets/eval/cmd/sql) or fixture noise — do not `--apply` until operator triage.
