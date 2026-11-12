# Repo 31 direct Scan Now verification

Date: 2026-06-02  
Commit: `c528dbc`  
Live URL: `http://127.0.0.1:8081/ui/repos/31`  
Repository: `commstech/AMMBER` (repo ID 31)

## Deploy

| Item | Value |
|------|-------|
| Image rebuilt | YES |
| Image revision | `c528dbc` |
| Container | `repository-detective` (host network) |

## Route verification

| Check | Result |
|-------|--------|
| `GET /ui/repos/31` | 200 (API key header) |
| Scan Now button (`#rd-scan-now-open`) | Present |
| Inline modal (`#rd-scan-now-modal`) | Present |
| Reconciliation panel | Present (`#issue-finding-reconciliation`) |
| Favicon | `favicon.svg?v=2`; `/favicon.ico` → 302 redirect |
| Deep link `/ui/repos/31/scan` | Available (full-page form) |

## Manual scan (report-only)

Triggered via API (same engine as modal POST):

```json
{
  "owner": "commstech",
  "repository": "AMMBER",
  "ref": "main",
  "report_only_dry_run": true
}
```

| Field | Value |
|-------|-------|
| Scan ID | `513488d52c871497` |
| Trigger type | `manual` |
| Report-only | `true` |
| Scan status | `completed` |
| Issues created | **0** |
| PRs created | **0** |

## Reconciliation (AMMBER validation case)

| Count | Value |
|-------|-------|
| Scan findings | 249 |
| Forge open issues (mapped DB) | 1 |
| Mapped open issues | 1 |
| Findings without forge issue | 304 |
| Skipped (report-only) | 304 |
| `counts_differ_expected` | true (expected — report-only + unmapped findings) |

Explanation text visible on page: scan findings vs forge issues may differ; report-only persists findings without filing Gitea issues.

## Tests

| Command | Result |
|---------|--------|
| `go test ./...` | PASS |
| `go vet ./...` | PASS |
| `staticcheck ./...` | PASS |
| `operator-smoke-test.sh` | PASS |
| `make beta-release` | PASS |

## Safety gates

| Gate | Status |
|------|--------|
| Non-product issue filing default | Disabled in beta bundle |
| All-repo scan | Not started |
| Issues created during verification | 0 |
| PRs created during verification | 0 |
| Backlog-control | Active (homelab config) |

## Remaining UX follow-ups (non-blocking)

- Browser modal flow: confirm in UI (curl verified HTML; modal uses fetch + CSRF from rendered form)
- Live homelab `issue_filing_enabled` may read true globally while report-only scans still skip filing — panel shows both states

## Recommendation

**Ready for tester re-check** on `/ui/repos/31` — Scan Now is on the repo detail page with reconciliation adjacent.
