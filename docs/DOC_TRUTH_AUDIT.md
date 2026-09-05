# Documentation truth audit (RD-029)

**Date:** 2026-09-05 (Phase 6B closed)  
**Method:** Unit/integration tests + disposable Gitea E2E against the **published** `v0.1.0-beta.3` digest + clean-install against that digest.

Proof levels: `CODE_PRESENT` / `WIRED` / `UNIT_TESTED` / `INTEGRATION_TESTED` / `E2E_PROVEN` / `PARTIAL` / `NOT_PROVEN`.

## Explicit Gitea baseline

| Item | Value |
|------|-------|
| Tested Gitea | **1.22.3** only |
| Release | `v0.1.0-beta.3` |
| Image digest | `sha256:6a615548c8a1fc2494140e73f1c3bd3f78f0ed54a7b15eaa7a1025e83e308727` (Gitea = GHCR) |
| Sanitized evidence | `docs/release/ACCEPTANCE_v0.1.0-beta.3.md` |
| Raw local artifacts | `e2e/results/` (gitignored) |

## Root cause of prior DOC_TRUTH drift (RD-029A)

Canonical Gitea `docs/DOC_TRUTH_AUDIT.md` advanced in Phase 6A, while the public GitHub tree is a **sanitized snapshot** (`scripts/sync-gitea-to-github.sh --github-snapshot`). Drift was **snapshot lag** after canonical updates — not sanitization rewriting an older blob in place. Fix: update canonical DOC_TRUTH, then re-run snapshot and validate content trees (not commit SHA equality).

## Capability-by-capability (Phase 6B)

| Capability | Classification | Notes |
|------------|----------------|-------|
| Webhook registration | **E2E_PROVEN** | Gitea 1.22.3 |
| Real webhook delivery | **E2E_PROVEN** | Doctor `proof.webhook_delivery` |
| FIRST_SCAN_PROVEN | **E2E_PROVEN** | |
| Canonical issue lifecycle | **E2E_PROVEN** | Secret fixture + redaction |
| PR summary idempotency | **E2E_PROVEN** | |
| Required-scanner fail-closed | **E2E_PROVEN** | → `EVALUATION_INCOMPLETE` |
| LOCAL_ONLY + AI disabled | **E2E_PROVEN** | |
| Doctor (published image) | **E2E_PROVEN** | beta.3 |
| Restart/persistence | **E2E_PROVEN** | |
| Clean install from published digest | **E2E_PROVEN** | `PUBLISHED_IMAGE_CLEAN_INSTALL_E2E_PROVEN` |
| Core Gitea E2E from published digest | **E2E_PROVEN** | `PUBLISHED_IMAGE_CORE_E2E_PROVEN` |
| POLICY_MET (live forge) | **E2E_PROVEN** | Orphan clean tree; not a security claim |
| ACTION_REQUIRED (live forge) | **E2E_PROVEN** | Deterministic secret fixture |
| OBSERVATION_ONLY (live forge) | **E2E_PROVEN** | Observe mode |
| EVALUATION_INCOMPLETE (live forge) | **E2E_PROVEN** | Fail-closed + observed on standard PR |
| Secret auto-resolution after fix | **PARTIAL** | Intentional; [FINDING_RESOLUTION_SEMANTICS.md](FINDING_RESOLUTION_SEMANTICS.md) |
| SAST / dependency fixtures | **E2E_PROVEN** | |
| Class-B remediation sandbox | **NOT_PROVEN** | RD-008B Option C; disabled by default |
| Runner isolation | **NOT_PROVEN** | |
| Forgejo | **NOT_PROVEN** | |
| Upgrade E2E | **NOT_PROVEN** | beta.3 is baseline for next upgrade |
| Broader Gitea version range | **NOT_PROVEN** | Only 1.22.3 |
| GitHub issue-provider production | **NOT_PROVEN** / experimental | |

## Non-claims

Never equate POLICY_MET with “safe/secure”. Never claim Class-B sandboxing. Never advertise a Gitea version range from this single baseline. Never claim published-image proof from a locally overlaid rebuild alone.
