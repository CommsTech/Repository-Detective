# Documentation truth audit (RD-029)

**Date:** 2026-09-05 (Phase 8A closed)  
**Method:** Unit/integration tests + disposable Gitea E2E against the **published** `v0.1.0-beta.3` digest + clean-install against that digest + beta.3→current upgrade harness (RD-033).

Proof levels: `CODE_PRESENT` / `WIRED` / `UNIT_TESTED` / `INTEGRATION_TESTED` / `E2E_PROVEN` / `PARTIAL` / `NOT_PROVEN`.

## Explicit Gitea baseline

| Item | Value |
|------|-------|
| Tested Gitea | **1.22.3** only |
| Immutable public-beta baseline | `v0.1.0-beta.3` |
| Image digest | `sha256:6a615548c8a1fc2494140e73f1c3bd3f78f0ed54a7b15eaa7a1025e83e308727` (Gitea = GHCR) |
| Sanitized evidence | `docs/release/ACCEPTANCE_v0.1.0-beta.3.md` |
| Raw local artifacts | `e2e/results/` (gitignored) |
| GitHub release history | Begins at **beta.3** (no retroactive beta.2 tag manufacture) |

## Root cause of prior DOC_TRUTH drift (RD-029A)

Canonical Gitea `docs/DOC_TRUTH_AUDIT.md` advanced in Phase 6A, while the public GitHub tree is a **sanitized snapshot** (`scripts/sync-gitea-to-github.sh --github-snapshot`). Drift was **snapshot lag** after canonical updates — not sanitization rewriting an older blob in place. Fix: update canonical DOC_TRUTH, then re-run snapshot and validate content trees (not commit SHA equality).

## Capability-by-capability (Phase 6B + 8A)

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
| Forgejo | **NOT_PROVEN** | Estimate only in [TECH_DEBT_AUDIT.md](TECH_DEBT_AUDIT.md); start with 15 LTS |
| Upgrade beta.3 → current main | **UPGRADE_FROM_BETA3_TO_CURRENT_MAIN_INTEGRATION_PROVEN** | `scripts/e2e-upgrade-from-beta3.sh` — not published-release upgrade until beta.4 digest |
| Published release→release upgrade | **NOT_PROVEN** | Requires exact beta.3 → beta.4 digests |
| Broader Gitea version range | **NOT_PROVEN** | Only 1.22.3 |
| GitHub issue-provider production | **NOT_PROVEN** / experimental | |
| New-install local auth recommendation | **UNIT_TESTED** + **WIRED** | Runtime default remains `api_key_only` for upgrades (RD-032) |
| Diagnostic redaction | **UNIT_TESTED** + **PARTIAL** | Central sanitizer + corpus; heuristic remainder in KNOWN_LIMITATIONS |
| Finding-quality metrics | **UNIT_TESTED** + **WIRED** | Local-only; no telemetry (RD-024) |
| Calibration transparency | **WIRED** + **UNIT_TESTED** (store) | History + accepted revert (RD-025) |

## Non-claims

Never equate POLICY_MET with “safe/secure”. Never claim Class-B sandboxing. Never advertise a Gitea version range from this single baseline. Never claim published-image proof from a locally overlaid rebuild alone. Never claim Forgejo support without E2E. Never claim perfect log redaction.
