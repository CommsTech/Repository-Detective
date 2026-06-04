# RuView pre-install audit — before/after comparison

**Date:** 2026-06-04 (UTC)

| Audit | ID | Gitleaks |
|-------|-----|----------|
| Before fix | `dae05e0c-4c24-441e-9c05-c8ce5db4cbe0` | `parse_failed` (0 findings) |
| After fix (binary hot-swap) | `07483617-e3a5-4df1-bef2-85e6512a1aac` | **`found` (10 findings)** |
| After fix (image `0b5005a2a2b3` recreate) | `bd8a34c0-daff-43d7-bff5-bbc0155d97f2` | **`found` (10 findings)** |

Same repository, depth, and commit SHA (`872d7593`).

---

## Gitleaks

| Metric | Before | After |
|--------|--------|-------|
| Status | `parse_failed` | **`found`** |
| Detail | `no JSON array in output` | *(none)* |
| Findings count | 0 | **10** |
| Root cause | `--report-path -` empty on gitleaks 8.x | Temp report file + file-first parser |

---

## Totals

| Metric | Before | After | Delta |
|--------|--------|-------|-------|
| Stored findings | 198 | 198 | 0 (cap) |
| critical | 2 | 2 | 0 |
| high | 18 | **28** | **+10** (gitleaks) |
| medium | 19 | 19 | 0 |
| low | 159 | 149 | -10 (displaced by gitleaks highs) |
| Recommendation | `do_not_install` | `do_not_install` | unchanged |
| Risk score | 100 | 100 | unchanged |

---

## Public vs private classification

| Area | Before | After |
|------|--------|-------|
| Public issue drafts | 8 `general_bug` (CVE/hardening) | Same policy — **gitleaks not public** |
| Private security | Semgrep CI/JWT patterns | **+ gitleaks 10** (docs/workflow/example paths) |
| Share package wording | “gitleaks inconclusive” | **Update to “gitleaks completed; redacted”** |

---

## Report wording changes needed

| Document | Change |
|----------|--------|
| `ruview-preinstall-shareable-report.md` | Secret section: inconclusive → completed (10 redacted matches) |
| `ruview-private-security-disclosure-draft.md` | Add gitleaks pattern summary |
| `ruview-public-issue-drafts.md` | Explicitly exclude gitleaks from public issues |

No change to overall **do_not_install** recommendation tone.

---

## Remaining caveats

1. **checkov** / **grype** still incomplete — IaC/SBOM coverage partial.  
2. **Gitleaks matches** may include documentation and example curl commands — human triage required.  
3. Some gitleaks `rule_id` values embed scanner metadata paths — cosmetic DB display issue, not secret leakage.  
4. **External sharing:** Ready for **human review** after image-recreate validation audit `bd8a34c0` — confirm false positives in docs/workflows before private disclosure.

---

## Tests run (fix validation)

```bash
go test ./...        # pass (golang:1.23-alpine container)
go vet ./...         # pass
staticcheck ./...    # pre-existing S1031 in operator/runner_telemetry.go (unchanged)
```
