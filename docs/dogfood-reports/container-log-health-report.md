# Container log health report

Generated: 2026-06-11T02:08:49Z
Container: `repository-detective` (last 500 lines)

## Pattern counts
- Panics: 0
- Fatals: 0
- DB lock/deadlock: 0
- Auth failures: 0
- AI recommendation failures: 0
- Scanner failures: 11
- Possible secrets in logs: 0

## Known expected
- OpenClaw/provider non-JSON responses when strict JSON not configured on provider side

## Recent errors (sample)
```
time="2026-06-11T01:10:51Z" level=warning msg="Failed to fetch ui/templates/error.html: failed to make request: Get \"https://git.commsnet.org/api/v1/repos/commstech/Repository-Detective/contents/ui/templates/error.html?ref=main\": context deadline exceeded"
```
