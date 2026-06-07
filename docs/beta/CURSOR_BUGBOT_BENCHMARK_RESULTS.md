# Cursor Bugbot benchmark results

Date: 2026-06-02  
Fixture: `benchmark/fixture/` (controlled workspace, not production-scanned)  
Mode: deterministic harness + report-only policy (0 issues, 0 PRs)

## Repository Detective results

| Metric | Result |
|--------|--------|
| True positives (injected patterns detected by harness) | 4/4 — hardcoded secret, SQL concat, outdated dependency, structural duplicate hash match |
| False positive candidates identified | 3 — mock test secret, vendor JS, safe internal URL (classified by path/heuristic) |
| False negatives | 0 known (fixture injects verified by test) |
| Duplicate grouping | PASS — `dup_pattern_a` / `dup_pattern_b` share structural hash |
| Issue creation | 0 |
| PR creation | 0 |
| Reachability adjustment | Test path → info severity (finding remains visible) |
| LLM sanity gate | Disabled (default) |

## Harness command

```bash
go test ./benchmark/... -count=1 -v
```

## Cursor Bugbot comparison (documented capabilities, not live run)

| Dimension | Cursor Bugbot | Repository Detective (this fixture) |
|-----------|---------------|-------------------------------------|
| PR-native review | Core strength | N/A — fixture is local harness |
| Autofix agents | Supported | Not enabled (policy) |
| Self-hosted / Gitea | GitHub-primary | Yes |
| Report-only dry run | N/A | 0 issues filed |
| Structural dedup | Not measured | PASS on fixture |
| Per-repo calibration | Team/project rules | Repo-scoped rules exercised separately |
| SBOM / dependency | Not primary claim | Fixture includes outdated `requirements.txt` |

**No superiority claim** — Cursor Bugbot was not executed on this fixture (no GitHub mirror). RD results are evidence from the internal harness only.

## Actionability (operator 1–5)

| Finding type | Score | Notes |
|--------------|-------|-------|
| Hardcoded secret inject | 5 | Clear, actionable |
| SQL concat inject | 5 | Clear injection pattern |
| Mock test secret | 2 | Correctly down-ranked via test-path heuristic |
| Graph orphan (fixture) | 3 | Informational in homelab profile |

## Remaining

- Mirror fixture to GitHub and run Cursor Bugbot for side-by-side PR metrics (future batch).
