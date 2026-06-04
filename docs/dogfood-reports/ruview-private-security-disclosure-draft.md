# Private security disclosure draft — RuView

> **Do not submit automatically.** Send only through the project’s preferred security channel after human review. **No raw secrets, tokens, or exploit steps** are included.

**To:** RuView maintainers (security contact — confirm before send)  
**Repository:** https://github.com/ruvnet/RuView  
**Commit reviewed:** `872d7593bbeeed63524386aa60e6805bb4e1b26c`  
**Assessment tool:** Repository Detective pre-install audit (standard depth)

---

## Summary

Repository Detective detected several **potential security and install-hardening concerns** during a third-party pre-install review. We recommend maintainer review before production deployment. Findings below are **scanner-reported patterns and advisory IDs**; impact depends on how components are deployed.

Secret scanning via gitleaks was **inconclusive** (parser failure). Pattern-based checks (Semgrep/Trivy) still flagged items that may relate to credentials, CI trust boundaries, and dependency exposure.

---

## Findings for private review

### 1. Docker build secret handling (critical class)

| Field | Value |
|-------|--------|
| Scanner | trivy |
| Rule | DS031 |
| Location | `docker/Dockerfile.rust` |
| Description | Build configuration may pass secrets via build-args, environment variables, or copied secret files |

**Why review:** Images built with embedded secrets can leak credentials to registries or runtime environments.

**Suggested remediation:** Use runtime secret injection (secrets manager, mounted secrets); avoid copying secret files into build context; scan images after build.

---

### 2. Gradio / Python dependency advisories (critical / high class)

| Advisory IDs (sample) | Location |
|----------------------|----------|
| CVE-2025-23042 | `aether-arena/space/requirements.txt` |
| CVE-2024-8966, CVE-2026-28414, CVE-2026-28416 | same |

**Why review:** Dependency scanner linked multiple advisories to the Gradio-related requirements set. Confirm versions in use and upgrade paths per upstream guidance.

**Suggested remediation:** Pin and upgrade affected packages; rerun dependency scan; validate file-upload and web UI surfaces in a test environment.

---

### 3. GitHub Actions workflow patterns (high class)

| Rule (Semgrep) | Workflow file |
|----------------|---------------|
| `github-script-injection` | `.github/workflows/security-scan.yml` |
| `run-shell-injection` | `.github/workflows/cd.yml`, `.github/workflows/desktop-release.yml` |

**Why review:** Workflows may execute untrusted context if inputs are attacker-influenced. This is a **pattern match**, not proof of exploitability.

**Suggested remediation:** Restrict `permissions:`; avoid executing untrusted PR/event data in scripts; use pinned actions and explicit input sanitization.

---

### 4. Hardcoded credential pattern (high class)

| Field | Value |
|-------|--------|
| Scanner | semgrep |
| Rule | `python.jwt.security.jwt-hardcode.jwt-python-hardcoded-secret` |
| Location | `archive/v1/test_auth_rate_limit.py` |

**Why review:** Static analysis matched a JWT hardcoding pattern. **No secret value is included in this draft.**

**Suggested remediation:** Confirm whether test/archive code ships to users; move secrets to environment or secret store; rotate if any material was ever committed.

---

### 5. Container and cluster configuration (high class)

| Rule | Location | Topic |
|------|----------|--------|
| DS002 | `docker/Dockerfile.python`, `docker/Dockerfile.rust` | Non-root user recommended |
| KSV118, KSV014 | `logging/fluentd-config.yml` | Security context / read-only root FS |

**Suggested remediation:** Run containers as non-root where feasible; tighten pod security context for deployed configs.

---

### 6. Additional dependency advisories (high class — Rust / Node)

| Location | Scanner notes |
|----------|----------------|
| `v2/Cargo.lock` | WASM3, quinn-proto, rustls-webpki advisory classes |
| `ui/mobile/package-lock.json` | axios-related advisories |

Review in context of whether these artifacts ship to production binaries or mobile releases.

---

## Secret scanning caveat

**gitleaks:** `parse_failed` during automated audit — **secret scan inconclusive.**  
**Semgrep/Trivy:** reported possible secret-handling and credential-like patterns only.  
**This draft:** contains **no raw secret values.**

**Recommendation:** Maintainers run gitleaks (or equivalent) locally on commit `872d7593` and address any confirmed leaks separately.

---

## Safe reproduction (maintainers)

1. Check out commit `872d7593bbeeed63524386aa60e6805bb4e1b26c` in an isolated environment.  
2. Run `trivy`, `semgrep`, and `hadolint` with project-appropriate configs.  
3. Run **gitleaks** locally to complete secret coverage.  
4. Do not publish exploit proof-of-concepts in public channels.

---

## Validation after fixes

- Re-run dependency and container scanners.  
- Confirm workflow permission reductions in CI.  
- Confirm Docker images no longer embed build-time secrets.  
- Re-run gitleaks before release.

---

## Repository Detective

Generated by Repository Detective — Inspect. Analyze. Improve.

_Review by a human before sending. Repository Detective does not submit this draft upstream._
