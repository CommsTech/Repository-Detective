# Private beta release notes

Version: private beta (commit `dedd0c6` baseline + packaging sprint)  
Product: Repository Detective — Inspect. Analyze. Improve.

## Key features

- Multi-scanner repository analysis (Trivy, Grype, Gitleaks, Semgrep, Go toolchain, IaC, linters)
- Deterministic scan profiles including `beta_standard` for low-noise private beta
- Dashboard with findings, severity/confidence filters, and code graph / risk map
- Evidence-based issue closure when fixes are verified in subsequent scans
- Continuous learning engine: structural dedup, scanner health, repo-scoped calibration recommendations
- Report-only dry-run mode for safe validation without forge side effects
- Pre-install audit for third-party repo trust assessment
- Executive reports with print/PDF export via browser
- API + webhook integration for Gitea and GitHub

## Safety defaults (private beta)

| Control | Default |
|---------|---------|
| Issue filing | Off |
| Remediation PRs | Off |
| LLM sanity gate | Off |
| LLM auditors | Off |
| Runner delegation | Off |
| Notifications | Off |
| Evidence closure | On |
| Backlog control | On |
| Global calibration auto-accept | Blocked |
| First-scan workflow | Report-only dry-run |

Shipped config: `config/private-beta.example.yaml` and `docker-compose.beta.yml`.

## Known limitations

- SBOM in bundle optional unless `cyclonedx-gomod` installed at build time
- Some scanners timeout on very large repos in default container timeouts
- Learning events on dry-runs require image built from learning-engine commits
- Python/Ruff homelab gating validated on operator repos; extend before broad Python beta
- No bundled macOS/Windows native binary — use Docker
- All-repo scan intentionally not supported in private beta
- CI green status may lag local verification (staticcheck fix in `572635b`)

## Unsupported workflows (private beta)

- All-repo bulk scan
- Non-product issue filing without operator approval
- Auto-remediation PRs
- Mandatory LLM review on every finding
- Global auto-calibration accept
- Public anonymous dashboard access

## How to report bugs

1. Reproduce with report-only scan if possible
2. Fill [PRIVATE_BETA_FEEDBACK_TEMPLATE.md](PRIVATE_BETA_FEEDBACK_TEMPLATE.md)
3. Attach redacted logs (no tokens)
4. Include scan ID, repo slug, and commit SHA of Repository Detective

## How to report false positives

- Finding ID or fingerprint from UI
- File path and line
- Why you believe it is a false positive
- Optional: suggest calibration rule (repo-scoped only in beta)

## How to report scanner failures

- Scanner name from `/health` or scan detail
- Timeout vs missing binary vs parse error
- Approximate repo size and language

## What data not to share

- `.env` files
- Gitea/GitHub tokens
- `data/repository-detective.db` (may contain repo metadata)
- Unredacted API keys in logs
- Customer proprietary source code (describe issue abstractly)

## Privacy and security notes

- Repository Detective stores findings locally in SQLite by default
- Forge tokens stay in environment — never commit to git
- Webhook secrets validate inbound push authenticity
- Beta package is distributed out-of-band; verify `checksums.txt` before run
- Issue filing creates visible forge artifacts — keep disabled until approved

## Upgrade from candidate verification build

If running pre-packaging commit:

1. Backup database
2. Replace binary or rebuild Docker image
3. Merge `config/private-beta.example.yaml` safety keys into your live config
4. Confirm `auto_create_issues: false` after restart

## References

- [PRIVATE_BETA_TESTER_GUIDE.md](PRIVATE_BETA_TESTER_GUIDE.md)
- [PRIVATE_BETA_OPERATOR_RUNBOOK.md](PRIVATE_BETA_OPERATOR_RUNBOOK.md)
- [../dogfood-reports/private-beta-report-only-validation.md](../dogfood-reports/private-beta-report-only-validation.md)
