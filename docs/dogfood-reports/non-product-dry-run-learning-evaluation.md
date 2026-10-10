# Non-product dry-run learning evaluation

Generated: 2026-06-07  
Repos: `commstech/nextcloud_scripts` (small), `commstech/netmapper` (medium)

## Executive summary

Both report-only dry runs completed successfully. Repository Detective **persisted findings and scan metadata** while **blocking all Gitea issue creation**. The medium repo produced a realistic finding mix that validates scanner coverage but also exposes **graph-rule noise** that must be calibrated before any limited issue filing.

## Top actionable findings

### Small — `nextcloud_scripts`

No findings. Scanner pipeline validated end-to-end with zero false positives.

### Medium — `netmapper`

| Priority | Rule | Count | Recommendation |
|----------|------|------:|----------------|
| P0 | SEC-EVAL | 1 | Review `eval()` — only finding at critical severity |
| P1 | HEALTH-PY-NO-TEST | 5 | Add tests for core modules if filing later |
| P2 | REL-INTERNAL-INFRA-REF | 12 | Confirm homelab hostnames are intentional |
| P3 | QUAL-DEBUG | 18 | Strip debug prints before production use |
| P3 | OPT-NESTED-LOOP | 3 | Optional performance cleanup |

## Noisy findings

| Repo | Category | Count | % of total |
|------|----------|------:|-----------:|
| netmapper | GRAPH-ORPHAN-FILE | 36 | 41% |
| netmapper | GRAPH-SUSPICIOUS-ISLAND | 12 | 14% |
| netmapper | QUAL-DEBUG (low) | 18 | 21% |

**Assessment:** Graph heuristics would dominate issue backlog if filed verbatim. Recommend suppressing or down-ranking graph orphan/island for repos under ~100 files until repo map maturity improves.

## Scanner stability

| Scanner | Small | Medium | Variance explanation |
|---------|-------|--------|----------------------|
| grype | parse_failed | parse_failed | JSON parse error in container — env/tooling, not repo-specific |
| shellcheck | binary_missing | n/a | Not in image — shell repos under-covered |
| ruff | n/a | binary_missing | Python linter absent — static rules compensate partially |
| trivy/gitleaks/semgrep | clean | clean | Stable |
| static/health/graph | clean | found | Expected for Python medium repo |

## Duplicate prevention

- Dedup: 113 → 87 on netmapper (26% merged).
- Gitea open count unchanged on both repos.
- No duplicate issue burst.
- `issue_sync=skipped` correctly reflects report-only mode.

## Confidence scoring

- SEC-EVAL critical appears well-targeted (single high-confidence security rule).
- Graph findings likely **medium/low confidence** for homelab utilities — need repo-profile tuning.
- REL-INTERNAL-INFRA-REF may be **expected** in homelab context — suppression by profile recommended.

## Calibration recommendations

1. **Graph noise gate:** Raise threshold or disable GRAPH-ORPHAN-FILE / GRAPH-SUSPICIOUS-ISLAND for repos with `<100` files or `primary_ecosystem=python` homelab profile.
2. **Install ruff + shellcheck** in scanner image for complete Python/shell coverage.
3. **Fix grype parse** — investigate stderr contamination in JSON output.
4. **Homelab profile:** Add reporting hint to downgrade REL-INTERNAL-INFRA-REF when `docs_security=false` and repo size < 1 MB.
5. **Issue filing gate:** When enabling limited filing, start with **critical + high only** and backlog-control on.

## Report quality (repo owner perspective)

| Aspect | Small | Medium |
|--------|-------|--------|
| Executive clarity | N/A (clean) | Good — severity breakdown actionable |
| Repo map usefulness | Minimal (1 file) | Useful — 164 nodes / 258 edges |
| Actionability | — | SEC-EVAL + test gaps clear; graph noise dilutes signal |
| Understandability | High | Medium — owner would need graph noise explanation |

## Learning from prior product work

- Fingerprint lifecycle and backlog-control **did not interfere** with non-product dry runs.
- Report-only flag correctly overrides global `auto_create_issues: true`.
- Context propagation bug (fixed in `87800af`) was caught early on small repo — important for fleet safety.

## Would limited issue filing be safe later?

**Not yet.** Technical guardrails work, but:

- Graph noise (~55% of netmapper findings) would create poor first impression.
- Missing ruff/shellcheck reduces Python/shell confidence.
- grype parse failure leaves dependency scan gap.
- **Operator explicit approval still required.**

Recommend **one more report-only iteration** on a second medium repo (different language) after graph calibration + scanner image fixes, then re-evaluate.
