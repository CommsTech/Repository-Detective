# Release readiness

Checklist for deploying Repository Detective. Evidence paths are relative to repo root.

## Build and test

| Check | Command | Notes |
|-------|---------|-------|
| Unit tests | `go test ./... -count=1` | Must run with Go **1.26.11** (toolchain directive in `go.mod`); use an isolated CI runner with network/module access |
| Vet | `go vet ./...` | No new static analysis findings |
| Source build | `CGO_ENABLED=0 go build -o build/repository-detective .` | The supplied archive has no `vendor/`; do not use `-mod=vendor` |
| Docker readiness | `curl -fsS http://127.0.0.1:8081/health/ready` | Expect HTTP 200 after initialization; liveness alone is insufficient |

## Functional smoke

| Area | Verify |
|------|--------|
| Dashboard | `/ui/` loads; charts or empty states |
| Scanner health | `/ui/health` — live tools match `/health` |
| Findings | `/ui/findings?focus=1` includes old criticals before newer medium noise; all pages and CSV/JSON export agree with the effective filters and DB total; see [export contract](EXPORT_FILTER_CONTRACT.md) and [beta acceptance matrix](BETA_RELEASE_ACCEPTANCE_2026-11-18.md) |
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

- Live: https://git.example.com/<your-org>/repository-detective/wiki
- Source: `docs/wiki/`
- Publisher: API script (git `*.wiki.git` push may HTTP 500) — see [WIKI_PUBLISHING.md](WIKI_PUBLISHING.md)

## Self-scan (dogfood)

- Guide: [DOGFOODING.md](DOGFOODING.md)
- Script: [scripts/dogfood-self-scan.sh](../scripts/dogfood-self-scan.sh)
- After SBOM/tool image upgrades, always rescan before trusting historical scan summaries

## Authentication and portability

- [Local accounts and bootstrap](AUTH_LOCAL.md) are available; [Authentik/Gitea SSO and RBAC](ACCESS_AND_SSO_DESIGN.md) are **not** production-complete.
- Benchmark authenticated navigation and scanner throughput before claiming performance improvements.
- Release tags and mirrored images require successful CI, complete scan coverage, an image-digest record, and rollback evidence.

## Sign-off

| Role | Status |
|------|--------|
| Engineering | Tests + live health verified on deploy |
| Security | Privacy-aware handling documented; not compliance certified |
| Accessibility | WCAG-aligned improvements; formal audit not run |
| Operations | Admin hardening + retention docs provided |
