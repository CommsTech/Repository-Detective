# Post-stabilization rescan report — 2026-06-06 (final)

## Trigger

After gate-unblock fixes (Docker build + API auth):

```bash
POST /api/v1/analyze
{"owner":"commstech","repository":"Repository-Detective","ref":"main"}
```

Auth: `X-Repository-Detective-API-Key` header (from `.env` `REPOSITORY_DETECTIVE_API_KEY`)

## Scan results

| Field | Scan 1 | Scan 2 (post-stabilization) |
|-------|--------|----------------------------|
| Scan ID | `4a6dadc9b1132982` | **`f85f8e66e3c9fc9a`** |
| Status | completed | **completed** |
| Started | — | 2026-06-06T15:59:02Z |
| Finished | — | 2026-06-06T16:04:02Z |
| Duration | — | ~299s |
| Findings | 1035 | **1036** |
| Files analyzed | — | 618 |
| Overall score | — | 0.6 |

## Scanner status

All 10 configured scanners enabled and ran:

`trivy`, `grype`, `gitleaks`, `semgrep`, `govulncheck`, `gosec`, `staticcheck`, `hadolint`, `checkov`, `linters`

No scan-level failure; results persisted to database.

## Graph status

| Field | Value |
|-------|-------|
| Code graph enabled | yes |
| Graph nodes | **3062** |
| Graph edges | **5026** |
| Graph state | **populated** (scan summary includes graph counts) |

## Executive report / system health

From operator smoke + dashboard:

| Capability | Status |
|------------|--------|
| Database | healthy |
| Scanners | 10/10 configured, 10/10 available |
| Remediation PR | disabled (intentional) |
| Runner delegation | disabled |
| Notifications | disabled |
| Evidence closure | enabled (`evidence_closure_close_issues=false`) |

## Gitea issue count

| Metric | Value |
|--------|-------|
| Open issues (before gate-unblock) | 241 |
| Open issues (after rescan) | **244** |
| Manual closes | **none** |

## Comparison vs prior scan

| Metric | Delta |
|--------|-------|
| Findings | +1 (1035 → 1036) |
| Graph nodes | populated (3062) |
| API auth | fixed — rescan no longer returns `Invalid API key` |

## Blockers resolved

| Blocker | Status |
|---------|--------|
| Docker build (`apk add git=*`) | **Fixed** |
| API key mismatch | **Fixed** |
| Rescan API | **Working** |
| Scanner persistence | **Verified** |

## Remaining

- CI green gate before Batch 2
- Container rebuild from latest `main` optional (runtime functional on current image)
