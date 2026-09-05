# Integration & quality audit

Last run: automated verification on the development tree. Use `scripts/verify-all.sh` to repeat locally.

## Executive summary

| Area | Status | Notes |
|------|--------|--------|
| Unit / integration tests | **Pass** | `go test ./...` (all packages) |
| `go vet` | **Pass** | |
| `staticcheck` (app code) | **Pass** | Excludes `vendor/` |
| Production build | **Pass** | `go build -o repository-detective .` |
| Docker image build | **Configured** | See `docker-compose.yml` |
| Govulncheck (stdlib path) | **Fixed** | `golang.org/x/net` bumped to v0.38.0+ |
| Secrets in git | **Fixed** | `config/config.yaml` sanitized; secrets in `.env` only |
| UI / charts | **Pass** | Route smoke tests; chart JSON embedding fixed |
| Reporting / labels | **Integrated** | `profile` → engine → `filterIssuesForForge` → `issues.Manager` |

## Safe remediation loop (wired in `main.go`)

```text
Webhook / scheduler / manual scan
  → analyzers.Engine (profile + scanners + optional LLM)
  → store scan recorder (SQLite findings)
  → filterIssuesForForge (reporting_action + policy)
  → issues.Manager → Gitea issues (repository-detective/* labels)
  → UI triage (/ui/findings, repo reports)
  → remediation planner (optional)
  → remediation PR (optional, config off by default)
  → evidence closure (optional verify)
```

### Feature integration matrix

| Feature | Config / entry | Store | UI | Gitea |
|---------|----------------|-------|-----|-------|
| Repo profiling | `profile.*`, engine | Scan summary | Reports | — |
| Reporting modes | `reporting.*` | Findings | Dashboard | Issue gating |
| Deterministic scanners | `enable_*` | Scanner results | Scan detail | Status |
| LLM auditors | `enable_llm_auditors` | Findings | Finding detail | — |
| Auto issues | `auto_create_issues`, `reporting` | External issues | Findings | Create/update |
| Semantic dedup | `qdrant_*` | Optional | — | Skip dupes |
| Scheduler | `scheduler_enabled` | Scans | Scans list | — |
| Gitea status | `enable_gitea_status` | — | — | Commit status |
| Operator UI | `ui_enabled` | All | `/ui/*` | Links |
| Pre-install audit | `preinstall_*` | Audits | Preinstall | — |
| Runners | `runner_*` | Jobs | — | — |
| Remediation PR | `remediation_pr_enabled` | Patch attempts | Finding actions | PR |
| Evidence closure | `evidence_closure_*` | Closure rows | Finding actions | Comments |

## Scan readiness (self-scan / dogfood)

Repository Detective scanning **its own repo** will flag:

1. **Hardcoded secrets** — Use `.env` for tokens; keep `config/config.yaml` free of secrets (see `config/config.yaml.example`). `config/config.yaml` is gitignored for local overrides.
2. **Dependency CVEs** — Run `govulncheck ./...` before release; CI runs this job.
3. **Medium/low volume** — `reporting.mode: high_signal` limits Gitea issues to critical/high + configured categories; expect many dashboard-only findings.
4. **Vendor paths** — `false_positive_reduction.suppress_vendor` suppresses findings under `vendor/` when profile detects them.

### Recommended pre-scan checklist

```bash
./scripts/verify-all.sh
docker compose up -d --build --force-recreate
curl -sf http://127.0.0.1:8081/health
# Trigger dogfood scan (API or Gitea webhook)
```

## CI pipeline (`.gitea/workflows/ci.yml`)

| Job | Command |
|-----|---------|
| Format | `gofmt -s -l .` |
| Vet | `go vet ./...` |
| Staticcheck | `staticcheck ./...` |
| Golangci-lint | `golangci-lint run ./...` |
| Test | `go test -race ./...` |
| Build | `go build` |
| Govulncheck | `govulncheck ./...` |
| Docker | build + `/health` curl |

**Note:** CI `go-version` should match `go.mod` (currently Go 1.23+ after dependency updates).

## Known operational limits (not bugs)

| Item | Behavior |
|------|----------|
| `remediation_pr_enabled: false` | No automatic fix PRs until enabled |
| `max_issues_per_scan: 25` | Caps Gitea noise per run |
| `health` / status endpoints | Tool probes can take ~2s (uncached `CheckTools`) |
| Optional scanners | Missing binary = `binary_missing`, not scan failure |

## Package test coverage

All application packages under `git.commsnet.org/commstech/repository-detective/...` include tests except:

- `cmd/repository-detective-runner` (thin CLI)
- `models/`, `web/` (shared types / static assets)

E2E: `e2e/workflow_test.go` exercises remediation + closure lifecycle on SQLite fakes.
