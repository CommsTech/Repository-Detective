# Phase 1 implementation report — Public-beta contradictions

**Date:** 2026-09-04  
**Program:** Repository Detective Product Hardening backlog

---

## RD-001 — Public feedback path

| Field | Value |
|-------|-------|
| Status | **implemented** + **wired** (GitHub Issues verified enabled; private vuln reporting enabled) |
| Behavior | Public bugs/features → GitHub Issues templates; security → SECURITY.md / private advisories; Gitea = canonical forge |
| Files | `SECURITY.md`, `CONTRIBUTING.md`, `README.md`, `docs/PUBLIC_BETA.md`, `.github/ISSUE_TEMPLATE/*`, `.gitea/ISSUE_TEMPLATE/config.yml`, installation/scanner templates |
| Tests | Manual: `gh repo view` → `hasIssuesEnabled: true`; private vulnerability reporting PUT succeeded |
| Docs | Yes |
| Limitations | No dedicated security@ email; UI health “Report issue” still opens Gitea maintainer templates |
| Follow-up | RD-019 README identity polish; optional sync of remaining Gitea-only templates |

## RD-002 — Canonical installation

| Field | Value |
|-------|-------|
| Status | **implemented** (docs + examples) |
| Behavior | **Recommended Installation** = `docker compose pull` + `:8081`; advanced options demoted; bridge networking documented correctly |
| Files | `README.md`, `QUICK_SETUP.md`, `docs/SETUP.md`, `docs/QUICKSTART.md`, `DEPLOYMENT.md`, `docs/DEPLOYMENT.md`, `docs/DOCKER.md`, `docs/NETWORKING.md`, `docs/DEPLOYMENT_ISSUES.md`, `.env.example`, `config.env.template` |
| Tests | Doc consistency review (no live pull in this pass) |
| Docs | Yes |
| Limitations | Live clean-install proof → RD-018 |
| Follow-up | RD-013 onboarding wizard stages; RD-014 doctor |

## RD-003 — AI optional everywhere

| Field | Value |
|-------|-------|
| Status | **implemented** + **runtime tested** (unit) |
| Behavior | Missing AI is Disabled, not broken; startup still only requires AI when LLM auditors enabled |
| Files | `store/settings.go`, `operator/status.go`, `main_status.go`, `api/ai_handler.go`, UI templates, onboarding HTML/JS, docs |
| Tests | `go test ./api/ ./store/` OK; `TestNeedsAIProvider`, `TestBuildReadinessAIAnalysisDisabledWithoutClient` OK (Docker golang:1.25) |
| Docs | Yes |
| Limitations | Deep profile still enables AI when selected (by design); locality enforcement → RD-007 |
| Follow-up | RD-007 privacy modes |

## RD-029 — Documentation truth audit

| Field | Value |
|-------|-------|
| Status | **implemented** (audit artifact) |
| Behavior | Capability classification matrix published |
| Files | `docs/DOC_TRUTH_AUDIT.md`, `docs/README.md` index |
| Tests | N/A (audit) |
| Limitations | Partial — focused on Phase 1 claims; full pass continues as later tasks land |

---

## Commits

| Hash | Task |
|------|------|
| `00953db` | RD-001 |
| `0a768c0` | RD-002 |
| `3f970d5` | RD-003 |
| tip of `main` after RD-029 commit | RD-029 (`git log -1 --oneline -- docs/DOC_TRUTH_AUDIT.md`) |

|------|
| `00953db` | RD-001 |
| `0a768c0` | RD-002 |
| `3f970d5` | RD-003 |
|  | RD-029 |

## Remaining Phase 1 gaps

- E2E clean install not re-proven in this session (RD-017/018).
- Formatting/lint of Go via local `golangci-lint` not run (no local Go toolchain; Docker tests only).
