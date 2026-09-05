# Deterministic Scanners

Repository Detective runs **external security and lint tools** before (and instead of) LLM analysis where possible. Findings are reported as forge issues with severity, file, and line metadata.

See [RUBRICS.md](RUBRICS.md) for the security, pipeline governance, public-release, and advisory optimization scoring rubrics.

## Pipeline order

```
Changed files fetched from Gitea
  ├─ Static regex rules          (built-in, no binaries; see [FALSE_POSITIVES.md](FALSE_POSITIVES.md))
  ├─ Trivy fs scan               (CVEs, secrets, misconfig)
  ├─ Grype dir scan              (dependency CVEs)
  ├─ Gitleaks dir scan           (hardcoded secrets, filesystem snapshot)
  ├─ Semgrep scan                (SAST rulesets, filesystem snapshot)
  ├─ govulncheck                 (Go module vulnerabilities, optional)
  ├─ gosec                       (Go security rules, optional)
  ├─ staticcheck                 (Go static analysis, optional)
  ├─ hadolint                    (Dockerfile lint, optional)
  ├─ checkov                     (IaC/config policy, optional)
  ├─ Language linters            (golangci-lint, ruff, shellcheck)
  ├─ Health checks               (tech debt, reliability, maintainability, test gaps, performance; depth ≥ 2)
  └─ LLM auditors                (only on flagged files, if enabled)
```

High-confidence deterministic findings **skip LLM debate and PoC generation**. Health check sources (`tech_debt`, `reliability`, `maintainability`, `test_gap`, `performance`, `ai_generated_risk`) register as deterministic.

## Tools

