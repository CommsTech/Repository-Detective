# Dogfood reports retention

Most files here are **gitignored by default** (`docs/dogfood-reports/*`).

## Commit to git (force-add when needed)

- Gate/blocker summaries (CI, push readiness)
- Batch queue + verification reports (sanitized)
- Lifecycle burndown summaries (no tokens/hostnames)

## Keep local only

- Raw scanner JSON/CSV exports
- Full issue dumps
- Unsanitized operator notes

## Do not commit

- API tokens, internal hostnames, `.env` contents
- Database dumps

## Growth control

- Prefer one summary report per sprint over many raw exports
- Archive large JSON to operator storage outside the repo
- Reference scan IDs instead of embedding full finding lists
