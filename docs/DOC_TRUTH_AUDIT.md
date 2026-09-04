# Documentation truth audit (RD-029)

**Date:** 2026-09-04 (Phase 6B RD-018 / RD-017B–D / RD-029A)  
**Method:** Unit/integration tests + disposable Gitea E2E harness (`scripts/e2e-gitea-acceptance.sh`) against the **published** image digest + clean-install (`scripts/e2e-clean-install.sh`).  

Proof levels: `CODE_PRESENT` / `WIRED` / `UNIT_TESTED` / `INTEGRATION_TESTED` / `E2E_PROVEN` / `PARTIAL` / `NOT_PROVEN`.

## Explicit Gitea baseline

| Item | Value |
|------|-------|
| Tested Gitea | **1.22.3** only (do not advertise a range from this single baseline) |
| Harness | `docker-compose.e2e.yml` + `scripts/e2e-gitea-acceptance.sh` |
| Artifacts | `e2e/results/<run-id>/acceptance.json` (gitignored raw); sanitized summary in `docs/release/` |
| Release pin | `v0.1.0-beta.3` (see release evidence for digests) |

## Root cause of prior DOC_TRUTH drift (RD-029A)

Phase 6A updated the **canonical Gitea** `docs/DOC_TRUTH_AUDIT.md`, but the public GitHub tree is a **sanitized snapshot** (`scripts/sync-gitea-to-github.sh --github-snapshot`), not a shared commit identity. Drift occurred when:

1. Canonical file advanced on Gitea after the last successful snapshot publish, **or**
2. Snapshot/sanitization ran from a tree that omitted the Phase 6A DOC_TRUTH update.

Fix: keep DOC_TRUTH on the canonical mainline, then re-run the snapshot workflow and validate **content** (tree equivalence after sanitization), not commit SHA equality. See Phase 6B completion report / `docs/release/ACCEPTANCE_v0.1.0-beta.3.md`.

## Capability-by-capability (Phase 6B)

| Capability | Classification | Notes |
|------------|----------------|-------|
| Webhook registration | **E2E_PROVEN** | Gitea 1.22.3 |
| Real webhook delivery | **E2E_PROVEN** | HMAC + Doctor `proof.webhook_delivery` |
| FIRST_SCAN_PROVEN | **E2E_PROVEN** | Distinct from webhook delivery |
| Canonical issue lifecycle | **E2E_PROVEN** | Secret fixture + redaction |
| PR summary idempotency | **E2E_PROVEN** | Upsert; user comments untouched |
| Required-scanner fail-closed | **E2E_PROVEN** | → `EVALUATION_INCOMPLETE` |
| LOCAL_ONLY + AI disabled | **E2E_PROVEN** | Deterministic acceptance default |
| Doctor (published image) | **E2E_PROVEN** | Included in beta.3 |
| Restart/persistence | **E2E_PROVEN** | Proofs survive restart |
| Clean install from published digest | **E2E_PROVEN** | `PUBLISHED_IMAGE_CLEAN_INSTALL_E2E_PROVEN` |
| Core Gitea E2E from published digest | **E2E_PROVEN** | `PUBLISHED_IMAGE_CORE_E2E_PROVEN` |
| POLICY_MET (live forge) | **E2E_PROVEN** | Controlled clean fixture; not a security claim |
| ACTION_REQUIRED (live forge) | **E2E_PROVEN** | Deterministic secret fixture |
| OBSERVATION_ONLY (live forge) | **E2E_PROVEN** | Observe mode + finding |
| EVALUATION_INCOMPLETE (live forge) | **E2E_PROVEN** | Fail-closed required scanner |
| Secret auto-resolution after fix | **PARTIAL** | Intentional; see [FINDING_RESOLUTION_SEMANTICS.md](FINDING_RESOLUTION_SEMANTICS.md) |
| SAST fixture lifecycle | **E2E_PROVEN** | gosec/weak-crypto |
| Dependency fixture | **E2E_PROVEN** | Pinned requirements |
| Class-B remediation sandbox | **NOT_PROVEN** | RD-008B Option C; disabled by default |
| Runner isolation | **NOT_PROVEN** | |
| Forgejo | **NOT_PROVEN** | |
| Upgrade E2E | **NOT_PROVEN** | beta.3 becomes baseline for next upgrade |
| Broader Gitea version range | **NOT_PROVEN** | Only 1.22.3 tested |
| GitHub issue-provider production | **NOT_PROVEN** / experimental | |

## Non-claims

Never equate POLICY_MET with “safe/secure”. Never claim Class-B sandboxing. Never advertise a Gitea version range from this single baseline. Never claim published-image proof from a locally overlaid rebuild alone.
