# RuView gitleaks human triage

**Audit:** `bd8a34c0-daff-43d7-bff5-bbc0155d97f2`  
**Repository:** https://github.com/ruvnet/RuView @ `872d7593bbeeed63524386aa60e6805bb4e1b26c`  
**Scanner:** gitleaks 8.21.2 (`--redact`, report-file parser)  
**Date:** 2026-06-04 (UTC)

> **No raw secret values below.** Redacted-match column reflects gitleaks `--redact` behavior; the audit database stores rule titles only, not secret material.

---

## Summary counts

| Disposition | Count |
|-------------|-------|
| `likely_false_positive` | **8** |
| `needs_maintainer_review` | **2** |
| `private_disclosure_only` (all gitleaks) | **10** |
| `public-safe` | **0** |

---

## Triage table

| # | File path | Rule / detector | Redacted match | Context type | Recommended disposition | Public-safe | Notes |
|---|-----------|-----------------|----------------|--------------|-------------------------|-------------|-------|
| 1 | `.claude/agents/payments/agentic-payments.md` (L35) | `generic-api-key` | `[redacted — not stored]` | documentation/example | likely_false_positive | no | Agent prompt/metadata markdown; typical placeholder or illustrative API key pattern |
| 2 | `.github/workflows/sensing-server-docker.yml` (L162) | `curl-auth-header` | `[redacted — not stored]` | workflow/config | needs_maintainer_review | no | CI workflow curl with Authorization-style header; may be example token or test credential — maintainer must confirm |
| 3 | `archive/v1/docs/api/rest-endpoints.md` (L81) | `generic-api-key` | `[redacted — not stored]` | documentation/example | likely_false_positive | no | Archived API doc sample request |
| 4 | `archive/v1/docs/api/rest-endpoints.md` (L84) | `generic-api-key` | `[redacted — not stored]` | documentation/example | likely_false_positive | no | Same file, second illustrative example |
| 5 | `archive/v1/docs/api_reference.md` (L65) | `generic-api-key` | `[redacted — not stored]` | documentation/example | likely_false_positive | no | Archived API reference sample |
| 6 | `archive/v1/docs/security-features.md` (L101) | `curl-auth-header` | `[redacted — not stored]` | documentation/example | likely_false_positive | no | Security doc curl example with auth header |
| 7 | `archive/v1/docs/user-guide/api-reference.md` (L45) | `generic-api-key` | `[redacted — not stored]` | documentation/example | likely_false_positive | no | User-guide API example |
| 8 | `v2/crates/homecore-api/README.md` (L107) | `curl-auth-header` | `[redacted — not stored]` | documentation/example | likely_false_positive | no | README curl usage example |
| 9 | `v2/crates/homecore-api/README.md` (L111) | `curl-auth-header` | `[redacted — not stored]` | documentation/example | likely_false_positive | no | Second README curl example |
| 10 | `v2/crates/wifi-densepose-desktop/ui/.vite/deps/chunk-YQTFE5VL.js.map` (L4) | `generic-api-key` | `[redacted — not stored]` | test fixture / build artifact | likely_false_positive | no | Vite dependency source map (generated); consider `.gitignore` / not tracking build artifacts |

---

## Operator interpretation

- **None of the 10 findings should be treated as confirmed live credential exposure** based on path/context alone.
- **Two items** (workflow #2, and optionally #10 if artifact is tracked unintentionally) warrant **maintainer confirmation**.
- **All gitleaks items remain private-disclosure scope** — do not reference in public GitHub issues.

---

## Recommended maintainer actions (for private disclosure)

1. Run `gitleaks dir . --redact` locally at commit `872d7593` and compare with this table.
2. Replace illustrative keys in docs with obvious placeholders (`YOUR_API_KEY_HERE`).
3. Review workflow #2 for real vs example Authorization headers; use GitHub secrets for any live values.
4. Confirm whether `.vite/deps` artifacts should be tracked; add to `.gitignore` if generated.

---

## External-share readiness (gitleaks slice)

| Question | Answer |
|----------|--------|
| Ready to claim “no secrets found”? | **No** — patterns detected; triage suggests mostly examples |
| Ready for private disclosure mentioning gitleaks? | **Yes** — with “redacted patterns, maintainer review recommended” wording |
| Ready for public issues citing gitleaks hits? | **No** |

See `ruview-preinstall-shareable-report.md` for full package readiness.
