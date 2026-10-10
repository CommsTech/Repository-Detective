# Manual scan UX

## Overview

Operators and testers can trigger **Scan now** from the UI without waiting for the scheduler.

## Entry points

| Location | Action |
|----------|--------|
| Repository list | **Scan now** per row |
| Repository detail | **Scan now** button |
| Scan detail | **Scan again** (when repo context exists) |
| Dedicated form | `/ui/repos/:id/scan` |

## Confirmation form

The scan form shows:

- Repository name
- Branch/ref (default `main`)
- Optional scan profile override
- **Report-only dry-run** checkbox (default ON when issue filing disabled)
- Issue filing mode summary
- Remediation PR status (off by default)

Submit uses CSRF protection and a browser confirm dialog.

## API

`POST /api/v1/analyze` (unchanged path, enhanced response):

```json
{
  "owner": "org",
  "repository": "repo",
  "ref": "main",
  "scan_profile": "beta_standard",
  "report_only_dry_run": true
}
```

Response:

```json
{
  "status": "analysis started",
  "scan_id": "…",
  "report_only_dry_run": true,
  "trigger_type": "manual"
}
```

## Safety

- Report-only enforced when `issue_policy=off` or global auto-create disabled
- `trigger_type` recorded as `manual`
- Issue filing honors repo effective settings and backlog-control
- No all-repo scan from UI

## After start

UI redirects to `/ui/scans/{scan_id}` for live status.