| Tool | Purpose | Install |
|------|---------|---------|
| [Trivy](https://github.com/aquasecurity/trivy) | Dependency CVEs, secrets, Dockerfile/K8s misconfig | Included in **all-in-one** image |
| [Grype](https://github.com/anchore/grype) | Dependency vulnerability matching | Included in **all-in-one** image |
| [Gitleaks](https://github.com/gitleaks/gitleaks) | Hardcoded secrets (filesystem snapshot) | Included in **all-in-one** image (8.21.2+) |
| [Semgrep](https://github.com/semgrep/semgrep) | SAST (registry rulesets or operator rules) | Included in **all-in-one** image |
| [govulncheck](https://pkg.go.dev/golang.org/x/vuln/cmd/govulncheck) | Go module vulnerability scanning | `go install golang.org/x/vuln/cmd/govulncheck@latest` |
| [gosec](https://github.com/securego/gosec) | Go security anti-patterns | `go install github.com/securego/gosec/v2/cmd/gosec@latest` |
| [staticcheck](https://staticcheck.dev/) | Go static analysis (first-class scanner) | `go install honnef.co/go/tools/cmd/staticcheck@latest` |
| [hadolint](https://github.com/hadolint/hadolint) | Dockerfile best-practice lint | Install from distro package manager or release binary |
| [checkov](https://www.checkov.io/) | IaC policy (Terraform, K8s, Helm, workflows) | `python3 -m pip install --user checkov` |
| golangci-lint | Go static analysis (includes many linters) | Included in **all-in-one** image |
| ruff | Python lint | Included in **all-in-one** image |
| shellcheck | Shell script analysis | Included in **all-in-one** image |
| OpenSCAP | Self-hosted runner/server hardening evidence | Run outside Repository Detective on the runner host |

If a binary is missing (e.g. **core** image without tools), Repository Detective logs a warning and continues with other scanners.

## Configuration

`config/config.yaml` or environment variables:

```yaml
enable_trivy: true
enable_grype: true
enable_gitleaks: false
gitleaks_config: ""              # optional path to operator-controlled gitleaks.toml
gitleaks_timeout_seconds: 0       # 0 = use scanner_timeout_seconds
enable_semgrep: false
semgrep_config: p/ci              # registry ruleset or operator path (p/security-audit, /etc/repository-detective/rules)
semgrep_timeout_seconds: 0        # 0 = use scanner_timeout_seconds
semgrep_max_findings: 100
semgrep_severity_threshold: INFO    # minimum Semgrep severity: INFO, WARNING, ERROR
enable_govulncheck: false           # Go module vulnerabilities (requires go.mod or Go files, depth ≥ 2)
enable_gosec: false                # Go security rules (requires .go files, depth ≥ 2)
enable_staticcheck: false          # Go static analysis first-class scanner (depth ≥ 2)
govulncheck_timeout_seconds: 0      # 0 = use scanner_timeout_seconds
gosec_timeout_seconds: 0
staticcheck_timeout_seconds: 0
go_scanner_max_findings: 100        # cap per Go scanner
enable_hadolint: false              # Dockerfile lint (requires Dockerfile paths, depth ≥ 2)
enable_checkov: false               # IaC/config policy scan (depth ≥ 2)
hadolint_timeout_seconds: 0
checkov_timeout_seconds: 0
iac_scanner_max_findings: 100       # cap per IaC scanner
enable_linters: true
enable_llm_auditors: false   # deterministic-only default; set true only with AI configured
scanner_timeout_seconds: 120
```

Environment equivalents:

```bash
REPOSITORY_DETECTIVE_ENABLE_TRIVY=true
REPOSITORY_DETECTIVE_ENABLE_GRYPE=true
REPOSITORY_DETECTIVE_ENABLE_GITLEAKS=false
REPOSITORY_DETECTIVE_GITLEAKS_CONFIG=
REPOSITORY_DETECTIVE_GITLEAKS_TIMEOUT_SECONDS=0
REPOSITORY_DETECTIVE_ENABLE_SEMGREP=false
REPOSITORY_DETECTIVE_SEMGREP_CONFIG=p/ci
REPOSITORY_DETECTIVE_SEMGREP_TIMEOUT_SECONDS=0
REPOSITORY_DETECTIVE_SEMGREP_MAX_FINDINGS=100
REPOSITORY_DETECTIVE_SEMGREP_SEVERITY_THRESHOLD=INFO
REPOSITORY_DETECTIVE_ENABLE_GOVULNCHECK=false
REPOSITORY_DETECTIVE_ENABLE_GOSEC=false
REPOSITORY_DETECTIVE_ENABLE_STATICCHECK=false
REPOSITORY_DETECTIVE_GOVULNCHECK_TIMEOUT_SECONDS=0
REPOSITORY_DETECTIVE_GOSEC_TIMEOUT_SECONDS=0
REPOSITORY_DETECTIVE_STATICCHECK_TIMEOUT_SECONDS=0
REPOSITORY_DETECTIVE_GO_SCANNER_MAX_FINDINGS=100
REPOSITORY_DETECTIVE_ENABLE_LINTERS=true
REPOSITORY_DETECTIVE_ENABLE_LLM_AUDITORS=true
REPOSITORY_DETECTIVE_SCANNER_TIMEOUT_SECONDS=120
```

### Gitleaks

Gitleaks runs in **`dir` mode** (filesystem snapshot only — no git history scan). Command shape:

```bash
gitleaks dir <workspace> --report-format json --report-path=- --no-banner --redact
```

When `gitleaks_config` is set, Repository Detective passes `--config <path>` (operator-controlled). If unset, gitleaks may still load `(workspace)/.gitleaks.toml` per [gitleaks config precedence](https://github.com/gitleaks/gitleaks#configuration) — there is no safe flag to disable repo config in this phase.

Findings use category `secret`, severity `high`, and skip LLM debate when `enable_llm_auditors` is false.

### Semgrep

Semgrep runs against the prepared workspace directory (filesystem snapshot only — no git history, no dependency install, no autofix). Command shape:

```bash
semgrep scan --json --quiet --metrics=off --config <semgrep_config> <workspace>
```

**Config precedence:** Repository-Detective always passes an operator-controlled `--config` value (default `p/ci`). Repo-local Semgrep rule files are **not** loaded unless the operator config points at them. Semgrep does not auto-load repo rules when `--config` is explicit. Avoid `--config auto` with `--metrics=off` (Semgrep may refuse auto-config without metrics).

**Severity threshold:** `semgrep_severity_threshold` is the minimum Semgrep severity included in results (`INFO` < `WARNING` < `ERROR`). Default `INFO` includes all findings.

**Max findings:** When results exceed `semgrep_max_findings` (default 100), Repository-Detective keeps the first N and sets scanner detail to `truncated to N findings (M total)`.

**Severity mapping:** `ERROR` → `high`, `WARNING` → `medium`, `INFO` → `low`.

Example deterministic scan config:

```yaml
workspace_mode: auto
analysis_depth: 2
enable_llm_auditors: false

enable_trivy: true
enable_grype: true
enable_gitleaks: true
enable_semgrep: true
enable_linters: true

semgrep_config: p/ci
```

For security-focused scans with less CI noise, try `semgrep_config: p/security-audit` first.

### Go scanners (Phase 13A)

Native Go analysis runs when enabled, `analysis_depth >= 2`, and the workspace contains `go.mod` or `.go` files. **Disabled by default.** No `go get`, `go mod tidy`, or other dependency-changing commands.

| Scanner | Command | Purpose |
|---------|---------|---------|
| govulncheck | `govulncheck -json ./...` | Known Go module vulnerabilities (OSV IDs) |
| gosec | `gosec -fmt=json ./...` | Go security anti-patterns (G-rules) |
| staticcheck | `staticcheck -f json ./...` | Go static analysis (SA/S/ST/QF checks) |

Install on core or runner hosts:

```bash
go install golang.org/x/vuln/cmd/govulncheck@latest
go install github.com/securego/gosec/v2/cmd/gosec@latest
go install honnef.co/go/tools/cmd/staticcheck@latest
```

Per-repo overrides (nullable in DB): `enable_govulncheck`, `enable_gosec`, `enable_staticcheck`, timeouts, `go_scanner_max_findings`.

**Noise:** staticcheck can overlap golangci-lint Staticcheck rules — fingerprint dedup applies. Prefer staticcheck for deterministic Go repos; use linters for multi-language repos.

Recommended deterministic Go repo policy:

```yaml
analysis_depth: 2
ai_policy: disabled
enable_govulncheck: true
enable_gosec: true
enable_staticcheck: true
enable_semgrep: true
enable_gitleaks: true
enable_health_checks: true
```

### IaC / container scanners (Phase 13B)

**hadolint** lints Dockerfiles (`Dockerfile`, `*.Dockerfile`, paths containing `/Dockerfile`). No Docker builds are executed.

**checkov** scans the workspace directory for Terraform (`.tf`), Kubernetes/Helm YAML, Docker Compose, GitHub/Gitea workflow YAML, CloudFormation, and related config. Command:

```bash
checkov -d . -o json --quiet --skip-download
```

`--skip-download` avoids fetching external Terraform modules. Repository Detective does not run `pip install` or install checkov for you.

Both scanners are **disabled by default**, deterministic, and capped by `iac_scanner_max_findings`.

Recommended deterministic infrastructure policy:

```yaml
analysis_depth: 2
ai_policy: disabled
enable_hadolint: true
enable_checkov: true
enable_trivy: true
enable_semgrep: true
enable_gitleaks: true
enable_health_checks: true
```

### Deterministic-only mode

To disable all LLM scanning (no AI token usage for audits):

```bash
REPOSITORY_DETECTIVE_ENABLE_LLM_AUDITORS=false
```

Static rules, Trivy, Grype, Gitleaks (when enabled), Semgrep (when enabled), linters, and health checks still run and create issues.

## Repository health checks (Phase 10)

Built-in deterministic checks (no external binaries) analyze maintainability, reliability, tech debt, test coverage gaps, performance footguns, and optional cautious AI-code-risk signals.

- Runs when `enable_health_checks: true` and `analysis_depth >= 2`
- Does not install dependencies, execute repo code, or run tests
- Findings use categories: `tech_debt`, `reliability`, `maintainability`, `code_quality`, `test_gap`, `performance`, `ai_generated_risk`
- `enable_ai_risk_checks` defaults to `false` — enable only when you accept higher false-positive rate

Full reference: [HEALTH_CHECKS.md](HEALTH_CHECKS.md)

```yaml
enable_health_checks: true
enable_tech_debt_checks: true
enable_reliability_checks: true
enable_maintainability_checks: true
enable_test_gap_checks: true
enable_performance_checks: true
enable_ai_risk_checks: false
health_max_findings: 100
health_large_file_lines: 1000
health_large_function_lines: 150
health_max_nesting_depth: 5
health_max_function_params: 7
```

## Dependency manifests

On push/PR scans, Repository-Detective automatically fetches common manifest files from the repo (even if unchanged) so Trivy and Grype can detect CVEs:

- `go.mod`, `package.json`, `requirements.txt`, `Cargo.toml`, `Dockerfile`, etc.

See `scanners/workspace.go` for the full list.

## Docker image

The official Repository-Detective Dockerfile installs Trivy, Grype, golangci-lint, ruff, and shellcheck. Use that image for full scanner coverage.

Bare-metal / custom installs: install the binaries above and ensure they are on `PATH` for the Repository-Detective process.

## Not integrated

- **[web-check](https://github.com/lissy93/web-check)** — analyzes live websites (DNS, headers, SSL), not repository source. Use separately for deployed app checks.
- **Black / Prettier / ESLint** — ruff covers Python lint; ESLint requires a project `eslint` config and Node.js. Open an issue if you need first-class ESLint/Prettier support.

## Issue labels

Scanner findings use categories such as `dependency_vulnerability`, `secret`, `sast`, `hardcoded_secret`, `misconfiguration`, and `lint` in issue bodies.
Built-in static rules may also emit advisory `optimization`, `pipeline_governance`, and `public_release` findings when source text contains detectable patterns.
