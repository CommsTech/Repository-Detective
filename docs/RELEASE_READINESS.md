# Release readiness

Checklist for deploying Repository Detective. Evidence paths are relative to repo root.

## Build and test

| Check | Command | Notes |
|-------|---------|-------|
| Unit tests | `docker run --rm -v $PWD:/src -w /src golang:1.25-bookworm go test ./... -count=1` | Match `go.mod` (`go 1.25`) |
| Vendor build | `CGO_ENABLED=0 go build -mod=vendor -o build/repository-detective .` | Production binary |
| Docker health | `curl -sf http://127.0.0.1:8081/health` | Expect `status: healthy`, `tools_summary.missing: []` |

## Functional smoke

| Area | Verify |
|------|--------|
| Dashboard | `/ui/` loads; charts or empty states |
| Scanner health | `/ui/health` — live tools match `/health` |
| Findings | `/ui/findings` queue loads |
| Learning | `/ui/learning` accept/reject/recompute |
| Configure | `/ui/configure` platform settings |
| Reports | `/ui/reports` executive summary |
| Scans UI | `/ui/scans` list |
| API | `GET /api/v1/status` + `GET /api/v1/repos/:id/scans` with API key |

## Documentation

- [x] Privacy, accessibility, scanner health, dashboard, admin hardening
- [x] Known limitations + this checklist
- [x] Agent / MCP / OpenAPI (`docs/AGENT_QUICKSTART.md`, `docs/MCP.md`, `docs/openapi.yaml`)
- [x] Wiki source under `docs/wiki/` — publish with `scripts/publish-gitea-wiki-api.py`

## Wiki

- Live: https://git.commsnet.org/commstech/repository-detective/wiki
- Source: `docs/wiki/`
- Publisher: API script (git `*.wiki.git` push may HTTP 500) — see [WIKI_PUBLISHING.md](WIKI_PUBLISHING.md)

## Self-scan (dogfood)

- Guide: [DOGFOODING.md](DOGFOODING.md)
- Script: [scripts/dogfood-self-scan.sh](../scripts/dogfood-self-scan.sh)
- After SBOM/tool image upgrades, always rescan before trusting historical scan summaries

## Sign-off

| Role | Status |
|------|--------|
| Engineering | Tests + live health verified on deploy |
| Security | Privacy-aware handling documented; not compliance certified |
| Accessibility | WCAG-aligned improvements; formal audit not run |
| Operations | Admin hardening + retention docs provided |
