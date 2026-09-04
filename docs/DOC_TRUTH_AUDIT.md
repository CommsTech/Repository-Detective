# Documentation truth audit (RD-029)

**Date:** 2026-09-04 (updated Phase 2 closure + Phase 3)  
**Scope:** Claims vs reachable runtime behavior.  
**Method:** Source inspection + unit/integration tests in `golang:1.25-bookworm`. Full Gitea E2E remains RD-017.

Legend:

| Class | Meaning |
|-------|---------|
| STABLE | Documented, wired, and routinely used |
| BETA | Works for intended path; edge cases expected |
| EXPERIMENTAL | Code present; limited proof |
| DISABLED_BY_DEFAULT | Implemented but off until operator enables |
| PLANNED | Roadmap only |
| NOT_SUPPORTED | Must not be advertised as available |

Proof levels: `CODE_PRESENT` / `WIRED` / `UNIT_TESTED` / `INTEGRATION_TESTED` / `E2E_PROVEN`.

---

## Product posture

| Claim | Classification | Evidence |
|-------|----------------|----------|
| Public feedback via GitHub Issues | STABLE | Issues enabled; templates under `.github/ISSUE_TEMPLATE/` |
| Gitea = canonical forge (CI/wiki/dev) | STABLE | remotes + CONTRIBUTING / GITHUB_MIRROR |
| Security via private advisory / SECURITY.md | BETA | Private vulnerability reporting enabled |
| Recommended install = compose pull :8081 | STABLE | README, QUICKSTART, SETUP, docker-compose.yml |
| AI optional / LLM auditors off by default | STABLE | `enable_llm_auditors` default false |
| Local LLM (Ollama) + locality classification | BETA | `internal/privacy` UNIT_TESTED |
| No auto-merge remediation PRs | STABLE | Intentional non-feature |
| Policy outcomes ≠ “secure/safe” | STABLE | PR summary + policy docs |
| Issues canonical for findings | STABLE | Issue manager + fingerprint mapping |
| GitHub forge issue filing | EXPERIMENTAL | Code path exists |
| GitLab | NOT_SUPPORTED | Explicitly documented |

---

## Policy / scanners (Phase 2)

| Capability | Classification | Proof |
|------------|----------------|-------|
| POLICY_MET / ACTION_REQUIRED / EVALUATION_INCOMPLETE / OBSERVATION_ONLY | BETA | UNIT_TESTED |
| Observe / Warn / Enforce | BETA | WIRED + UNIT_TESTED |
| Compact PR summary (one comment) | BETA | WIRED |
| PR summary **idempotent upsert** (RD-006A) | BETA | UNIT_TESTED; E2E_PROVEN pending RD-017 |
| Required scanners cannot shrink when disabled (RD-012A) | BETA | UNIT_TESTED |
| `SKIPPED_BY_POLICY` incomplete for REQUIRED | BETA | UNIT_TESTED |
| Silent `0/0` → POLICY_MET | NOT_SUPPORTED | Blocked for Custom empty set |

Phase 2 classification: **IMPLEMENTED + UNIT/INTEGRATION TESTED; FULL GITEA E2E PENDING RD-017**.

---

## Privacy / security (Phase 3)

| Capability | Classification | Proof |
|------------|----------------|-------|
| Privacy modes local_only / hybrid / external_ai_enabled | BETA | CODE_PRESENT + WIRED + UNIT_TESTED |
| LOCAL_ONLY AI egress fail-closed | BETA | WIRED + UNIT_TESTED |
| LOCAL_ONLY notification EXTERNAL channel disable | BETA | WIRED + UNIT_TESTED (classifier) |
| SECURITY_MODEL.md threat model | STABLE doc | PROVEN/PARTIAL/NOT_* labels |
| MinimalSubprocessEnv for SBOM + clone | BETA | CODE_PRESENT + WIRED |
| Query API key reject (optional) | BETA | UNIT_TESTED; default still compat false |
| Local session auth recommended new install | BETA | Docs + UNIT/INTEGRATION; runtime default unchanged |
| Class-B ephemeral sandbox default | PLANNED | NOT_IMPLEMENTED |
| Per-scan seccomp/network ns | PLANNED | NOT_IMPLEMENTED |

---

## Remaining doc risks

1. Some historical dogfood notes still mention host-network-as-default — historical only.
2. Policy outcome persistence on scan UI detail pages still thin.
3. Do not claim E2E forge behavior until RD-017 closes.
4. Do not advertise sandboxing stronger than SECURITY_MODEL.md.

---

## Phase acceptance

| Phase | Status |
|-------|--------|
| Phase 1 RD-001–003, RD-029 | Pass |
| Phase 2 RD-004–006, RD-011–012 + **006A/012A closure** | Pass (unit/integration); E2E pending |
| Phase 3 RD-007–010 | Pass at stated proof levels (see above) |
