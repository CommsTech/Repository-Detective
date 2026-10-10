# Finding resolution semantics (RD-017D)

This document records the **designed** lifecycle for finding closure. It prefers avoiding false resolution over automatic convenience.

## Short answers (Phase 6B)

| Question | Answer |
|----------|--------|
| Does a finding auto-close when absent from a later webhook scan? | **No.** Absence from a later scan alone does not close the finding. |
| Who owns verified forge/issue closure? | **Issue reconciliation Apply** (`POST /api/v1/repos/:id/reconcile-issues`) when `issue_reconciliation_close_verified: true`, and/or **evidence-based closure** after merge+rescan (see [EVIDENCE_CLOSURE.md](EVIDENCE_CLOSURE.md)). |
| Is reconcile invoked automatically on push/PR webhooks? | **No.** Webhooks run detection + issue create/update; reconcile remains an explicit operator (or scheduled) action. |
| Can absence from a PR/diff/changed-files scan prove resolution? | **No.** Partial scope cannot prove the finding is gone from the repository. |
| What prevents incorrect close when a later scan did not cover the file? | Closure paths require scanner success + fingerprint absence under **sufficient scope** (full-repo / post-merge evidence), not “not seen in this partial scan.” |

## Why partial scans must not auto-close

Webhook and PR scans often analyze **changed files only**. For those scopes:

```text
“No finding in this scan”  ≠  “Finding no longer exists in the repository.”
```

Naive absence→close would incorrectly resolve secrets or SAST findings when the affected path was outside the scan window.

## Intended lifecycle (current product)

States remain the existing finding/issue statuses (no new enum required for Phase 6B):

| State | Meaning |
|-------|---------|
| Open / still present | Fingerprint observed; forge issue may exist |
| Retained after fix push | Fingerprint identity kept until reconcile/evidence path confirms |
| Reconcile: `already_fixed_verify` | Absent from **latest qualifying** scan; verify before close |
| `resolved_verified` / forge closed | Only after evidence/reconcile rules succeed |

Operator-facing detail: [ISSUE_RECONCILIATION.md](ISSUE_RECONCILIATION.md), [EVIDENCE_CLOSURE.md](EVIDENCE_CLOSURE.md).

## Proof level (Gitea 1.22.3)

| Capability | Level | Notes |
|------------|-------|-------|
| Secret detect → issue → fingerprint stable across fix/reintroduce | **E2E_PROVEN** | Phase 6A/6B harness |
| Auto-close immediately after fix push without reconcile | **PARTIAL** (intentional) | Retained until full-scope reconcile/evidence; not a defect by itself |
| Naive absence-based close on PR/diff scans | **Not implemented** (by design) | Do not add |

## Future (out of Phase 6B)

Optional auto-reconcile **only** after a full-repository scan with required scanners successful — still never on changed-files-only webhook scope.
