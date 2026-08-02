# Gitea Bugbot Plugin - Implementation Status

**Last updated:** 2026-08-02  
**Repository:** https://git.commsnet.org/commstech/Bugbot.git

## Live deploy (2026-08-02) — RD findings closeout

| Item | Value |
|------|-------|
| Focus | Clear open findings for `commstech/Bugbot` (repo_id=1) from Repository Detective |
| Code | Gitleaks allowlist + test fixture hardening; workflow SHA pins; reliability error handling |
| Config | `gitleaks_config: config/gitleaks.toml`; expanded `skip_patterns`; calibration seed rules |
| UI smoke | 15/15 routes HTTP 200 (`docs/dogfood-reports/ui-route-smoke-2026-08-02.md`) |
| Follow-up | Hotpatch/redeploy binary, dogfood rescan, run `scripts/closeout-repo1-findings.py` |

## Live deploy (2026-08-01)

| Item | Value |
|------|-------|
| Image | `repository-detective:rc-c45ebb8` (hotpatched scanners base + current `main` binary/entrypoint) |
| `/health` | healthy, ready=true, version=`rc-c45ebb8`, tools **10/10** |
| Ops fixes applied | Embedding model + vector size from `.env`; corrupt grype DB cleared; TMPDIR scratch cleanup; skill-loop JSON/auth fixes on `b18f53c` |
| Learning | Nightly cron `17 2 * * *` → `scripts/rd-deterministic-daily.sh`; manual promote run kicked after redeploy |
| Follow-up | Full `docker build --target all-in-one` once Go-tool install fix (`b18f53c`) is used; refresh expired GitHub token (401 on startup) |

## Current sprint (2026-08-01)

| Item | Status |
|------|--------|
| #352 CVE-2026-39829 (`golang.org/x/crypto`) | **Shipped** on `main` (`c45ebb8`) and live as `rc-c45ebb8` |
| #48 AI/Qdrant connectivity | Soft-fail + `.env` embedding/Qdrant settings loaded into live container |
| Rate-limiter unbounded map | Already fixed on `main` (bounded at 4096) |
| Scanner test TMPDIR leak | Fixed in `scanners/grype_cache_test.go` |
| Skill-loop crash (bytes JSON) + API Bearer auth | Fixed on `main` (`b18f53c`) |

Verify:

```bash
export PATH="$HOME/.local/go/bin:$PATH"
go test -mod=vendor ./issues ./handlers ./internal/auth ./gitea ./scanners -run 'TestEnsureScannerTempDir|TestCleanupStale|Hadolint|Checkov'
go build -mod=vendor -o /tmp/repository-detective .
go list -m golang.org/x/crypto   # expect v0.52.0
```

## Live deploy (2026-07-22 ops hardening)

See `docs/dogfood-reports/container-ops-health-2026-07-22.md`.

Key runtime fixes shipped:

- Scanner temp under `/app/data/tmp` + startup cleanup of abandoned grype/getter scratch (prevents overlay disk fill)
- Grype DB warmup when missing/invalid
- OpenClaw embedding model + 768-d Qdrant collection alignment
- Stronger scheduled-scan ref resolution + default_branch refresh
- `apk-retry.sh` source-safe function (full scanner image install)

## Live deploy (2026-07-13)

| Item | Value |
|------|-------|
| Image | `repository-detective:rc-04db228` (also tagged `all-in-one`) |
| Variant | Full all-in-one with `INSTALL_EXTERNAL_TOOLS=true` |
| Size | ~3.87GB (was ~542MB without external scanners) |
| `/health` | healthy, ready=true, version=`rc-04db228` |
| `tools_summary` | **10/10 available**, missing=[] |
| Scanners present | trivy, grype, gitleaks, semgrep, hadolint, checkov, ruff, shellcheck, gosec, govulncheck, staticcheck |
| Network | host; mounts `config/`, `data/`, `certs/`; `--env-file .env` |

## Current State: BUILD PASSING (Go 1.25) + CORE TESTS PASSING

```bash
go build -mod=vendor -o bin/gitea-bugbot .
go test -mod=vendor ./issues ./handlers ./internal/auth ./gitea
```

## Recent additions

| Feature | Status |
|---------|--------|
| Deterministic static scanner (`analyzers/static.go`) | Done — runs before LLM |
| File content in SCAN stage | Done — fetched via Gitea API |
| `enable_security` / `enable_quality` flags | Done — wired in engine |
| Repository include/exclude filters | Done — webhook + config |
| Issue labels (resolve/create) | Done — `gitea.ResolveLabelIDs` |
| PoC / file / line in issues | Done — from Prove stage |
| Onboarding Web UI | Done — `/onboard` |
| Docker compose env alignment | Done — `docker-compose.minimal.yml` |
| Config unmarshaling fix | Done — `skip_patterns`, `language_mapping`, repo patterns |

## Multi-Provider AI

Supported backends: OpenAI, Anthropic, OpenRouter, Ollama, Open WebUI, OpenClaw

See [docs/AI_PROVIDERS.md](docs/AI_PROVIDERS.md).

## Onboarding

Browser wizard at `/onboard` — see [docs/ONBOARDING.md](docs/ONBOARDING.md).

Requires `public_url` / `BUGBOT_PUBLIC_URL` for webhook registration.

## CI/CD

| Workflow | Trigger | Purpose |
|----------|---------|---------|
| `.gitea/workflows/ci.yml` | push/PR to main | lint, vet, staticcheck, tests, build, Docker smoke |
| `.gitea/workflows/release.yml` | tag `v*` | multi-platform binaries + Gitea release |

Go version pin: **1.25** (matches `go.mod` after `x/crypto` v0.52.0).

## Architecture

```
main.go              → HTTP server, routes, webhook processor
handlers/onboarding  → Web UI + setup API
handlers/webhook     → Rate limit, secret verify, repo filter
analyzers/engine.go  → CAH pipeline + static pre-scan
analyzers/static.go  → Deterministic pattern rules
ai/client.go         → Multi-provider LLM client
gitea/hooks.go       → Repos, webhooks, labels
issues/manager.go    → Labeled issues with PoC
web/                 → Embedded onboarding assets
```

## Configuration (key settings)

```yaml
api_key: ""   # set via .env only — never commit secrets
public_url: "https://bugbot.example.com"
ai_provider: openai
enable_security: true
enable_quality: true
repository_exclude_patterns:
  - "archived-*"
skip_startup_checks: false
```

Legacy `openwebui_url` / `openwebui_token` still supported.
