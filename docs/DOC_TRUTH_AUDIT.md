# Documentation truth audit (RD-029)

**Date:** 2026-09-04 (Phase 4 RD-013/RD-014 + RD-008B)  
**Method:** Source inspection + unit tests in `golang:1.25-bookworm`. Full Gitea E2E remains RD-017.

Proof levels: `CODE_PRESENT` / `WIRED` / `UNIT_TESTED` / `INTEGRATION_TESTED` / `E2E_PROVEN`.

## Phase 4

| Capability | Classification | Proof |
|------------|----------------|-------|
| Onboarding Connect→Select→Protect→Verify→Ready | BETA | WIRED (wizard + APIs); UNIT_TESTED (doctor engine / recommend) |
| Permission matrix (Gitea pull/push/admin proxy) | BETA | WIRED; not full scope enumeration |
| Privacy egress disclosure in Protect | BETA | WIRED + privacy UNIT_TESTED |
| Verify uses shared doctor package | BETA | WIRED |
| READY / READY_WITH_LIMITATIONS / NOT_READY | BETA | UNIT_TESTED (mapping) |
| FIRST_SCAN_PROVEN | PLANNED/partial | Documented; not yet persisted as store flag |
| `repository-detective doctor` CLI | BETA | WIRED |
| `GET /api/v1/doctor` (+ bundle) | BETA | WIRED |
| `/ui/doctor` | BETA | WIRED |
| Support bundle redaction | BETA | UNIT_TESTED |
| Webhook delivery E2E | NOT_PROVEN | Registration vs delivery distinguished |
| RD-008B Class-B decision | STABLE doc | Option C; control-plane allowlisted validation WIRED; sandbox NOT_PROVEN |

## Earlier phases (unchanged)

Phase 1–3 remain accepted at documented proof levels. Phase 2: **IMPLEMENTED + UNIT/INTEGRATION TESTED; FULL GITEA E2E PENDING RD-017**.

## Non-claims

- Do not claim SAFE/SECURE/SECURITY PASSED
- Do not claim Class-B sandboxing
- Do not claim webhook E2E without forge-originated delivery
