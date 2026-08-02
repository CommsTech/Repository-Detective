# Repository Detective - Implementation Status

**Last updated:** 2026-08-02  
**Repository:** https://git.commsnet.org/commstech/repository-detective.git

## Live deploy (2026-08-02) — Full brand purge

| Item | Value |
|------|-------|
| Focus | Zero legacy product-name aliases for public release |
| Shipped | REPOSITORY_DETECTIVE_* only; X-Repository-Detective-API-Key only; rd- fingerprints; repository-detective labels; DB `repository-detective.db` |
| Gitea | https://git.commsnet.org/commstech/Repository-Detective.git |
| Live | `rc-rd-brand-purge` |


## Live deploy (2026-08-02) — Product rename + Gitea sync

| Item | Value |
|------|-------|
| Focus | Sync uncommitted work; public brand is Repository Detective |
| Shipped | Go module `repository-detective`; Gitea repo `commstech/Repository-Detective`; docs/UI scrub; silent legacy env/header/fingerprint shims retained |
| Live hotpatch baseline | `rc-invalid-ref-truth` (+ rename in source) |

## Live deploy (2026-08-02) — Invalid-ref / fleet failure truthfulness

| Item | Value |
|------|-------|
| Focus | `no valid ref` mass failures + dashboard counting historical noise as actionable |
| Finding | July 25–26 fleet failures were forge-probe outages mislabeled as missing refs; repos recovered (latest scans completed) |
| Shipped | ResolveRef returns `unable to verify refs` on probe outages; actionable failures = 14d non-noise; unhealthy-repos = failed latest scan; buckets windowed |
| Live | `rc-invalid-ref-truth` |

## Live deploy (2026-08-02) — Review follow-ups (stale scans / parse failures / AI UX)

| Item | Value |
|------|-------|
| Focus | External review: stale-reaped noise, parse failures, AI enablement friction |
| Shipped | Actionable vs stale failed-scan split; failure reason buckets; parse_failed surfacing; Deep/AI callout on Configure; Pre-install + health CTAs on dashboard |
| Live | `rc-review-followups` |
| Note | ~332 failed scans are **invalid_ref** (missing default branch), not stale reaps (~15). Stale reaps are demoted from primary lists. |

### Deferred backlog (from same review)

| Priority | Item | Why deferred |
|----------|------|--------------|
| HIGH | Root-cause fix for `no valid ref` fleet failures | Needs forge/ref investigation per repo class |
| MEDIUM | Decompose flat `Config` (~180 fields) + split `main.go` bootstrap | Large safe refactor; schedule separately |
| MEDIUM | GitHub forge parity (RC-unproven) | Product expansion |
| MEDIUM | Syft/SBOM completeness | Image/tooling |
| MEDIUM | Finding interactive filters / MTTR charts / exports | UI expansion |
| LOW | RBAC multi-operator | Auth slice 2 |
| LOW | Notifications default-on polish | Ops preference |

## Live deploy (2026-08-02) — System Health UX

| Item | Value |
|------|-------|
| Focus | Scanner versions showed `unknown`; no failure drill-down; no easy product issue report |
| Change | Parallel cached version probes; scanner-failure + failed-scan tables; Report issue prefills `system_health.md` on Gitea (no auto-submit) |
| Live | `rc-health-ux` |
| Note | Hard-refresh `/ui/health`; first version probe after restart may take a few seconds then caches 5m |

## Live deploy (2026-08-02) — Qdrant removed

| Item | Value |
|------|-------|
| Focus | Qdrant semantic dedup unused / empty collections / ops cost |
| Change | Removed `memory/qdrant`, embeddings, semantic issue path; fingerprint + SQLite forge mappings only |
| Live | `rc-no-qdrant` (after hotpatch + env recreate) |
| Monitor | Fingerprint dedup accuracy via SQLite `external_issues` + forge reopen/update behavior |

## Live deploy (2026-08-02) — Intuitive scan profile names

