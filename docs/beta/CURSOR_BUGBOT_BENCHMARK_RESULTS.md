# Cursor Bugbot benchmark results

Date: 2026-06-07 (verified)  
Fixture: `benchmark/fixture/`  
Mode: deterministic harness; report-only policy (0 issues, 0 PRs)

## Fixture cases

| Case | File | Expected | Actual |
|------|------|----------|--------|
| Hardcoded secret | `secret_hardcoded.go.src` | True positive pattern | PASS — detected in harness |
| SQL concat | `sql_concat.go.src` | True positive | PASS |
| Outdated dependency | `requirements.txt` | SBOM/dependency candidate | PASS — pinned old requests |
| Mock test secret | `mock_secret_test.go.src` | Not real secret | PASS — test-path heuristic → info |
| Vendor JS | `vendor/minified.js` | FP candidate | PASS — vendor path classified |
| Safe internal URL | `safe_internal_url.go.src` | FP candidate | PASS — homelab URL present |
| Env fallback | `env_fallback.go.src` | Not hardcoded secret | PASS — getenv pattern |
| Structural duplicate | `dup_pattern_a/b.go.src` | Same structural hash | PASS — hashes match |
| Orphan module | `orphan_module.go.src` | Graph/dead-code candidate | PASS — UnusedHelper present |

## Metrics

| Metric | Result |
|--------|--------|
| True positives | 4/4 harness checks (secret, SQL, dependency, structural dup) |
| False positives identified | 3 (mock secret, vendor JS, internal URL) |
| False negatives | 0 known |
| Structural grouping | PASS |
| Issue creation | 0 |
| PR creation | 0 |
| Learning events from fixture | N/A — fixture not registered in Gitea |
| Global calibration | None applied |

## Command

```bash
go test ./benchmark/... -count=1 -v
```

Output: **PASS** (2026-06-07)

## Repository Detective strengths (evidence-based)

- Self-hosted deterministic harness
- Structural dedup without LLM
- Reachability/test-path down-ranking (finding stays visible)
- Report-only mode

## Cursor Bugbot strengths (documented, not measured here)

- GitHub/Cursor PR-native workflow
- Autofix agent loop
- Cloud-scale PR throughput

**No superiority claim** — Cursor Bugbot was not run on this fixture.
