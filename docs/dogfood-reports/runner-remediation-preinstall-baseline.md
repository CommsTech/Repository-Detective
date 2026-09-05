# Runner, remediation, and pre-install baseline

Recorded: 2026-06-02

## Latest commit (baseline)

`fef00ea` — docs(wiki): update gitea wiki publish diagnostics

Sprint start branch: `main`, clean except in-progress sprint work.

## Pre-install failed audit UX (observed bug)

| Field | Observed (before fix) |
|-------|------------------------|
| Status | `failed` |
| Risk score | **0 / 100** (misleading — looks safe) |
| Recommendation | `unknown` or empty |

**Expected after fix:** `Risk score: unavailable`, `Recommendation: audit failed`, failure stage, sanitized error, next action, sandbox metadata when present, explicit “No conclusion was made because audit did not complete.”

Successful audit reference: `061578b7-2df7-4d13-b79f-69e0265928f7` (Hello-World, completed, 0 issues/PRs).

## Runner delegation status

| Setting | Value |
|---------|-------|
| `runner_delegation_enabled` | `false` (default) |
| `runner_mode` | `core` / `native` when enabled |
| `runner_shared_secret` | must be set via env/secrets only |
| Native worker API | `/api/v1/runner/ping`, `/jobs/claim`, `/jobs/:id/result` |
| HMAC headers | `X-Runner-Timestamp`, `X-Runner-Nonce`, `X-Runner-Signature` |

**Architecture note:** Gitea `act_runner` (Actions workflows) is separate from Repository Detective native runner delegation. Native runners handle scans/SBOM/graph/pre-install; Gitea Actions is optional for repo-native test verification.

## Remediation PR status

| Setting | Default |
|---------|---------|
| `remediation_pr_enabled` | `false` |
| `remediation_pr_require_approval` | `true` |
| `remediation_pr_require_tests` | `true` |
| `remediation_pr_use_runner_verification` | `true` |
| Pre-install audits | never create PRs |

## Gitea Actions runner status

| Setting | Default |
|---------|---------|
| `gitea_actions_test_backend_enabled` | `false` |
| Workflow template | `.gitea/workflows/repository-detective-verify.yml` |

Gitea act_runner registration tokens are **sensitive**. If a token was shared in chat, **rotate/reset** it in Gitea before any public docs or screenshots. Never commit tokens.

## Secrets handling

- No `.env`, runner registration tokens, or local binaries staged in git.
- `runner_shared_secret` and Gitea tokens via env/secrets only.
- Failure messages redact token patterns before display.

## Open blockers (baseline)

- Gitea wiki push HTTP 500 (server-side).
- Product repo issue #48 (operator task) open; active-present findings need triage.
- Runner delegation disabled until operator enables with shared secret.
