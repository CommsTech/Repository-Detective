# Homelab infra profile notes

Generated: 2026-06-07

## Purpose

Improve signal quality for homelab/infra repositories without hiding real security findings.

## Detection (`profile.IsHomelabInfra`)

A repo is treated as homelab/infra when any of:

- Layout is `infrastructure` or `documentation`
- File count ≤ 150 and manifests include docker-compose, Dockerfile, Makefile, or README
- Python repo ≤ 100 files with compose/docker manifests
- Shell repo ≤ 80 files (typical operator scripts)

## Scan profile: `homelab_infra`

Added to `store/profiles.go` — same deterministic scanner set as `standard_deterministic` with:

- `severity_gate: high`
- `confidence_gate: 0.75`
- `remediation_policy: off`
- Graph enabled with calibration hooks

Auto-detection applies even under `standard_deterministic` scans.

## Rule behavior

| Rule | Homelab behavior |
|------|------------------|
| REL-INTERNAL-INFRA-REF | Downgraded to **info** unless line contains password/token/secret/credential context |
| SEC-EVAL | **Never** downgraded |
| QUAL-DEBUG | Lower confidence in test/debug paths |
| GRAPH-ORPHAN-* / GRAPH-SUSPICIOUS-ISLAND | Downgraded to **info** in small/homelab repos with calibration notes |

## Operational entrypoints

Graph analysis treats files referenced from Makefile, Dockerfile, docker-compose, README, CI workflows, and pyproject scripts as entrypoints — not orphans.

## Not suppressed

- Secrets (gitleaks, hardcoded credentials)
- Dependency vulnerabilities (when grype DB available)
- Dangerous eval/command execution
- Findings remain in reports and DB — only severity/confidence/actionability adjusted

## Follow-up

- Ruff/shellcheck findings may need homelab severity gating when linters are installed in the scanner image.
- Full scanner image rebuild required for shellcheck on all-in-one deployments (install script updated).
