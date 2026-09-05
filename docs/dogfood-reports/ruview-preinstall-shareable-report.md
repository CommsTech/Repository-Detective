# RuView pre-install assessment (shareable draft)

**Repository:** [ruvnet/RuView](https://github.com/ruvnet/RuView)  
**Commit reviewed:** `872d7593bbeeed63524386aa60e6805bb4e1b26c` (`main`)  
**Assessment depth:** standard  
**Date:** 2026-06-04 (UTC)

> **Private review draft.** Repository Detective produced this report for operator review before any external communication. Nothing in this document was submitted upstream automatically.

---

## Executive summary

Repository Detective found several **high-risk install concerns** that should be reviewed before production use. Scanners reported dependency advisories, container configuration items, CI workflow permission patterns, and possible secret-handling risks. **No raw secret values** appear in this report.

| Metric | Value |
|--------|--------|
| Install risk score | 100 / 100 (automated composite) |
| Recommendation | **Do not install until the listed critical/high findings are reviewed or mitigated** |
| Findings reviewed (stored) | 198 |

This score reflects breadth of signals (including maintainability and graph noise), not a claim that every finding is exploitable in your environment.

---

## Scanner coverage

| Scanner | Status | Notes |
|---------|--------|--------|
| trivy | found (16) | Dependency and config advisories |
| semgrep | found (8) | Static patterns including CI and credential-like code |
| hadolint | found (5) | Dockerfile hygiene |
| gitleaks | **found (10)** | Redacted scan completed (`bd8a34c0`); **maintainer triage required** — see `ruview-gitleaks-triage.md` |
| checkov | parse_failed | IaC rules not fully applied this run |
| grype | timed_out | Secondary SBOM scan incomplete |

### Secret scanning (gitleaks completed — triage required)

**Gitleaks completed successfully** on audit **`bd8a34c0`** (image `repository-detective:all-in-one` @ `0b5005a2a2b3`). **Ten** redacted credential-like patterns were reported. **No raw secret values** are included in this report.

Human triage (`ruview-gitleaks-triage.md`):

- **8** likely documentation/example false positives  
- **2** need maintainer review (CI workflow curl-auth pattern; tracked Vite source map)  
- **0** confirmed live credential exposure from this review alone  

Maintainers should run **gitleaks locally** with `--redact` to validate before any disclosure.

---

## Critical install blockers

Review these before any production install or image publish.

| Scanner | Rule | Location | Finding (neutral) |
|---------|------|----------|-------------------|
| trivy | CVE-2025-23042 | `aether-arena/space/requirements.txt` | Dependency advisory detected for Gradio-related stack (blocked path ACL class issue per advisory ID) |
| trivy | DS031 | `docker/Dockerfile.rust` | Build configuration appears to pass or copy secret material via build-args, environment, or build context |

**Suggested maintainer actions:** Upgrade/pin Gradio and related Python dependencies per upstream advisories; refactor Docker builds so secrets are not embedded in images or build-args.

---

## High-priority security review

Items detected by scanners that warrant maintainer review (private disclosure may be appropriate for some).

| Scanner | Rule | Location | Topic |
|---------|------|----------|--------|
| trivy | CVE-2024-8966, CVE-2026-28414, CVE-2026-28416 | `aether-arena/space/requirements.txt` | Additional Gradio-related dependency advisories |
| trivy | CVE-2022-39974, CVE-2026-31812, GHSA-82j2-j2ch-gfr8 | `v2/Cargo.lock` | Rust ecosystem advisories (WASM/QUIC/TLS components) |
| trivy | CVE-2026-44492, CVE-2026-44494 | `ui/mobile/package-lock.json` | axios-related dependency advisories |
| semgrep | `jwt-hardcode` | `archive/v1/test_auth_rate_limit.py` | Pattern consistent with hardcoded JWT material (**value not stored**) |
| semgrep | `github-script-injection` | `.github/workflows/security-scan.yml` | GitHub Actions pattern flagged for untrusted script context |
| semgrep | `run-shell-injection` | `.github/workflows/cd.yml`, `desktop-release.yml` | Workflow pattern flagged for shell injection class risk |

**Note:** CVE titles in scanner output can read dramatically; treat them as **review triggers**, not confirmed impact in your deployment model.

---

## Supply-chain / install hygiene

| Severity | Source | Finding |
|----------|--------|---------|
| medium | preinstall | CI workflows appear to use elevated or broad `permissions` (multiple files under `.github/workflows/`) |
| medium | preinstall | Install script pattern flagged for manual review |
| low | preinstall | Some Python manifests appear to lack lockfiles |

**Suggested maintainer actions:** Narrow workflow permissions to least privilege; document supported install paths; add lockfiles where reproducibility matters.

---

## Container / IaC hardening

| Severity | Scanner | Location | Finding |
|----------|---------|----------|---------|
| high | trivy | `docker/Dockerfile.python`, `docker/Dockerfile.rust` | Container image configured to run as root (DS002) |
| high | trivy | `logging/fluentd-config.yml` | Kubernetes-style config: default security context / writable root filesystem flags |
| medium | hadolint | Multiple Dockerfiles | Unpinned `apt` / `pip` install patterns (DL3008, DL3013) |

Checkov did not contribute findings this run (parser issue). Trivy and hadolint informed this section.

---

## Inconclusive / partial checks

| Check | Status | Guidance |
|-------|--------|----------|
| gitleaks | **found (10)** | Redacted patterns triaged — see `ruview-gitleaks-triage.md` |
| checkov | parse_failed | IaC rules not fully applied this run |
| grype | timed_out | Optional second opinion SBOM scan |

---

## Repository structure / maintainability

Repository map (standard depth): **4,727 nodes**, **9,137 edges**. Graph analysis reported **59** “disconnected file” style observations—often large repos with auxiliary trees (`v2/crates/*`, tooling, or agent metadata paths). These are **structure/maintainability signals**, not standalone install blockers.

Tech-debt and maintainability markers accounted for much of the low/medium volume; they support prioritization but should not be read as security defects without context.

---

## Suggested maintainer actions (checklist)

1. **Upgrade Gradio** and related dependencies in `aether-arena/space/requirements.txt`; verify with dependency scanner after bump.
2. **Review Docker build secret handling** in `docker/Dockerfile.rust` (DS031 class finding).
3. **Review container root usage** and Kubernetes security contexts for deployed manifests.
4. **Review GitHub Actions** permissions and untrusted input in workflow scripts (see private disclosure draft for detail).
5. **Add lockfiles** where Python installs are user-facing.
6. **Review gitleaks triage** (`ruview-gitleaks-triage.md`) — 10 redacted patterns; mostly docs/examples; confirm workflow #2 locally.
7. **Validate** Semgrep/Trivy/hadolint findings in an isolated environment before release.

---

## Related documents (this package)

| Document | Audience |
|----------|----------|
| `ruview-private-security-disclosure-draft.md` | Maintainer security contact (manual send) |
| `ruview-public-issue-drafts.md` | Optional public issues — non-sensitive items only |
| `ruview-gitleaks-triage.md` | Human triage of 10 redacted gitleaks findings |
| `ruview-preinstall-audit.md` | Internal operator notes (audit metadata) |

---

## External-share readiness

| Question | Answer |
|----------|--------|
| Ready for **private security disclosure** (manual send)? | **Yes** — use `ruview-private-security-disclosure-draft.md`; lead with dependency/Docker/CI items; gitleaks as redacted patterns requiring maintainer confirmation |
| Ready for **public hygiene issues**? | **Yes** — drafts 1–5 in `ruview-public-issue-drafts.md` only (no secrets/CVE exploit detail/gitleaks hits) |
| Ready for **public filing without human review**? | **No** |
| `do_not_install` recommendation | **Unchanged** until critical/high items reviewed or mitigated |

### Remaining caveats

- Gitleaks: **10 redacted patterns** — not confirmed secrets; workflow item #2 needs maintainer eyes-on.  
- checkov / grype: partial coverage.  
- CVE and CI-injection items: private channel preferred over public issues.  
- Qdrant not used; AI off during audit.

---

## Repository Detective

Generated by Repository Detective — Inspect. Analyze. Improve.

Pre-install assessment and evidence-based remediation guidance.  
Project: https://github.com/ruvnet/RuView (subject repository only; operator product URL not embedded).

_This report was generated by Repository Detective and should be reviewed by a human before submission or sharing._
