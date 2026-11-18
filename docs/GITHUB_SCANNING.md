# GitHub scanning

Repository Detective can scan GitHub repositories alongside Gitea when a GitHub personal access token (classic or fine-grained) is configured.

## Configuration

Set in `.env` (or `config/config.yaml`):

| Variable | Description |
|----------|-------------|
| `REPOSITORY_DETECTIVE_GITHUB_TOKEN` | GitHub PAT with `repo` (private repos) and `read:org` if listing org repos |
| `REPOSITORY_DETECTIVE_GITHUB_URL` | API base URL (default `https://api.github.com`; use `https://your-ghe-host/api/v3` for GitHub Enterprise) |

At least one of `gitea_token` or `github_token` must be set. Gitea webhooks and commit status checks remain Gitea-only.

## Manual scan

```bash
curl -sS -H "Authorization: Bearer $REPOSITORY_DETECTIVE_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{"forge_type":"github","owner":"my-org","repository":"my-repo","ref":"main","type":"repository"}' \
  "$REPOSITORY_DETECTIVE_PUBLIC_URL/api/v1/analyze"
```

## Bulk scan

`POST /api/v1/analyze/all` scans all configured forges when `forge` is omitted or `all`.

- `GITEA_SCAN_ORGS` — extra Gitea orgs (comma-separated)
- `GITHUB_SCAN_ORGS` — extra GitHub orgs (comma-separated)

```bash
./scripts/scan-all-forges.sh
# GitHub only:
FORGE=github ./scripts/scan-all-forges.sh
```

Request body fields:

| Field | Values |
|-------|--------|
| `forge` | `all` (default), `gitea`, `github` |
| `orgs` | Override org list for the selected forge(s) |
| `dry_run` | List repos without queuing |
| `scan_profile` | Profile name (e.g. `standard_deterministic`) |

Queued repo names are prefixed (`gitea:owner/repo`, `github:owner/repo`) when both forges are scanned.

## Findings and issues

- Scan results, findings, and the operator UI work for GitHub repos (`forge_type=github` in the database).
- **GitHub Issues** are created/updated with the same fingerprinting, labels, and lifecycle comments as Gitea (`repository-detective/*` labels).
- Pull request–scoped analysis is **Gitea-only** today; full-repo and changed-file scans work on GitHub.
