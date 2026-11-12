# Pre-public / release checks (Gitea #45)

Operator checklist before publishing a repository or cutting a release.

## Automated (Repository Detective)

| Check | How |
|-------|-----|
| Secrets in history | Enable gitleaks; run full-repo scan |
| Hardcoded credentials | static `SEC-HARDCODED-SECRET`, trivy/grype secrets |
| Internal IPs / hostnames | static `REL-INTERNAL-INFRA-REF` (advisory) |
| Pipeline secret echo | static `GOV-PIPELINE-SECRET-ECHO` on workflow files |
| Dependency CVEs | trivy, grype, govulncheck |

Configure `scan_profile` with security scanners enabled. Use [DOGFOODING.md](DOGFOODING.md) on this repo first.

## Manual (required)

| Check | Owner |
|-------|--------|
| PHI / PII / IP review | Security + legal |
| Internal codenames in comments/commits | Code review |
| Test data realism | Remove production-like samples |
| Third-party license review | Legal |

Scanners detect patterns; they do not classify regulated data.

## Pre-install audit

For third-party repos before install: [PREINSTALL_AUDIT.md](PREINSTALL_AUDIT.md) (`preinstall_audit_enabled`).
