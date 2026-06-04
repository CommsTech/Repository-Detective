# RuView pre-install audit (third-party)

**Date:** 2026-06-04 (UTC)  
**Target:** [ruvnet/RuView](https://github.com/ruvnet/RuView) @ `872d7593bbeeed63524386aa60e6805bb4e1b26c` (`main`)  
**Depth:** `standard`  
**Audit ID:** `dae05e0c-4c24-441e-9c05-c8ce5db4cbe0` (operator-local; not a secret)

> **Sanitized dogfood report.** Manual review required before any external sharing. No upstream issues filed, no email, no auto-submit. Qdrant and LLM auditors were off during this run.

---

## Safety controls (verified)

| Control | Setting |
|---------|---------|
| Pre-install API only | Yes — third-party audit path |
| `qdrant_enabled` | **false** |
| `enable_llm_auditors` / AI risk | **off** |
| Upstream issue creation | **none** |
| Email / auto-submit | **none** |
| Raw secrets in this report | **none** (titles/rules/CVE IDs only) |

---

## Install risk recommendation

| Field | Value |
|-------|--------|
| **Risk score** | 100 / 100 |
| **Recommendation** | **`do_not_install`** |
| **Duration** | ~4m 30s |
| **Findings stored (cap)** | 198 (summary reports 200 analyzed) |

**Summary:** Multiple **critical** dependency and container misconfiguration issues, numerous **high** CVEs (Gradio, Rust/npm lockfiles, Kubernetes config), elevated CI workflow permissions, and graph signals of disconnected subsystems. Do not install or deploy until dependencies, Dockerfiles, CI permissions, and disclosed security patterns are reviewed in isolation.

---

## Scanner status

| Scanner | Status | Findings |
|---------|--------|----------|
| trivy | found | 16 |
| semgrep | found | 8 |
| hadolint | found | 5 |
| grype | timed_out | 0 |
| gitleaks | parse_failed | 0 |
| checkov | parse_failed | 0 |
| govulncheck | clean | 0 |
| gosec | clean | 0 |
| staticcheck | clean | 0 |
| ruff | binary_missing | 0 |
| shellcheck | binary_missing | 0 |

**Notes:** Grype timeout and gitleaks/checkov parse failures reduce secret/IaC coverage; treat absence of gitleaks hits as **inconclusive**, not clean.

---

## Critical / high findings (representative)

### Critical

| Source | Rule | Location | Title (sanitized) |
|--------|------|----------|-------------------|
| trivy | CVE-2025-23042 | `aether-arena/space/requirements.txt` | Gradio blocked path ACL bypass |
| trivy | DS031 | `docker/Dockerfile.rust` | Secrets via build-args / env / copied secret files |

### High (sample)

| Source | Rule | Location | Title (sanitized) |
|--------|------|----------|-------------------|
| trivy | CVE-2024-8966, CVE-2026-28414, CVE-2026-28416 | `aether-arena/space/requirements.txt` | Gradio DoS / path traversal / SSRF |
| trivy | CVE-2022-39974, CVE-2026-31812, GHSA-82j2-j2ch-gfr8 | `v2/Cargo.lock` | Rust/WASM / QUIC / rustls-webpki issues |
| trivy | CVE-2026-44494, CVE-2026-44492 | `ui/mobile/package-lock.json` | axios MITM / proxy-bypass issues |
| trivy | DS002 | `docker/Dockerfile.python`, `docker/Dockerfile.rust` | Container runs as root |
| trivy | KSV118, KSV014 | `logging/fluentd-config.yml` | Default security context / writable root FS |
| semgrep | `jwt-hardcode` | `archive/v1/test_auth_rate_limit.py` | Hardcoded JWT secret pattern (value not stored) |
| semgrep | `github-script-injection` | `.github/workflows/security-scan.yml` | GitHub Actions script injection risk |
| semgrep | `run-shell-injection` | `.github/workflows/cd.yml`, `desktop-release.yml` | Workflow shell injection risk |

---

## Supply-chain concerns

| Severity | Rule | Summary |
|----------|------|---------|
| medium | `preinstall.workflow_permissions` | Multiple workflows use elevated or risky permissions (9 files under `.github/workflows/`) |
| medium | `preinstall.install_script` | Risky install script present |
| low | `preinstall.missing_lockfile` | Missing lockfile for some Python manifests |

**Safer install guidance:** Avoid curl/bash one-liners; pin dependencies; verify checksums; run installs only in an isolated VM; review workflow `permissions:` blocks before trusting CI.

---

## Docker / IaC concerns

| Severity | Source | Issue |
|----------|--------|-------|
| critical | trivy (DS031) | Secret material in Docker build context (`docker/Dockerfile.rust`) |
| high | trivy (DS002) | Images run as root (`docker/Dockerfile.python`, `docker/Dockerfile.rust`) |
| high | trivy (KSV*) | Kubernetes/fluentd config: weak security context |
| medium | hadolint (DL3008, DL3013) | Unpinned apt/pip installs in Dockerfiles |
| medium | preinstall | Workflow permission risks on Docker-related CI workflows |

Checkov did not contribute (parse failure); rely on trivy + hadolint for this pass.

---

## Secrets check (redacted)

| Check | Result |
|-------|--------|
| gitleaks | **parse_failed** — no automated secret list |
| semgrep secret patterns | **8** security-rule hits; evidence stored as rule titles only |
| Risk summary flag | `secret_finding: true` in audit metadata (caution flag) |
| Raw tokens in Qdrant / report | **none committed** |

**Representative patterns (no values reproduced):**

- JWT hardcoded-secret rule in test/archive Python
- GitHub Actions injection rules in workflow YAML
- Trivy DS031 “secrets in Docker build” (build-time secret handling)

**Operator action:** Re-run gitleaks/checkov in a fixed environment before treating secrets as absent.

---

## Dependency risks

Primary clusters:

1. **Python / Gradio** (`aether-arena/space/requirements.txt`) — multiple high/critical CVEs on Gradio-related advisories.
2. **Rust workspace** (`v2/Cargo.lock`) — WASM3, quinn-proto, rustls-webpki advisories.
3. **Node mobile UI** (`ui/mobile/package-lock.json`) — axios CVEs.
4. **Lockfile hygiene** — preinstall noted missing Python lockfiles in places.

---

## Repository Map observations

| Metric | Value |
|--------|--------|
| Nodes | 4,727 |
| Edges | 9,137 |
| Graph-derived findings | 59 (mostly `Possible disconnected file: …`) |

**Interpretation:** Large monorepo-style layout (`v2/crates/*`, agents/skills trees). Many **disconnected-file** graph warnings suggest modules or paths with weak linkage to the main graph — review before assuming cohesive build/deploy boundaries. No graph data stored in this markdown (metadata only).

---

## Generated disclosure drafts (API)

Nine drafts stored for audit `dae05e0c-4c24-441e-9c05-c8ce5db4cbe0`:

| Type | Count | Use |
|------|-------|-----|
| `install_risk_summary` | 1 | Internal install decision |
| `general_bug` | 8 | **Public** issue drafts (high-confidence, non-secret) |

Retrieve via API: `GET /api/v1/preinstall/audits/{audit_id}/reports` and `GET /api/v1/preinstall/reports/{id}`.

---

## Manual disclosure draft (private — review before sending)

> **Private security disclosure draft** — do not paste raw secrets publicly.

**Subject:** Docker build secret handling + Gradio CVE cluster in RuView  
**Repository:** https://github.com/ruvnet/RuView  
**Commit:** `872d7593bbeeed63524386aa60e6805bb4e1b26c`

**Summary:** Pre-install audit flagged **critical** misconfiguration (Docker build secrets / build-args) and **critical/high** dependency issues in Gradio-related Python requirements, plus GitHub Actions injection class findings in CI workflows.

**Impact:** Potential credential leakage in images, exploitable web UI dependencies, and CI workflow injection if inputs are attacker-controlled.

**Evidence:** Scanner rule IDs and paths only (see tables above). Reproduce from a local checkout at the audited commit; do not publish exploit steps.

**Suggested remediation:** Remove secrets from build contexts; pin and upgrade Gradio stack; harden workflow permissions and untrusted `github.script` usage; add lockfiles; re-scan with gitleaks/checkov when parsers succeed.

**Validation:** Re-run trivy/semgrep/hadolint in isolation after fixes.

---

## Public issue draft (non-sensitive example)

Suitable for upstream **public** issue after human edit (from `general_bug` draft):

```markdown
**Title:** Gradio Blocked Path ACL Bypass (CVE-2025-23042) in aether-arena requirements

**Repository:** https://github.com/ruvnet/RuView
**Commit:** `872d7593bbeeed63524386aa60e6805bb4e1b26c`

## Affected location

- **File:** `aether-arena/space/requirements.txt`
- **Rule:** CVE-2025-23042
- **Scanner:** trivy

## Summary

Dependency scan reported a critical Gradio ACL bypass advisory on the pinned requirements set.

## Suggested fix

Upgrade Gradio (and related deps) to patched versions per upstream advisory; verify with `trivy` or equivalent.

## Validation

Re-run dependency scanner after bump; exercise Gradio file-upload paths in a test environment.

---
Generated by Repository Detective — Inspect. Analyze. Improve.
_Review by a human before submission._
```

---

## Repository Detective footer

Reports generated in-product include:

```text
Generated by Repository Detective — Inspect. Analyze. Improve.
Gitea-first repository assessment, pre-install audit, and evidence-based remediation.
Project: <operator-configured URL>
```

This markdown omits operator-specific project URLs. Configure `preinstall_report_include_project_link` and `repository_detective_project_url` locally if needed.

---

## Follow-up

| Item | Status |
|------|--------|
| Qdrant for RuView | **Not used** (disabled; embedding/UUID fixes pending) |
| External sharing | **Blocked** until tone/format review |
| Re-audit | Optional after upstream fixes or scanner parser fixes |

**Risk breakdown (from audit summary):** critical 70, high 360, medium 190, low 173, scanner_failure 25 (pre-cap counts in `risk_explanation`; stored findings capped at 200).
