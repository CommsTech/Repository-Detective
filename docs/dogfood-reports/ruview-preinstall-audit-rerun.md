# RuView pre-install audit — rerun (post-gitleaks fix)

**Date:** 2026-06-04 (UTC)  
**Repository:** https://github.com/ruvnet/RuView  
**Commit:** `872d7593bbeeed63524386aa60e6805bb4e1b26c` (`main`)  
**Depth:** standard  
**Audit ID:** `07483617-e3a5-4df1-bef2-85e6512a1aac` (binary hot-swap validation)

**Image-recreate validation:** `bd8a34c0-daff-43d7-bff5-bbc0155d97f2` — same result (`gitleaks found`, 10 redacted).

---

## Controls

| Control | Value |
|---------|--------|
| Qdrant | off |
| LLM / AI auditors | off |
| Upstream submit | none |
| Code revision | gitleaks report-file parser fix in image `0b5005a2a2b3` (container recreated 2026-06-04) |

---

## Install recommendation

| Field | Value |
|-------|--------|
| Risk score | 100 / 100 |
| Recommendation | **Do not install until listed critical/high findings are reviewed or mitigated** |
| Stored findings | 198 |
| Duration | ~4m 30s |

---

## Scanner status (rerun)

| Scanner | Status | Findings |
|---------|--------|----------|
| **gitleaks** | **found** | **10** |
| trivy | found | 16 |
| semgrep | found | 8 |
| hadolint | found | 5 |
| grype | timed_out | 0 |
| checkov | parse_failed | 0 |
| govulncheck / gosec / staticcheck | clean | 0 |
| ruff / shellcheck | binary_missing | 0 |

### Secret scanning (completed, redacted)

Gitleaks completed successfully. **Ten** credential-like matches were reported using **redacted** output (`--redact`). Rule classes observed include `generic-api-key` and `curl-auth-header` in documentation, workflow examples, and build artifacts — **no raw secret values** are stored in the audit database or this report.

Maintainers should still validate matches locally (many may be documentation examples).

---

## Severity snapshot

| Severity | Count |
|----------|-------|
| critical | 2 |
| high | 28 (+10 gitleaks vs prior run) |
| medium | 19 |
| low | 149 |

---

## Gitleaks findings (summary, private-only)

| Pattern class | Example paths (repo-relative) |
|---------------|------------------------------|
| `curl-auth-header` | `.github/workflows/sensing-server-docker.yml`, archived docs |
| `generic-api-key` | `archive/v1/docs/*`, `.claude/agents/payments/*`, vendored `.vite` map (review whether tracked) |

All gitleaks items: **private security review only** — not for public GitHub issues.

---

## Related

- Prior audit: `dae05e0c-4c24-441e-9c05-c8ce5db4cbe0`
- Comparison: `ruview-preinstall-compare.md`
- Diagnosis: `ruview-gitleaks-parse-diagnosis.md`
