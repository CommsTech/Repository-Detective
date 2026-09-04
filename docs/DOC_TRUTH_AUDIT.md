# Documentation truth audit (RD-029)

**Date:** 2026-09-04 (Phase 6A RD-017A / RD-018)  
**Method:** Unit/integration tests + disposable Gitea E2E harness (`scripts/e2e-gitea-acceptance.sh`) + clean-install script (`scripts/e2e-clean-install.sh`).  

Proof levels: `CODE_PRESENT` / `WIRED` / `UNIT_TESTED` / `INTEGRATION_TESTED` / `E2E_PROVEN`.

## Explicit Gitea baseline

| Item | Value |
|------|-------|
| Tested Gitea | **1.22.3** only (do not advertise a range from this single baseline) |
| Harness | `docker-compose.e2e.yml` + `scripts/e2e-gitea-acceptance.sh` |
| Artifacts | `e2e/results/<run-id>/acceptance.json` (gitignored) |
| Canonical acceptance run | `e2e/results/20260904T182636Z-2505621/` (local) |

## Phase 6A capabilities (capability-by-capability)

| Capability | Classification | Notes |
|------------|----------------|-------|
| Webhook registration (API create + readback) | **E2E_PROVEN** | Scenario `webhook_registration` |
| Webhook delivery (HMAC validated + persisted) | **E2E_PROVEN** | `WEBHOOK_DELIVERY_E2E_PROVEN`; Doctor `proof.webhook_delivery` |
| FIRST_SCAN_PROVEN | **E2E_PROVEN** | Distinct from webhook delivery; `operator_evidence` / Doctor `proof.first_scan` |
| PR summary create/upsert idempotency | **E2E_PROVEN** | Exactly one `repository-detective-policy-summary`; user comments untouched |
| Canonical issue lifecycle (secret fixture) | **E2E_PROVEN** | Synthetic Slack bot token; redaction checked; fingerprint reopen |
| Secret resolve after fix | **PARTIAL_E2E** | Fingerprint retained; auto-close may require reconcile apply (documented) |
| SAST fixture lifecycle | **E2E_PROVEN** | gosec/weak-crypto fixture |
| Dependency fixture | **E2E_PROVEN** | Pinned requirements; CVE text not brittle-asserted (DB drift tradeoff) |
| Policy outcome on PR summary | **E2E_PROVEN** | Observed `EVALUATION_INCOMPLETE` under incomplete required coverage |
| Required-scanner fail-closed | **E2E_PROVEN** | Controlled gitleaks stub → `EVALUATION_INCOMPLETE`; no `POLICY_MET` |
| Optional scanner failure | **E2E_PROVEN** | hadolint stub visibility |
| Privacy LOCAL_ONLY + AI disabled | **E2E_PROVEN** | Deterministic-only acceptance default |
| Doctor JSON | **E2E_PROVEN** | `/api/v1/doctor` authorized; missing token → 401 |
| Restart persistence of proofs | **E2E_PROVEN** | Webhook/first-scan evidence survives restart |
| Webhook negatives | **E2E_PROVEN** | Bad/missing/malformed → 401; replay via fingerprints (not delivery-ID reject) |
| Clean install RD-018 | harness | `scripts/e2e-clean-install.sh` (published image + `.env.example`) |
| Upgrade E2E | **NOT_PROVEN** | No trustworthy prior public-beta baseline selected |
| Remediation planner | CODE_PRESENT / prior unit | Not expanded this phase |
| Class-B remediation execution | **NOT_PROVEN** | Intentionally excluded (RD-008B Option C) |
| Runner isolation | **NOT_PROVEN** | |
| Forgejo | **NOT_PROVEN** | |
| GitHub issue provider | **EXPERIMENTAL** | |

## Phases 1–4

Remain accepted at prior documented proof levels. Phase 2 forge E2E advanced by Phase 6A where scenarios PASS.

## Non-claims

Never equate POLICY_MET with “safe/secure”. Never claim Class-B sandboxing. Never advertise a Gitea version range from this single baseline.
