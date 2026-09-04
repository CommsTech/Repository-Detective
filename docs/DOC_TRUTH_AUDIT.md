# Documentation truth audit (RD-029)

**Date:** 2026-09-04  
**Scope:** Phase 1 public-beta blockers — claims vs reachable runtime behavior.  
**Method:** Source inspection + doc cross-check (not full E2E proof unless noted).

Legend:

| Class | Meaning |
|-------|---------|
| STABLE | Documented, wired, and routinely used |
| BETA | Works for intended path; edge cases expected |
| EXPERIMENTAL | Code present; limited proof |
| DISABLED_BY_DEFAULT | Implemented but off until operator enables |
| PLANNED | Roadmap only |
| NOT_SUPPORTED | Must not be advertised as available |

---

## Product posture (Phase 1 decisions)

| Claim | Classification | Evidence |
|-------|----------------|----------|
| Public feedback via GitHub Issues | STABLE | Issues enabled; templates under `.github/ISSUE_TEMPLATE/` |
| Gitea = canonical forge (CI/wiki/dev) | STABLE | remotes + CONTRIBUTING / GITHUB_MIRROR |
| Security via private advisory / SECURITY.md | BETA | Private vulnerability reporting enabled; no dedicated security@ mailbox documented |
| Recommended install = compose pull :8081 | STABLE | README, QUICKSTART, SETUP, docker-compose.yml |
| AI optional / LLM auditors off by default | STABLE | `viper.SetDefault("enable_llm_auditors", false)`; `needsAIProvider()` gate |
| Local LLM (Ollama) supported | BETA | Provider factory + docs; locality enforcement → RD-007 |
| No auto-merge remediation PRs | STABLE | Remediation PR off by default; no auto-merge path advertised |
| Policy outcomes ≠ “secure/safe” | BETA | Messaging updated in README; full policy model → RD-004/005 |
| Issues canonical for findings | STABLE | Issue manager + fingerprint mapping |
| GitHub forge issue filing | EXPERIMENTAL | Code path exists; RC-unproven |
| GitLab | NOT_SUPPORTED | Explicitly documented |

---

## Installation & configuration

| Claim | Classification | Notes |
|-------|----------------|-------|
| Port 8081 recommended | STABLE | Default compose |
| Port 8080 minimal compose | STABLE | Advanced path |
| Bridge networking default | STABLE | Host-network is overlay only |
| Forge token required for forge features | STABLE | `main.go` validation |
| AI vars required | **NOT_SUPPORTED** (false claim removed) | Was in QUICK_SETUP/SETUP/DEPLOYMENT — fixed |
| External DB | BETA | Documented; SQLite is default |
| Runner topology | BETA | Optional |

---

## Scanners & AI

| Capability | Classification |
|------------|----------------|
| Trivy / Grype / Gitleaks / Semgrep / Go tools / Hadolint / Checkov / linters | BETA (image-dependent) |
| Scanner coverage classification (REQUIRED/OPTIONAL states) | PLANNED (RD-011) |
| LLM auditors | DISABLED_BY_DEFAULT |
| AI recommendations | DISABLED_BY_DEFAULT |
| Deep profile AI | BETA when AI configured |

---

## Policy / remediation / privacy (later phases)

| Capability | Classification |
|------------|----------------|
| POLICY_MET / ACTION_REQUIRED / EVALUATION_INCOMPLETE | PLANNED (RD-004) |
| Observe / Warn / Enforce modes | PLANNED (RD-005) |
| Compact PR summaries (no per-finding comments) | PLANNED (RD-006/026) |
| LOCAL_ONLY privacy mode | PLANNED (RD-007) |
| Isolation threat model doc | PLANNED (RD-008) |
| `repository-detective doctor` | PLANNED (RD-014) |
| Gitea E2E acceptance suite | PLANNED (RD-017) |
| Forgejo proven support | EXPERIMENTAL / unproven (RD-027) |
| Enterprise multi-tenant / SSO / HA | PLANNED / deferred (RD-028) |

---

## Remaining doc risks (follow-ups)

1. Some historical dogfood / beta notes still mention host-network-as-default — treat as historical evidence, not operator docs.
2. `docs/guides/INSTALL_STEP_BY_STEP.md` and some wiki stubs remain thin — advanced only.
3. UI “Report issue” links still prefer Gitea templates for operator health reports (intentional for maintainers); public README/CONTRIBUTING point to GitHub.
4. Full capability matrix refresh after Phase 2–3 semantics land.

---

## Phase 1 acceptance (this audit)

| Task | Doc/code consistency |
|------|----------------------|
| RD-001 feedback path | Pass — single public destination documented; security separated |
| RD-002 recommended install | Pass — one labeled path; advanced demoted |
| RD-003 AI optional | Pass — docs/UI/defaults aligned with runtime gate |