| Item | Value |
|------|-------|
| Focus | Unintuitive profile IDs (`beta_standard`, `fast`, `maintainer_deep`, …) |
| Change | Operator profiles are **Light / Standard / Deep / Custom**; legacy IDs still map |
| Live | `rc-scan-profiles` (after hotpatch) |
| Note | Hard-refresh Configure / Repos / Scan form so labels update |

## Live deploy (2026-08-02) — Learning page graphical UI

| Item | Value |
|------|-------|
| Focus | Learning page was tables/wall of text vs dashboard Learning health cards |
| Live | `rc-learning-ui`; `/ui/learning` has stats, meters, charts, recommendation cards |
| Note | Hard-refresh so `learning-charts.js` / `theme.css` load |

## Live deploy (2026-08-02) — Dashboard 14-day scan trend fix

| Item | Value |
|------|-------|
| Focus | Scan activity graph showed everything on the last day |
| Live | `rc-scan-trend`; chart counts completed scans per UTC day across full 14-day window |
| Note | Hard-refresh dashboard; Jul 25–26 stay at 0 because those days were failed-only in DB |

## Live deploy (2026-08-02) — Manual scan UX + health responsiveness

| Item | Value |
|------|-------|
| Focus | Start scan felt locked; health/UI stalled on tool probes |
| Live | `rc-scan-ux`; `/health` ~1ms after warm; Start scan → scan detail with auto-refresh |
| Note | Hard-refresh browser so embedded `app.js` updates load |

## Live deploy (2026-08-02) — Configure UI save fix

| Item | Value |
|------|-------|
| Focus | Configure page save appeared to do nothing |
| Live | `rc-configure-save`, healthy; `/ui/configure` ~90ms |
| Fix | Skip tool probes on Configure; sticky Save; live apply of feature toggles; clear saved banner |
| Note | Notifications can show **degraded** when enabled but no webhook/Slack/etc. secrets in `.env` — that is expected |

## Live deploy (2026-08-02) — release readiness / fleet burn-down

| Item | Value |
|------|-------|
| Focus | Fleet findings accuracy, feature/UIX matrix, AI token policy, other-repo remediations |
| Live | `rc-release-ready`, healthy, tools **10/10**, `scan_profile=beta_standard` |
| Accuracy | Stable gitleaks RuleIDs; docs/archive/example/vendor actionability downgrades |
| AI defaults | Recommendations **off**; when enabled: 1500/1200 token CAH budget, no snippets/full files |
| Fleet queue | Open unsuppressed ~3.6k (from ~12.6k); high+critical ~67 (from ~900) before remediations settle |
| Other repos | House_Grocery_AI secrets removed from git; optouter CVE/container harden pushed |
| UIX | UI route smoke 19/19; feature-matrix UI/API pass; reconcile CSRF + containers page truth |
| Ops bugfix | Fleet health audit `started_at` TEXT→time parse (repos list warning) |

## Live deploy (2026-08-02) — RD findings closeout

| Item | Value |
|------|-------|
| Focus | Clear open findings for `commstech/Repository-Detective` (repo_id=1) from Repository Detective |
| Commits | `adff149`, `a26a5f1`, plus placeholder TECH-MARKER fix on `main` |
| Live | `rc-adff149`, healthy, tools **10/10** |
| Dogfood scan | `fed458d08455a5f8` completed (report-only) |
| Open queue | **0** unsuppressed open findings (`status=open&suppressed=false`) |
| UI smoke | 15/15 routes HTTP 200 (pre + post closeout) |
| Closeout | `scripts/closeout-repo1-findings.py` + expanded calibration seed |

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
go build -mod=vendor -o bin/repository-detective .
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

Requires `public_url` / `REPOSITORY_DETECTIVE_PUBLIC_URL` for webhook registration.

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
public_url: "https://repository-detective.example.com"
ai_provider: openai
enable_security: true
enable_quality: true
repository_exclude_patterns:
  - "archived-*"
skip_startup_checks: false
```

Legacy `openwebui_url` / `openwebui_token` still supported.
