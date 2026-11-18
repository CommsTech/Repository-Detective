# Findings export contract

Audience: operators and API clients. This is the **expected contract**, not proof that a particular deployed version has passed end-to-end tests.

## Applied filters

The **Findings** page exports the *effective* filters used for the currently displayed queue:

| Filter | Behavior |
| --- | --- |
| `q` | Literal search across indexed finding metadata (title, rule, package, file, fingerprint, repository). Wildcards are escaped. |
| `repo_id`, `severity`, `category`, `source` | Exact scoped values where specified. |
| `status` | Lifecycle filter; selecting `suppressed` or `false_positive` automatically includes the corresponding records. |
| `focus=1` | Always means **open critical/high**; overrides stale status or suppression options. |
| `owner=mine` / `owner=unassigned` | Requires a local authenticated user, with identity derived from the session. The generic API does not accept owner scope. |
| `show_suppressed`, `only_suppressed` | Explicit inclusion/exclusion; the same rules apply to view and export. |
| `page`, `limit`, `offset` | View pagination is **not** an export limit. Exports include the whole matched set up to 20,000 findings. |

CSV and JSON should return the **same finding IDs** for an identical effective filter. CSV neutralizes spreadsheet formula injection; JSON retains exact field values. API exports do not silently truncate at the old 5,000-row cap.

Exports are rejected if they exceed 20,000 rows or the stored set changes between count and retrieval. They are best-effort snapshots, **not transactionally frozen or cryptographically signed audit artifacts**. For authoritative audit evidence, record the scan and commit IDs and use the report/verification workflow.

## Acceptance cases

1. Apply `repo_id`, `status=open`, `severity=high`, and source filters. Compare visible total and exported CSV/JSON counts and IDs.
2. Apply a literal `q` containing `%` or `_`. It should not expand to wildcard matches.
3. Navigate to page three and export: the export still includes every matching row.
4. Open Priority with an outdated `status=resolved`: the export contains only open critical/high.
5. Export `status=suppressed` explicitly: results must not disappear because `show_suppressed` is absent.
6. In local login mode, My Work exports include only the current user's assignments; the API cannot request a different person's queue.
7. Simulate 20,001 results, concurrent changes, a database error, and CSV formula-like titles: no truncated success response and no executable spreadsheet formula.

Unit tests: `go test -count=1 ./api ./ui` with Go 1.26.11. Staging browser verification: `scripts/ui-findings-export-acceptance.js` plus direct API/HTML parity requests using the same scoped data.
