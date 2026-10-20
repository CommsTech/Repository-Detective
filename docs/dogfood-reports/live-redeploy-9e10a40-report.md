# Live redeploy — container scanning (`9e10a40`)

Recorded: 2026-06-10

## Deploy method

Static binary hot-swap into existing `repository-detective:all-in-one` container (no image rebuild).

```bash
CGO_ENABLED=0 go build -buildvcs=false -ldflags "-s -w -X main.version=9e10a40" -o dist/rd-static .
docker cp dist/rd-static repository-detective:/app/repository-detective
docker restart repository-detective
```

## Verification

| Check | Result |
|---|---|
| `/health` | `healthy`, version `9e10a40` |
| `/api/v1/status` | `running`, `database_healthy: true` |
| `/api/v1/containers/config` | **200** — `enabled: false` (default) |
| `/ui/repos/1/containers` | **401** (route exists; auth required) |
| Core Docker socket | not mounted |
| `preinstall_audit_enabled` | true (report-only) |
| Product `active_present_open` | 0 |
| Runner delegation | false |

## Notes

- Prior live revision was `f06bfd5` (`version: dev`); container API returned 404.
- Post-redeploy container scanning routes are live; scanning remains disabled until opt-in test window.
