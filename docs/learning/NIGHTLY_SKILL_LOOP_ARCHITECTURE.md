# Nightly skill loop — architecture

**Component:** `scripts/nightly-rd-skill-loop.py`  
**Pattern:** Karpathy Autoresearch adapted for calibration (fixed harness, small editable surface, keep/revert, morning digest)  
**Not in scope:** Analyzer source mutation, global security downgrades, LLM-required correctness

## Purpose

Improve Repository Detective **calibration quality** overnight by:

1. Running the fixed validation harness (Go tests, benchmark fixtures, smoke checks).
2. Ingesting compact evidence from the SQLite learning/calibration tables.
3. Synthesizing **repo-scoped** calibration candidates.
4. Auto-applying only **Tier 1** rules when promotion is enabled and all gates pass.
5. Writing machine-readable reports and a compact **operator digest** for Tier 3 / manual items.

The editable surface is **`repo_calibration_rules`** and accepted **`calibration_recommendations`** — not `analyzers/static.go` or other protected scanner code.

## Editable vs protected

| Protected (hash gate — must not change during loop) | Editable (with tier policy) |
|---------------------------------------------------|-----------------------------|
| `analyzers/static.go` | `repo_calibration_rules` (repo-scoped) |
| `analyzers/engine.go` | `calibration_recommendations` (accept via API) |
| `calibration/matcher.go` | Learning events (append-only audit) |
| `docs/beta/CALIBRATION_BETA_POLICY.md` | Candidate archive (`candidate_rules.jsonl`) |
| `.gitea/workflows/ci.yml` | Promotion decisions JSON |

## Data flow

```text
orchestration.lock
       │
       ▼
protected_hash_gate ──► baseline SHA256 of protected files
       │
       ▼
test_gate ──► go test ./... , ./benchmark/... , ./calibration/... , ./graph/...
       │
       ▼
dry_run_gate ──► operator-smoke-test.sh (+ optional report-only scans)
       │
       ▼
learning_ingest ──► rule_reliability_stats, learning_events, findings (compact aggregates)
       │
       ▼
candidate_synthesis ──► candidate_rules.jsonl (tier, repo, rule, path pattern, evidence)
       │
       ▼
tier_classification + promotion_policy (--promote / --no-promote)
       │
       ▼
adversarial_gate ──► benchmark + hardcoded-secret tests must still pass
       │
       ▼
rollback_check ──► revert last batch if post-promotion validation fails
       │
       ▼
reports/nightly-rd-evolution/latest/* + OPERATOR-DIGEST.md
```

## Tier model

| Tier | Examples | Auto-apply |
|------|----------|------------|
| **1** | Graph noise, test/vendor paths, maintainability on known paths, informational downgrade | Yes when `--promote` and gates pass |
| **2** | Broader repo-scoped rules with repeated FP evidence | After `consecutive_successful_runs >= 2` |
| **3** | Global suppressions, HIGH/CRITICAL downgrades, CVE/scanner suppressions, analyzer changes | **Never** — digest only |

Tier 3 and protected categories align with `learning.IsProtectedFromAutoDowngrade` and `docs/beta/CALIBRATION_BETA_POLICY.md`.

## Archive (Darwin-Gödel style)

| Artifact | Role |
|----------|------|
| `candidate_rules.jsonl` | Append-only candidate history |
| `promotion_decisions.json` | Per-run apply/skip/pending decisions |
| `rollback_events.json` | Append-only rollback audit |
| `full_loop_state.json` | Run metadata, consecutive success counter, promoted rule IDs |

No silent overwrite — rollbacks deactivate rules (`active=0`) and append audit entries.

## Repo isolation

- Candidates must include `repository_id` and `repo_full_name`.
- Tier 1 rules insert with `scope='repo'` and `repository_id` set.
- Global scope candidates are classified Tier 3 and never auto-applied (matches API block in `main_calibration.go`).

## Token efficiency

- No LLM required for synthesis (deterministic SQL aggregates + rule heuristics).
- Compact JSON summaries — no full scan dumps in reports.
- Evidence keyed by `repo + source + rule_id + path_pattern + fingerprint counts`.

## Related code

| Area | Path |
|------|------|
| Repo calibration apply | `findinglearn/repo_calibration.go` |
| Recommendations API | `api/calibration_handler.go`, `main_calibration.go` |
| DB schema | `store/migrations.go` (`repo_calibration_rules`, `learning_events`) |
| Operator review script | `scripts/calibration-operator-review.py` |
| Benchmark fixtures | `benchmark/fixture_benchmark_test.go` |

## Cron entrypoint

`scripts/rd-deterministic-daily.sh` → `nightly-rd-skill-loop.py --daily-mode --promote`

Disable promotion: run with `--no-promote` or omit `--promote` from cron.
