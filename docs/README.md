# Documentation index

Repository Detective — **Inspect. Analyze. Improve.**

Operator and developer documentation for the full safe remediation loop:

```text
detect → issue → plan → approve → patch PR → merge → rescan → verified closure
```

## Private beta (start here)

| Document | Description |
|----------|-------------|
| [QUICKSTART.md](QUICKSTART.md) | **Recommended Installation** — zero to first scan (~15 min) |
| [PUBLIC_BETA.md](PUBLIC_BETA.md) | Public community beta — try it + [GitHub feedback](https://github.com/CommsTech/Repository-Detective/issues/new/choose) |
| [SECURITY.md](../SECURITY.md) | Product vulnerability reporting (private advisory) |
| [AGENT_QUICKSTART.md](AGENT_QUICKSTART.md) | AI agents — REST + MCP for OpenClaw / automations |
| [MCP.md](MCP.md) | MCP stdio bridge (`repository-detective-mcp`) |
| [OPENCLAW_INTEGRATION.md](OPENCLAW_INTEGRATION.md) | RD↔OpenClaw both directions |
| [BETA_READINESS.md](BETA_READINESS.md) | Go/no-go checklist and beta config |
| [BETA_SMOKE_TEST.md](BETA_SMOKE_TEST.md) | End-to-end operator validation |
| [TEST_MATRIX.md](TEST_MATRIX.md) | Full regression test areas |
| [CONFIGURATION.md](CONFIGURATION.md) | `.env` + `config.yaml` reference |
| [DEPLOYMENT.md](DEPLOYMENT.md) | Deployment index |
| [DOCKER.md](DOCKER.md) | Image targets and build |
| [RELEASE_NOTES_0.1.0_BETA.md](RELEASE_NOTES_0.1.0_BETA.md) | Beta release notes |
| [DOCS_AUDIT.md](DOCS_AUDIT.md) | Documentation completeness audit |
| [DOC_TRUTH_AUDIT.md](DOC_TRUTH_AUDIT.md) | RD-029 capability claims vs source (STABLE/BETA/…) |
| [FEATURE_COMPLETENESS_AUDIT.md](FEATURE_COMPLETENESS_AUDIT.md) | Feature inventory and gap audit |
| [API_ROUTES.md](API_ROUTES.md) | API route reference |
| [openapi.yaml](openapi.yaml) | OpenAPI 3 (also `GET /api/v1/openapi.yaml`) |
| [AI_RECOMMENDATIONS.md](AI_RECOMMENDATIONS.md) | Advisory AI recommendations |
| [AI_PROVIDERS.md](AI_PROVIDERS.md) | OpenClaw / OpenAI-compatible provider setup |

Scripts: `./scripts/release-test.sh` · `./scripts/operator-smoke-test.sh` · `go build ./cmd/repository-detective-mcp`

## Getting started

| Document | Description |
|----------|-------------|
| [SETUP.md](SETUP.md) | Installation and first run |
| [ONBOARDING.md](ONBOARDING.md) | Connect→Select→Protect→Verify→Ready wizard (RD-013) |
| [DOCTOR.md](DOCTOR.md) | Doctor CLI/API/UI diagnostics (RD-014) |
| [RD-008B_CLASS_B_EXECUTION.md](RD-008B_CLASS_B_EXECUTION.md) | Class-B remediation execution decision |
| [OPERATOR_READINESS.md](OPERATOR_READINESS.md) | Pre-deployment checklist (binaries, config, backups) |
| [DOGFOODING.md](DOGFOODING.md) | **Track A** — scan Repository Detective itself first |
| [DOGFOOD_REPORT_TEMPLATE.md](DOGFOOD_REPORT_TEMPLATE.md) | Report template after first self-scan |
| [TROUBLESHOOTING.md](TROUBLESHOOTING.md) | Common operator issues |
| [DASHBOARD_GUIDE.md](DASHBOARD_GUIDE.md) | Operator dashboard and charts |
| [SCANNER_HEALTH.md](SCANNER_HEALTH.md) | Scanner availability and degraded coverage |
| [PRIVACY_AND_DATA_PROTECTION.md](PRIVACY_AND_DATA_PROTECTION.md) | Privacy-aware handling (not compliance certification) |
| [PRIVACY_MODES.md](PRIVACY_MODES.md) | LOCAL_ONLY / HYBRID / EXTERNAL_AI operating modes (RD-007) |
| [ADMIN_HARDENING.md](ADMIN_HARDENING.md) | Operator security checklist |
| [DATA_RETENTION.md](DATA_RETENTION.md) | Retention and deletion responsibilities |
| [ACCESSIBILITY.md](ACCESSIBILITY.md) | WCAG-aligned UI practices (not certification) |
| [ISSUE_TRACKING.md](ISSUE_TRACKING.md) | Gitea backlog, templates, labels |
| [RELEASE_READINESS.md](RELEASE_READINESS.md) | Pre-release checklist |
| [COMPLIANCE_READINESS.md](COMPLIANCE_READINESS.md) | Compliance evidence index |
| [KNOWN_LIMITATIONS.md](KNOWN_LIMITATIONS.md) | Honest product limits |
| [SECURITY_MODEL.md](SECURITY_MODEL.md) | Trust boundaries + threat model honesty (RD-008) |
| [PR_SUMMARY_IDEMPOTENCY.md](PR_SUMMARY_IDEMPOTENCY.md) | Idempotent PR policy summary comments (RD-006A) |
| [DOC_TRUTH_AUDIT.md](DOC_TRUTH_AUDIT.md) | Claims vs runtime proof levels |
| [SECURITY_CHECK_MATRIX.md](SECURITY_CHECK_MATRIX.md) | Ten minimum checks vs shipped tools |
| [PRE_PUBLISH_CHECKS.md](PRE_PUBLISH_CHECKS.md) | Pre-public release checklist |
| [PIPELINE_GOVERNANCE.md](PIPELINE_GOVERNANCE.md) | CI/workflow and runner governance |
| [OPTIMIZATION_CHECKS.md](OPTIMIZATION_CHECKS.md) | Advisory optimization rules |
| [DOC_DETECTIVE_REVIEW.md](DOC_DETECTIVE_REVIEW.md) | Doc Detective integration review |
| [WIKI_PUBLISHING.md](WIKI_PUBLISHING.md) | Gitea wiki sync (`docs/wiki/` copies) |

## Product and policy

| Document | Description |
|----------|-------------|
| [NAMING.md](NAMING.md) | Product name, naming conventions |
| [POLICY.md](POLICY.md) | Severity gates, issue policy, remediation policy |
| [SCAN_PROFILES.md](SCAN_PROFILES.md) | Scan profiles: Light, Standard, Deep, Custom |
| [SECURITY_HARDENING.md](SECURITY_HARDENING.md) | OWASP baseline + post-remediation safety checklist |

## Scanning and analysis

| Document | Description |
|----------|-------------|
| [SCANNERS.md](SCANNERS.md) | Deterministic scanner tools and configuration |
| [REPORTING.md](REPORTING.md) | Repo profiling, reporting modes, and forge issue gating |
| [FALSE_POSITIVES.md](FALSE_POSITIVES.md) | Tuning heuristics and avoiding noisy Gitea issues |
| [SBOM.md](SBOM.md) | Pinned versions and software bill of materials |
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
| [ISSUE_BACKLOG.md](ISSUE_BACKLOG.md) | Feature issue tracking vs shipped work |

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
| [BRANDING_MIGRATION.md](BRANDING_MIGRATION.md) | Repository-Detective → Repository Detective migration |
| [BRANDING_COMPATIBILITY_AUDIT.md](BRANDING_COMPATIBILITY_AUDIT.md) | Preferred vs legacy naming audit |
| [EDITIONS.md](EDITIONS.md) | Community / Commercial / Enterprise matrix |
| [LICENSING_STRATEGY.md](LICENSING_STRATEGY.md) | AGPL Community + commercial editions |
| [COMMUNITY_EDITION.md](COMMUNITY_EDITION.md) | Community scope and limits |
| [COMMERCIAL_ENTERPRISE.md](COMMERCIAL_ENTERPRISE.md) | Paid tiers and feature-gate plan |
| [SCANNER_ROADMAP.md](SCANNER_ROADMAP.md) | Future scanner plans (not shipped) |
| [REPO_GUARDIAN_ARCHITECTURE.md](REPO_GUARDIAN_ARCHITECTURE.md) | Historical architecture notes (superseded naming) |

## Testing

See [TESTING.md](TESTING.md) for unit/integration test commands.

For a full integration and scan-readiness checklist, see [INTEGRATION_AUDIT.md](INTEGRATION_AUDIT.md). Run `./scripts/verify-all.sh` locally (mirrors CI + govulncheck).

```bash
go test ./...
go vet ./...
staticcheck ./...
```

Optional: `gosec ./...` when installed locally.
