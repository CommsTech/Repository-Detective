# Security check matrix (10 minimum checks)

Maps the requested **10 minimum checks** (Gitea #39) to Repository Detective capabilities. This is an operator matrix — not a certification.

| # | Check | Shipped in RD | Tools / modules |
|---|--------|---------------|-----------------|
| 1 | Secrets and credential detection | Yes | gitleaks, static `SEC-HARDCODED-SECRET`, preinstall secret checks |
| 2 | Supply chain / SBOM | Partial | trivy, grype, govulncheck; SBOM via [SBOM.md](SBOM.md); Syft/Dependency-Track = roadmap |
| 3 | Structural security (SAST) | Yes | semgrep, static rules, gosec, health analyzers |
| 4 | Input validation / sanitization | Partial | semgrep + static injection rules; no standalone sanitizer |
| 5 | Cryptographic compliance | Partial | static + dependency scanners; no full crypto audit module |
| 6 | Error handling / resources | Partial | health checks (reliability, maintainability) |
| 7 | Access control logic | Partial | static heuristics; no full authz model checker |
| 8 | Audit logging verification | Partial | static `QUAL-DEBUG`; pipeline `GOV-PIPELINE-SECRET-ECHO` |
| 9 | Configuration security | Yes | checkov, hadolint, profile misconfig detection |
| 10 | Public-release / history hygiene | Partial | gitleaks history, `REL-INTERNAL-INFRA-REF`, [PRE_PUBLISH_CHECKS.md](PRE_PUBLISH_CHECKS.md) |

## Overall score

After each scan, `ComputeScoreResult` in `analyzers/scoring.go` produces a **0–100 repository health score** (stored normalized as `overall_score` 0–1 on scan summary JSON and shown in Gitea summary issues).

### Scoring formula

- Start at **100**
- Subtract per finding: critical **−30**, high **−15**, medium **−5**, low **−1**
- Cap graph/low-health noise at **−10** total so noisy low findings cannot drive the score to zero
- **Suppressed** and **report-only** findings do not affect the score
- When required scanners fail and no findings exist, score is **`incomplete`** (not `0.00%`)
- `score_explanation` on the scan summary documents the breakdown

Example: 27 mixed non-critical findings typically score well above zero unless many are critical/high without caps applying.

## Roadmap (not blocking closeout)

- OpenSCAP / formal compliance scanning (#41)
- Dedicated optimization performance suite (#43)
- Full pipeline Actions audit UI (#46)
