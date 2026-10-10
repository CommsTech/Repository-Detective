# Learning and false-positive reduction

Repository Detective improves report quality through **auditable, reversible** learning — not black-box suppression.

## Principles

- Findings remain visible unless resolved with evidence.
- High/critical severities are protected from automatic downgrade.
- Product-repo-only evidence cannot create global suppressions.
- Repo-scoped calibration requires evidence, expiry, and operator review.
- Cross-repo recommendations require evidence from multiple repositories.

## Capabilities

### 1. Scanner reliability score

Each rule/source accumulates `rule_reliability_stats`: findings generated, issues filed, false-positive rate, duplicate rate, scanner failures.

### 2. Per-repo false-positive history

`learning_events` and `repo_calibration_rules` track repo-scoped outcomes.

### 3. Cross-repo calibration recommendations

Generated only when `calibration_min_findings_for_recommendation` threshold met across repos — never from product dogfood alone.

### 4. Finding explanations

- **Why this finding exists** — rule ID, source, file evidence, reachability note.
- **Why no issue was filed** — reporting policy, backlog control, report-only dry-run, severity gate.

### 5. Actionable vs informational split

Reconciliation panel separates medium+ actionable active from info/low findings.

### 6. Rule health dashboard

UI: `/ui/learning` — events, pending recommendations, false-positive rate, scanner failure rate.

### 7. Calibration review queue

API: `GET /api/v1/calibration/recommendations` — scope, action, evidence, expiry.

Accept: `POST /api/v1/calibration/recommendations/:id/accept`

### 8. Regression safety

Store tests verify calibration rules load **before** DB transactions (deadlock prevention).

See [CALIBRATION_REVIEW_GUIDE.md](beta/CALIBRATION_REVIEW_GUIDE.md).
