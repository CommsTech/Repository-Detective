# Privacy and local learning

## What Repository Detective learns locally

- User suppressions and false-positive marks
- Verified fix outcomes and issue lifecycle events
- Rule-level statistics (counts, rates — not code)

## What is never shared

- Source code
- Raw secrets or credential material
- Full issue titles/bodies or raw scanner evidence
- Private repository metadata

All calibration runs **on your SQLite database only**. Issue dedup uses fingerprints and forge mappings stored locally — not an external vector database.

A future community intelligence feed may share **sanitized rule-level statistics** only — not implemented here.

See [CALIBRATION.md](CALIBRATION.md) and [SECURITY_HARDENING.md](SECURITY_HARDENING.md).

## Operator-specific credentials

Repository Detective may be tested with a local operator’s Gitea and API credentials during dogfooding. These credentials are never required by the product and must not be committed.

Use environment variables, Docker secrets, or local untracked config for:

- `REPOSITORY_DETECTIVE_API_KEY`
- `REPOSITORY_DETECTIVE_GITEA_TOKEN`
- `REPOSITORY_DETECTIVE_WEBHOOK_SECRET`
- runner shared secrets
- notification tokens/webhooks

Reports and examples must use placeholders or sanitized values.
