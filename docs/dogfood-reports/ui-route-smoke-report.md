# UI route smoke report

Generated: 2026-06-10T12:34:41Z
Base: http://127.0.0.1:8081/ui

| Route | Status | Notes |
|-------|--------|-------|
| `/` | 200 | ok |
| `/repos` | 200 | ok |
| `/repos/1` | 200 | ok |
| `/repos/1/settings` | 200 | ok |
| `/repos/1/containers` | 200 | ok |
| `/repos/1/sbom` | 404 | not found (route or resource may need deploy) |
| `/repos/1/graph` | 200 | ok |
| `/scans` | 200 | ok |
| `/findings` | 200 | ok |
| `/findings/1` | 200 | ok |
| `/configure` | 200 | ok |
| `/learning` | 200 | ok |
| `/health` | 200 | ok |
| `/preinstall` | 200 | ok |
| `/reports` | 200 | ok |

## Summary
- Pass: 14
- Warn: 1
- Fail: 0
