# Remaining active-present triage

Recorded: 2026-06-09  
Scan: `f6102e4fed8e2b37`  
Active-present count: **89**

## Severity summary

| Severity | Count |
|----------|------:|
| info | 37 |
| low | 41 |
| medium | 11 |
| high | 0 |
| critical | 0 |

**High/critical remaining: 0** — no code fixes required in this batch for security blockers.

## Classification

| Class | Count | Notes |
|-------|------:|-------|
| needs_calibration | 77 | HEALTH-* rules (ignored errors, tech-debt markers, deprecated phrases) |
| needs_human_review | 8 | OPT-HTTP-CLIENT-PER-CALL, REL-INTERNAL-INFRA-REF |
| real_code_fix | 0 | — |
| resolved_absent_after_rescan | 0 | — |
| duplicate_or_stale_mapping | 0 | — |
| test_fixture_or_benchmark | 0 | — |
| scanner_self_match | 0 | — |
| docs_only_low_priority | 4 | HEALTH-COMMENT-BLOCK (commented code blocks) |

## Top rules

| Count | Source | Rule |
|------:|--------|------|
| 45 | reliability | HEALTH-IGNORED-ERROR |
| 15 | tech_debt | HEALTH-TECH-PHRASE |
| 11 | tech_debt | HEALTH-DEPRECATED |
| 6 | tech_debt | HEALTH-TECH-MARKER |
| 5 | static | OPT-HTTP-CLIENT-PER-CALL |
| 4 | tech_debt | HEALTH-COMMENT-BLOCK |
| 3 | static | REL-INTERNAL-INFRA-REF |

## Recommended disposition

1. **needs_calibration** — Tune health checker thresholds for product repo size; many `_ = err` patterns are intentional in orchestration code.
2. **needs_human_review** — OPT-HTTP-CLIENT-PER-CALL may be acceptable in CLI/dogfood scripts; REL-INTERNAL-INFRA-REF references homelab URLs by design.
3. **No immediate merges** — Nothing at high/critical; operator review queue is informational.

## Actions taken this sprint

- Clean rescan + external_issues sync (1 stale row repaired).
- No new issues filed.
- No high-confidence code changes (zero high/critical findings).

## Next batch

- Calibrate HEALTH-IGNORED-ERROR for large Go codebases (suppress info/low in non-production paths).
- Operator review of 8 OPT/REL medium/low items.
- Target active-present under 25 after calibration pass.
