# Continuous learning design

Repository Detective improves through **evidence-backed, deterministic learning** — not black-box ML.

## Goals

- Reduce false positives and duplicate findings/issues
- Improve scanner reliability and per-repo accuracy
- Improve report actionability while preserving security sensitivity
- Make every learned decision auditable, reversible, and expirable
- Avoid overfitting, token burn, and hidden suppression

## Non-goals

- No autonomous global suppression
- No mandatory LLM for scan correctness
- No hidden finding deletion
- No HIGH/CRITICAL security downgrade without explicit operator override
- No cross-repo overfitting from one homelab repo
- No issue filing from unverified ML-only claims

## Architecture

```text
Lifecycle outcomes → learning_events (append-only, idempotent)
                  → rule_reliability_stats (per-repo by default)
                  → calibration recommendations (proposed, repo-scoped)
                  → repo_calibration_rules (operator-approved only)
                  → display/confidence/issue eligibility (findings remain visible)
```

## Scoping

| Scope | Default | Global requires |
|-------|---------|-----------------|
| Learning events | per repo | n/a |
| Recommendations | per repo | multi-repo evidence + explicit flag |
| Applied rules | per repo | operator approval |
| Structural dedup | per repo scan | canonical finding in same repo |

## Optional LLM sanity gate

Disabled by default. Low/medium findings only. Token-capped. Redacted snippets. Never downgrades HIGH/CRITICAL automatically.

## ClawHub pattern mapping

| Self-improving agent pattern | Repository Detective equivalent |
|------------------------------|--------------------------------|
| Log failures/corrections | `learning_events` |
| Review before major tasks | Pending recommendations UI |
| Self-reflection | Deterministic stats + optional LLM gate |
| Learn from outcomes | Closure, suppressions, scanner health |

## Related docs

- [LEARNING_HEALTH.md](LEARNING_HEALTH.md)
- [../CALIBRATION.md](../CALIBRATION.md)
- [../beta/LEARNING_BETA_READINESS.md](../beta/LEARNING_BETA_READINESS.md)
