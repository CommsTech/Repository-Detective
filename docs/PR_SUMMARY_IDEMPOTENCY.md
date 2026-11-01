# PR policy summary idempotency (RD-006A)

Repository Detective posts **at most one** compact policy summary comment per pull request.

## Marker

```html
<!-- repository-detective-policy-summary -->
```

Only comments containing this **exact** marker are treated as RD-owned. User comments and lookalike markers are never modified or deleted.

## Algorithm

1. Acquire a short in-process lock keyed by `owner/repo#pr` (webhook retry safety).
2. List issue comments on the PR (Gitea PR comments use the issue comment API).
3. Collect RD-owned comments (exact marker).
4. **None** → create one comment.
5. **One or more** → update the first in place; best-effort delete additional RD-owned duplicates.
6. If **listing fails**, do **not** create a new comment (fail closed — avoids uncontrolled duplicates).

Later commits on the same PR update the same comment with the latest outcome, scan ID, commit SHA, coverage, finding count, and timestamp.

## Status

| Capability | Proof |
|------------|-------|
| Upsert create/update | UNIT_TESTED (`issues` package) |
| Dedupe legacy duplicates | UNIT_TESTED |
| Fail closed on list error | UNIT_TESTED |
| Wired on PR scan path | WIRED (`maybePostPRPolicySummary`) |
| Live Gitea E2E | NOT_PROVEN (RD-017) |
