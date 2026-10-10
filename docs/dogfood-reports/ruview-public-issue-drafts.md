# RuView — public issue drafts (non-sensitive only)

> **Manual review required.** Do not auto-submit. Use only for findings that are safe to discuss in a public tracker. **Do not open public issues** for secret patterns, JWT-hardcode signals, detailed CVE exploit narratives, or CI injection findings—use `ruview-private-security-disclosure-draft.md` instead.

**Repository:** https://github.com/ruvnet/RuView  
**Commit:** `872d7593bbeeed63524386aa60e6805bb4e1b26c`

---

## Classification summary

| Category | Public issue? | Notes |
|----------|---------------|--------|
| Gradio / CVE dependency cluster | **No** | Use private channel or maintainer dependency bump PR |
| Docker DS031 (build secrets) | **No** | Private security disclosure |
| JWT-hardcode pattern (semgrep) | **No** | Private security disclosure |
| gitleaks matches (any) | **No** | Private security disclosure only — redacted pattern scan |
| GitHub Actions injection patterns | **No** | Private security disclosure |
| axios / Rust CVE titles with attack detail | **No** | Private or dependency bump PR |
| Container runs as root (DS002) | **Optional** | Draft below — hardening, neutral tone |
| Unpinned apt/pip in Dockerfiles | **Yes** | Supply-chain hygiene |
| Missing Python lockfile | **Yes** | Reproducibility |
| Fluentd/K8s security context defaults | **Optional** | Draft below — config hardening |
| Graph disconnected-file observations | **Optional** | Maintainability, not security |

**In-product `general_bug` drafts** in the audit DB covered many CVE titles; those are **reclassified here as private-only** for public GitHub use.

---

## Draft 1 — Dockerfile: pin package versions (public)

**Title:** Hadolint: consider pinning apt/pip versions in Dockerfiles

**Labels:** `docker`, `supply-chain`, `good first issue` (optional)

**Body:**

Repository Detective (pre-install scan) detected hadolint rules **DL3008** / **DL3013** in Docker build files—for example `docker/Dockerfile.rust` and `v2/crates/nvsim-server/Dockerfile`—suggesting unpinned `apt-get` / `pip install` usage.

**Impact:** Unpinned installs can reduce reproducibility and make supply-chain review harder.

**Suggested change:** Pin package versions where practical, or document why floating versions are intentional.

**Validation:** Re-run hadolint on Dockerfiles after changes.

---

## Draft 2 — Reproducible Python dependencies (public)

**Title:** Consider lockfiles for Python dependency manifests

**Labels:** `dependencies`, `documentation`

**Body:**

A pre-install hygiene check noted Python projects without lockfiles in some paths. Adding lockfiles (or documenting a supported install method with hashes) may help reproducible installs.

**Suggested change:** Add lockfiles or document the supported install workflow in README/contributing docs.

---

## Draft 3 — Run containers as non-root user (public, hardening)

**Title:** Docker: consider non-root USER in Dockerfiles

**Labels:** `docker`, `security-hardening`

**Body:**

Trivy rule **DS002** was reported for `docker/Dockerfile.python` and `docker/Dockerfile.rust`, indicating images may run as root by default.

**Impact:** Running as non-root is a common container hardening step; actual risk depends on your deployment model.

**Suggested change:** Add a non-root `USER` where compatible with runtime requirements.

**Validation:** Build and smoke-test images after the change.

---

## Draft 4 — Kubernetes logging config defaults (public, hardening)

**Title:** Review securityContext defaults in `logging/fluentd-config.yml`

**Labels:** `kubernetes`, `configuration`

**Body:**

Configuration scan (trivy KSV118 / KSV014 class) suggested reviewing default security context and read-only root filesystem settings in `logging/fluentd-config.yml`.

**Suggested change:** Align with your cluster pod security standards; document exceptions if defaults are intentional.

---

## Draft 5 — Secret scanning in CI (public, generic hygiene)

**Title:** Consider adding secret scanning to CI (optional)

**Labels:** `ci`, `security-hardening`, `documentation`

**Body:**

For supply-chain hygiene, some projects run secret scanning (for example gitleaks or GitHub secret scanning) in CI to catch accidental credential commits early.

**Note:** This is a generic suggestion only — not a report of confirmed secrets in this repository.

**Suggested change:** Evaluate whether gitleaks or platform-native secret scanning fits your workflow; document the chosen approach in contributing/security docs.

---

## Not for public issues (reference only)

Maintain **private** communication for:

- `CVE-2025-23042` and related Gradio advisories in `aether-arena/space/requirements.txt`
- Docker **DS031** build secret handling
- Semgrep **jwt-hardcode** in `archive/v1/test_auth_rate_limit.py`
- Semgrep **github-script-injection** / **run-shell-injection** workflows
- **All gitleaks findings** (see `ruview-gitleaks-triage.md`) — private disclosure only
- High-severity axios / Rust lockfile advisories where titles imply exploit scenarios

See `ruview-private-security-disclosure-draft.md`.

---

## Repository Detective

Generated by Repository Detective — Inspect. Analyze. Improve.

_Review by a human before creating GitHub issues. Repository Detective does not open issues upstream._
