# Evidence-Based Closure (Repository Detective)

Repository Detective — **Inspect. Analyze. Improve.**

Findings are **not** closed because a remediation PR exists or a patch was generated. Closure requires verified evidence:

```text
PR merged → follow-up scan completed → fingerprint absent → original scanner succeeded
```

## Enable

```yaml
evidence_closure_enabled: true
evidence_closure_close_issues: false   # default: comment + label only
evidence_closure_comment: true
evidence_closure_require_scanner_success: true
```

Environment variables (prefer `REPOSITORY_DETECTIVE_*`; legacy `REPOSITORY_DETECTIVE_*` via envcompat):

```text
REPOSITORY_DETECTIVE_EVIDENCE_CLOSURE_ENABLED
REPOSITORY_DETECTIVE_EVIDENCE_CLOSURE_CLOSE_ISSUES
REPOSITORY_DETECTIVE_EVIDENCE_CLOSURE_COMMENT
REPOSITORY_DETECTIVE_EVIDENCE_CLOSURE_REQUIRE_SCANNER_SUCCESS
```

## Closure rule

All must be true to mark `resolved_verified`:

| Requirement | Detail |
|-------------|--------|
| PR merged | Linked patch attempt PR merged in Gitea |
| Post-merge scan | A completed scan ran after merge |
| Fingerprint absent | Original fingerprint not in that scan |
| Scanner evidence | Original source scanner ran successfully (`clean` or `found`) when `require_scanner_success: true` |

If evidence is incomplete, the finding stays open and lifecycle notes explain what is missing.

## Lifecycle states & labels

States: `pending_rescan`, `verified`, `blocked`, `still_present` (stored in `closure_evidence` and finding status).

Gitea labels (dual/legacy/new per `label_compat_mode`):

| Purpose | Legacy | New |
|---------|--------|-----|
| Fix PR opened | `repository-detective/fix-pr-opened` | `repository-detective/fix-pr-opened` |
| Fix PR merged | `repository-detective/fix-pr-merged` | `repository-detective/fix-pr-merged` |
| Pending rescan | `repository-detective/pending-rescan` | `repository-detective/pending-rescan` |
| Resolved verified | `repository-detective/resolved-verified` | `repository-detective/resolved-verified` |
| Closure blocked | `repository-detective/closure-blocked` | `repository-detective/closure-blocked` |

## Workflow

1. **PR opened** — label `fix-pr-opened` (no closure).
2. **On scan finish** — check open patch attempts; if PR merged → `fix-pr-merged`, `pending-rescan`, issue comment.
3. **On scan finish** — verify pending closure evidence against scan fingerprints + scanner results.
4. **Verified** — label `resolved-verified`, comment; close issue only if `close_issues: true`.
5. **Still present / blocked** — comment + appropriate label; issue stays open.

## API

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/v1/findings/:id/closure-evidence` | Latest evidence + blockers |
| POST | `/api/v1/findings/:id/verify-closure` | Verify using latest completed scan |
| POST | `/api/v1/patch-attempts/:attempt_id/check-merge` | Poll Gitea PR merge state |

## Database

Migration **v13**:

- `patch_attempts`: `merged_at`, `merge_commit_sha`, status `pr_merged`
- `closure_evidence` table with verification fields

## Notifications

When global notifications are enabled:

- `fix_pr_merged`
- `closure_verified`
- `closure_blocked`
- `remediation_still_present`

Sanitized summaries only — no raw evidence.

## Rollback

1. Set `evidence_closure_enabled: false` and restart.
2. Existing `closure_evidence` rows are historical.
3. Issues closed manually while enabled remain closed in Gitea.

See also: [REMEDIATION_PRS.md](REMEDIATION_PRS.md), [REMEDIATION.md](REMEDIATION.md), [POLICY.md](POLICY.md).
