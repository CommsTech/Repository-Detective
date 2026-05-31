# Documentation index

Repository Detective — **Inspect. Analyze. Improve.**

Operator and developer documentation for the full safe remediation loop:

```text
detect → issue → plan → approve → patch PR → merge → rescan → verified closure
```

## Getting started

| Document | Description |
|----------|-------------|
| [SETUP.md](SETUP.md) | Installation and first run |
| [ONBOARDING.md](ONBOARDING.md) | Repository onboarding workflow |
| [OPERATOR_READINESS.md](OPERATOR_READINESS.md) | Pre-deployment checklist (binaries, config, backups) |
| [DOGFOODING.md](DOGFOODING.md) | **Track A** — scan Repository Detective itself first |
| [DOGFOOD_REPORT_TEMPLATE.md](DOGFOOD_REPORT_TEMPLATE.md) | Report template after first self-scan |
| [TROUBLESHOOTING.md](TROUBLESHOOTING.md) | Common operator issues |

## Product and policy

| Document | Description |
|----------|-------------|
| [NAMING.md](NAMING.md) | Product name, legacy Bugbot compatibility |
| [POLICY.md](POLICY.md) | Severity gates, issue policy, remediation policy |
| [SCAN_PROFILES.md](SCAN_PROFILES.md) | Scan profile presets (`fast`, `standard_deterministic`, etc.) |
| [SECURITY_HARDENING.md](SECURITY_HARDENING.md) | OWASP baseline + post-remediation safety checklist |

## Scanning and analysis

| Document | Description |
|----------|-------------|
| [SCANNERS.md](SCANNERS.md) | Deterministic scanner tools and configuration |
| [HEALTH_CHECKS.md](HEALTH_CHECKS.md) | Repository health checks (tech debt, reliability, etc.) |
| [CODE_GRAPH.md](CODE_GRAPH.md) | Optional code graph analysis |
| [PREINSTALL_AUDIT.md](PREINSTALL_AUDIT.md) | Third-party repo audit before install (no scripts/deps) |

## Operations

| Document | Description |
|----------|-------------|
| [DATABASE.md](DATABASE.md) | SQLite schema, migrations, backup guidance |
| [UI.md](UI.md) | Operator web UI |
| [RUNNERS.md](RUNNERS.md) | Remote runner delegation |
| [NOTIFICATIONS.md](NOTIFICATIONS.md) | Webhook notifications (redacted payloads) |
| [GITEA_STATUS.md](GITEA_STATUS.md) | Gitea commit status integration |

## Safe remediation loop

| Document | Description |
|----------|-------------|
| [REMEDIATION.md](REMEDIATION.md) | Remediation planner (plans only — no auto-fix) |
| [REMEDIATION_PRS.md](REMEDIATION_PRS.md) | Safe remediation PRs (branch + PR only) |
| [EVIDENCE_CLOSURE.md](EVIDENCE_CLOSURE.md) | Evidence-based issue closure after merge + rescan |

## Example configurations

See [examples/](examples/) for copy-paste YAML profiles:

- `homelab-minimal.yaml` — smallest useful deployment
- `deterministic-standard.yaml` — recommended default scanners
- `dogfood-repository-detective.yaml` — **Track A** self-scan config
- `strict-security-gate.yaml` — high-severity gates, more scanners
- `runner-enabled.yaml` — runner delegation for heavy scans
- `preinstall-only.yaml` — audit third-party repos without connected scanning
- `remediation-pr-safe.yaml` — planner + safe PR + evidence closure

## Status endpoints

| Endpoint | Purpose |
|----------|---------|
| `GET /health` | Liveness; includes feature/tool summary when ready |
| `GET /api/v1/status` | Full operator readiness JSON (no secrets) |
| `GET /api/v1/about` | Product name, version, compatibility, safe loop |

## Legacy / reference

| Document | Description |
|----------|-------------|
| [BRANDING_MIGRATION.md](BRANDING_MIGRATION.md) | Bugbot → Repository Detective migration |
| [SCANNER_ROADMAP.md](SCANNER_ROADMAP.md) | Future scanner plans (not shipped) |
| [REPO_GUARDIAN_ARCHITECTURE.md](REPO_GUARDIAN_ARCHITECTURE.md) | Historical architecture notes (superseded naming) |

## Testing

See [TESTING.md](TESTING.md) for unit/integration test commands.

```bash
go test ./...
go vet ./...
staticcheck ./...
```

Optional: `gosec ./...` when installed locally.
