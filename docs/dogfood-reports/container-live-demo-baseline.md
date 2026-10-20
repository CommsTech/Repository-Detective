# Container live demo baseline

Recorded: 2026-06-10  
Latest git commit: `9e10a40`

## Product dogfood

| Metric | Value |
|---|---:|
| Active-present | 0 |
| High/critical | 0 |
| Latest product scan | `95a5551881e866d4` |

## Live state (pre-redeploy)

| Item | Value |
|---|---|
| Live revision | `f06bfd5` (static binary hot-deploy; version reports `dev`) |
| Container API | **404** (not on live yet) |
| Container UI | **404** |
| Core Docker socket | not mounted |
| Runner delegation | disabled |
| Container scanning enabled | false (default) |
| Container scan create issues | false |

## Runner state

| Item | Value |
|---|---|
| Native worker running | no |
| Runner shared secret | configured in `.env` (not committed) |
| Allowed job types (.env) | graph, sbom, remediation_verify only |

## Scanner tools (host)

| Tool | Available |
|---|---|
| trivy | yes (`~/.local/bin/trivy`) |
| docker | yes |
| grype | no (PATH) |
| syft | no (PATH) |

Live all-in-one `/health` tools_summary: 4/10 (govulncheck, gosec, staticcheck, linters).

## Wiki

| Item | Value |
|---|---|
| Gitea wiki populated | no |
| Blocker | HTTP 500 on wiki git push |

## Marketing blockers

1. Gitea wiki HTTP 500
2. Live redeploy to `9e10a40`
3. Container scan live demo
4. Pre-install public repo scale test
5. 2+ non-product beta scans
6. Scanner coverage UX polish
7. External clean install test
