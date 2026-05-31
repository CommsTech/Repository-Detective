# Deterministic Scanners

Bugbot runs **external security and lint tools** before (and instead of) LLM analysis where possible. Findings are reported as Gitea issues with the same severity, file, and line metadata as AI findings.

## Pipeline order

```
Changed files fetched from Gitea
  ├─ Static regex rules          (built-in, no binaries)
  ├─ Trivy fs scan               (CVEs, secrets, misconfig)
  ├─ Grype dir scan              (dependency CVEs)
  ├─ Language linters            (golangci-lint, ruff, shellcheck)
  └─ LLM auditors                (only on flagged files, if enabled)
```

High-confidence deterministic findings **skip LLM debate and PoC generation**.

## Tools

| Tool | Purpose | Install |
|------|---------|---------|
| [Trivy](https://github.com/aquasecurity/trivy) | Dependency CVEs, secrets, Dockerfile/K8s misconfig | Included in Bugbot Docker image |
| [Grype](https://github.com/anchore/grype) | Dependency vulnerability matching | Included in Bugbot Docker image |
| golangci-lint | Go static analysis | Included in Bugbot Docker image |
| ruff | Python lint | Included in Bugbot Docker image |
| shellcheck | Shell script analysis | Included in Bugbot Docker image |

If a binary is missing, Bugbot logs a warning and continues with the other scanners.

## Configuration

`config/config.yaml` or environment variables:

```yaml
enable_trivy: true
enable_grype: true
enable_linters: true
enable_llm_auditors: true    # set false for deterministic-only mode
scanner_timeout_seconds: 120
```

Environment equivalents:

```bash
BUGBOT_ENABLE_TRIVY=true
BUGBOT_ENABLE_GRYPE=true
BUGBOT_ENABLE_LINTERS=true
BUGBOT_ENABLE_LLM_AUDITORS=true
BUGBOT_SCANNER_TIMEOUT_SECONDS=120
```

### Deterministic-only mode

To disable all LLM scanning (no AI token usage for audits):

```bash
BUGBOT_ENABLE_LLM_AUDITORS=false
```

Static rules, Trivy, Grype, and linters still run and create issues.

## Dependency manifests

On push/PR scans, Bugbot automatically fetches common manifest files from the repo (even if unchanged) so Trivy and Grype can detect CVEs:

- `go.mod`, `package.json`, `requirements.txt`, `Cargo.toml`, `Dockerfile`, etc.

See `scanners/workspace.go` for the full list.

## Docker image

The official Bugbot Dockerfile installs Trivy, Grype, golangci-lint, ruff, and shellcheck. Use that image for full scanner coverage.

Bare-metal / custom installs: install the binaries above and ensure they are on `PATH` for the Bugbot process.

## Not integrated

- **[web-check](https://github.com/lissy93/web-check)** — analyzes live websites (DNS, headers, SSL), not repository source. Use separately for deployed app checks.
- **Black / Prettier / ESLint** — ruff covers Python lint; ESLint requires a project `eslint` config and Node.js. Open an issue if you need first-class ESLint/Prettier support.

## Issue labels

Scanner findings use categories such as `dependency_vulnerability`, `hardcoded_secret`, `misconfiguration`, and `lint` in issue bodies.
