# Post-stabilization rescan report — 2026-06-06

## Trigger

After gate-unblock fixes (Docker build + API auth), rescanned product repo:

```bash
POST /api/v1/analyze
{"owner":"commstech","repository":"Bugbot","ref":"main"}
```

Auth: `X-Repository-Detective-API-Key` header (from `.env` `BUGBOT_API_KEY`)

## Scan result

| Field | Value |
|-------|-------|
| Scan ID | `4a6dadc9b1132982` |
| Status | **completed** |
| Raw findings (scan summary) | 1035 |
| Gitea open issues (before) | 241 |
| Gitea open issues (after) | **243** (+2, no auto-close) |

## Scanner / graph status

- Rescan **started and completed** successfully via API
- Graph fields on scan API response: not populated on summary endpoint (verify via `/ui/scans/{id}/graph` after deploy of stabilization UI)
- Container running image built during gate-unblock (includes stabilization UI commits in source tree at build time)

## Health / capabilities

From `/health` and operator smoke:

- Database: healthy
- Scanners configured/available: 10/10
- Remediation PR: disabled (intentional)
- Runner delegation: disabled (default-off)
- Notifications: disabled (no channels configured)

## Evidence lifecycle

- **No issues manually closed** (`evidence_closure_close_issues=false`)
- Issue count increased slightly — expected without evidence closure
- Next: after Batch 2 fixes, verify fingerprints via `POST /api/v1/findings/{id}/verify-closure`

## Blockers resolved

| Blocker | Status |
|---------|--------|
| Docker build (`apk add git=*`) | **Fixed** |
| API key mismatch | **Fixed** (config loading + container recreate) |
| Rescan API | **Working** |

## Remaining

- Rebuild container from `de879ce` for fully aligned runtime
- CI green gate before Batch 2
- Graph UI verification on scan page post-deploy
