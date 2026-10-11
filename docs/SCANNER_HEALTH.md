# Scanner health and availability

Repository Detective separates **application health** (service up, DB reachable) from **scanner coverage** (which tools are configured vs installed).

## Concepts

| Term | Meaning |
|------|---------|
| **Configured** | Scanner enabled in effective config (`enable_trivy`, etc.) |
| **Installed / available** | Binary found on `PATH` at probe time |
| **Missing** | Configured but binary not found |
| **Optional / inactive** | Not configured — skipped by design (e.g. `hadolint`, `checkov` when disabled) |
| **Degraded coverage** | One or more configured scanners missing — scans continue with remaining tools |
| **Version unknown** | Binary runs but version string empty or unparseable (e.g. semgrep, gosec `dev`) |

## Where to check

| Surface | Path |
|---------|------|
| System Health UI | `/ui/health` |
| Dashboard | Scanner coverage table |
| API | `GET /api/v1/status` → `tools[]` |
| Liveness | `GET /health` → `tools_summary` when ready |

## Probe behavior

- Implemented in `operator.CheckTools` ([operator/scanners.go](../operator/scanners.go))
- **2 second timeout** per version command — probes do not block the dashboard indefinitely
- Version output truncated to 120 characters for safe display
- Failures do not log environment variables

## Example interpretation (validation case)

| Scanner | Configured | Installed | Notes |
|---------|------------|-----------|-------|
| git | yes | available | Required for clones |
| trivy | yes | missing | **Degraded** — install or disable |
| grype | yes | available | OK |
| gitleaks | yes | available | OK |
| semgrep | yes | available | Version may show `unknown` |
| govulncheck | yes | available | OK |
| gosec | yes | available | `dev` version is valid raw output |
| staticcheck | yes | available | OK |
| hadolint | no | missing | **Inactive** — not an error |
| checkov | no | missing | **Inactive** — not an error |

## Remediation

**Configured but missing (e.g. trivy):**

```text
trivy is configured but not installed. Install trivy or disable enable_trivy in config.
```

Rebuild container with external tools:

```bash
INSTALL_EXTERNAL_TOOLS=true ./deploy.sh
```

**Not configured (e.g. hadolint):**

```text
hadolint is not configured; Dockerfile linting is currently skipped.
```

Enable in `config.yaml` only when the binary is installed.

## Scan results vs runtime probes

- **Runtime probe**: current container/host `PATH`
- **scanner_results table**: historical per-scan `binary_missing` events

Dashboard may show both: degraded coverage from probes and historical missing events from scans.

## Troubleshooting

| Symptom | Likely cause |
|---------|----------------|
| All scanners missing in UI | Wrong container image or empty `PATH` |
| One scanner missing after upgrade | Binary not in new image layer |
| Version shows `—` | Version command failed or timed out |
| Version shows `unknown` | Scanner ran but returned empty first line |
| Scans fail with `binary_missing` | Same as missing probe — install or disable scanner |

See [SCANNERS.md](SCANNERS.md), [OPERATOR_READINESS.md](OPERATOR_READINESS.md), [TROUBLESHOOTING.md](TROUBLESHOOTING.md).
