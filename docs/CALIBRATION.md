# Calibration (per-repository learning)

Repository Detective calibrates noisy findings **per repository** by default.

## Scoping rules

| Scope | Applies to | Expires |
|-------|------------|---------|
| Repository suppression | Single repo fingerprint/rule | Operator review or disable |
| Global suppression | Proven universal false positives only | Requires explicit approval |
| Graph homelab profile | Infra repos via `homelab_infra` scan profile | Profile-bound, not global |

## Evidence used

- Duplicate rate across scans
- Resolved-verified vs false-positive disposition
- Scanner stability (grype DB, parse vs unavailable)
- Rule actionability and developer feedback

## Safety

- High-confidence security findings are **not** downgraded by repo-local graph noise rules alone.
- Cross-repo learning may **suggest** calibration; auto-apply remains off (`calibration_auto_apply: false`).

## API / storage

- Suppressions: `finding_suppressions` + calibration recommendations tables
- Matcher cache: per-repository via `calibration.Matcher.LoadRepository`

See also: `docs/beta/CALIBRATION_BETA_POLICY.md`
