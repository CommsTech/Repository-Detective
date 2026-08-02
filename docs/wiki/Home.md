# Repository Detective wiki

Operator documentation for **Repository Detective** — Inspect. Analyze. Improve.

Live control plane: after install, open `/ui` (see [Quick Start](QUICK_START) and [Public beta guide](https://github.com/CommsTech/Repository-Detective/blob/main/docs/PUBLIC_BETA.md)).

## Start here

- [Quick Start](QUICK_START)
- [Private Beta Install](PRIVATE_BETA_INSTALL)
- [Configuration](CONFIGURATION)
- [FAQ](FAQ)
- [Troubleshooting](TROUBLESHOOTING)
- [Operator runbook](OPERATOR_RUNBOOK)

## Scanning & accuracy

- [Report-only scans](REPORT_ONLY_SCANS)
- [Manual Scan Now](MANUAL_SCAN_NOW)
- [Repo settings and policies](REPO_SETTINGS_AND_POLICIES)
- [Scanner coverage](SCANNER_COVERAGE)
- [Scanner health](SCANNER_HEALTH)
- [Secret scanning and Git history](SECRET_SCANNING_AND_GIT_HISTORY)
- [SBOM](SBOM)
- [Pre-install audit](PREINSTALL_AUDIT)
- [Learning and calibration](LEARNING_AND_CALIBRATION)
- [Issue / finding reconciliation](ISSUE_FINDING_RECONCILIATION)

## Operations & UX

- [Dashboard guide](DASHBOARD_GUIDE)
- [Admin hardening](ADMIN_HARDENING)
- [Privacy and data protection](PRIVACY_AND_DATA_PROTECTION)
- [Accessibility](ACCESSIBILITY)
- [Release readiness](RELEASE_READINESS)
- [Known limitations](KNOWN_LIMITATIONS)
- [Wiki publishing notes](WIKI_PUBLISHING)

Live UI surfaces (homelab control plane): `/ui`, `/ui/repos`, `/ui/findings`, `/ui/scans`, `/ui/health`, `/ui/learning`, `/ui/configure`, `/ui/reports`.

## Developer & agent docs (main repo)

These live in the git tree (not duplicated here):

- [docs/README.md](https://git.commsnet.org/commstech/repository-detective/src/branch/main/docs/README.md) — documentation index
- [AGENT_QUICKSTART](https://git.commsnet.org/commstech/repository-detective/src/branch/main/docs/AGENT_QUICKSTART.md) — API auth for agents
- [MCP](https://git.commsnet.org/commstech/repository-detective/src/branch/main/docs/MCP.md) — MCP stdio bridge
- [OPENCLAW_INTEGRATION](https://git.commsnet.org/commstech/repository-detective/src/branch/main/docs/OPENCLAW_INTEGRATION.md)
- [OpenAPI](https://git.commsnet.org/commstech/repository-detective/src/branch/main/docs/openapi.yaml)
- [API_ROUTES](https://git.commsnet.org/commstech/repository-detective/src/branch/main/docs/API_ROUTES.md) — full route map

## Product

- Repository (wiki / HTML slug): https://git.commsnet.org/commstech/repository-detective
- Clone may use `Repository-Detective` casing; both resolve on this forge
- Auth header: `X-Repository-Detective-API-Key` (or `Authorization: Bearer`)
- Legacy `X-Bugbot-API-Key` is **rejected**
