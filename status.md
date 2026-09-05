# Repository Detective - Implementation Status

**Last updated:** 2026-09-05  
**Program:** Product Hardening & Public Beta Improvement Backlog (RD-001…RD-030)

### Phase 8A (2026-09-05) — Beta hardening & real-use readiness — COMPLETE

| Task | Status | Notes |
|------|--------|-------|
| RD-031 gofmt debt | **Done** | 140 files; CI `check-fmt` / workflow fails unclean |
| RD-032 new-install auth | **Done** | Recommend `AUTH_MODE=local`; runtime default `api_key_only` unchanged |
| RD-033 upgrade harness | **Done** | `scripts/e2e-upgrade-from-beta3.sh` → `UPGRADE_FROM_BETA3_TO_CURRENT_MAIN_INTEGRATION_PROVEN` when PASS |
| RD-034 redaction | **Done** | `SanitizeDiagnostic` + corpus; remaining heuristic limits documented |
| RD-035 dogfood | **Done** | [docs/release/DOGFOOD_2026-09-05.md](docs/release/DOGFOOD_2026-09-05.md) |
| RD-024 finding-quality metrics | **Done** | `GET /analytics/finding-quality?window=7d\|30d\|all` |
| RD-025 calibration transparency | **Done** | `/calibration/history` + accepted revert |
| RD-030 tech-debt audit | **Done** | [docs/TECH_DEBT_AUDIT.md](docs/TECH_DEBT_AUDIT.md) — **no code deleted** |
| Class-B / RD-015–016 | **Excluded** | Unchanged |

### Phase 7 (2026-09-05) — Public trust, presentation, release supply chain — COMPLETE

| Task | Status | Notes |
|------|--------|-------|
| RD-019 Public first impression | **Done** | README hierarchy, POLICY_*, privacy honesty, limitations, GitHub history |
| RD-020 Screenshots + DEMO | **Done** | Disposable synthetic captures under `docs/assets/screenshots/`; DEMO.md |
| RD-021 Public release surface | **Done** | Release notes + mirror process; GitHub tag/Release for beta.3 |
| RD-022 Container SBOM | **Done** | SPDX + CycloneDX for digest `sha256:6a615548…`; Syft 1.45.1 |
| RD-023 Integrity / signing | **CHECKSUM_ONLY** | VERIFY_RELEASE.md; SIGNING_NOT_IMPLEMENTED for cosign |
| Class-B / RD-015–016 | **Excluded** | RD-008B Option C unchanged |

### Phase 6B (2026-09-05) — Release alignment & proof closure — COMPLETE

| Task | Status | Notes |
|------|--------|-------|
| RD-018A Publish beta.3 | **Done** | Digest `sha256:6a615548…308727` on Gitea + GHCR (match) |
| RD-018B Clean install on published digest | **PUBLISHED_IMAGE_CLEAN_INSTALL_E2E_PROVEN** | Doctor present; empty storage |
| RD-017B Core E2E on published digest | **PUBLISHED_IMAGE_CORE_E2E_PROVEN** | Gitea 1.22.3; zero FAIL |
| RD-017C Four policy outcomes | **E2E_PROVEN** | POLICY_MET / ACTION_REQUIRED / OBSERVATION_ONLY / EVALUATION_INCOMPLETE |
| RD-017D Secret resolution semantics | **PARTIAL** intentional | Documented; no naive absence-close |
| RD-029A DOC_TRUTH drift | **Fixed** | Snapshot lag root cause |
| RD-021A Minimal badges | **Done** | CI / beta.3 / container / license / Gitea 1.22.3 |
| Class-B / RD-015–016 | **Excluded** | RD-008B Option C unchanged |

### Phase 6A (2026-09-04) — Real Gitea E2E (RD-017A / RD-018) — COMPLETE (stop before Phase 5)

| Task | Status | Notes |
|------|--------|-------|
| Disposable Gitea 1.22.3 + RD compose | **Done** | `docker-compose.e2e.yml`; host ports 13000/18081 |
| Webhook delivery + FIRST_SCAN evidence | **E2E_PROVEN** | migration 25 `operator_evidence`; Doctor proofs |
| E2E harness scenarios | **PASS** | `e2e/results/20260904T182636Z-2505621/` — zero FAIL |
| Required-scanner fail-closed | **E2E_PROVEN** | Controlled gitleaks stub |
| PR summary idempotency | **E2E_PROVEN** | RD-006A at Gitea 1.22.3 |
| Clean install RD-018 | **PASS** | health+onboard+scanners; Doctor absent on published beta.2 digest |
| Upgrade E2E | **NOT_PROVEN** | No prior public-beta baseline |
| Class-B remediation / RD-015–016 | **Excluded** | RD-008B Option C unchanged |

### Phase 1 (2026-09-04) — Public-beta contradictions

| Task | Status | Notes |
|------|--------|-------|
| RD-001 Public feedback path | **Done** | GitHub Issues = public feedback; Gitea canonical forge |
| RD-002 Recommended Installation | **Done** | Compose pull :8081 |
| RD-003 AI explicitly optional | **Done** | Deterministic-first defaults |
| RD-029 Doc truth audit | **Done** | Updated through Phase 3 |

### Phase 2 (2026-09-04) — Product semantics — CLOSED (unit/integration)

| Task | Status | Notes |
|------|--------|-------|
| RD-004 Policy outcomes | **Done** | Never “secure/safe” |
| RD-005 Observe/Warn/Enforce | **Done** | UI labels over monitor/issue/gate |
| RD-006 Compact PR summary | **Done** | One PR comment; issues canonical |
| RD-006A Idempotent PR summary | **Done** | Marker upsert; fail-closed list; UNIT_TESTED |
| RD-011 Scanner coverage | **Done** | Required incomplete blocks POLICY_MET |
| RD-012 / RD-012A Required scanners | **Done** | Disabled REQUIRED → EVALUATION_INCOMPLETE |

**Regression:** `go test ./...` in `golang:1.25-bookworm` — **PASS** (exit 0).  
**Classification:** `IMPLEMENTED + UNIT/INTEGRATION TESTED; Gitea 1.22.3 E2E advanced in Phase 6A (RD-017A)`.

### Phase 3 (2026-09-04) — Privacy / security

| Task | Status | Proof |
|------|--------|-------|
| RD-007 Privacy modes | **Done** | CODE_PRESENT + WIRED + UNIT_TESTED |
| RD-008 Threat model + MinimalSubprocessEnv | **Done** | SECURITY_MODEL.md; Class B sandbox NOT_PROVEN |
| RD-009 Credential transport | **Done** | Header preferred; query reject optional; redaction UNIT_TESTED |
| RD-010 UI session vs API auth | **Done** | Recommend local for new installs; runtime default api_key_only unchanged |

### Phase 4 (2026-09-04) — Onboarding + Doctor

| Task | Status | Proof |
|------|--------|-------|
| RD-013 CSPVR onboarding | **Done** | WIRED wizard stages + verify API |
| RD-014 Doctor | **Done** | CLI + `/api/v1/doctor` + `/ui/doctor` + UNIT_TESTED engine |
| RD-008B Class-B decision | **Done** | Option C documented; gate for Phase 5 |

**Next:** Phase 5 only after RD-008B Option C warnings remain / Option A runner path if expanding remediation UX.

**Repository:** https://git.commsnet.org/commstech/Repository-Detective.git
## Live deploy (2026-08-02) — Full application audit

| Item | Value |
|------|-------|
| Verdict | **Conditional GO** — accuracy/docs/wiki fixed; residual Go 1.23 image vs go.mod 1.25 |
| Evidence | [docs/dogfood-reports/full-application-audit-2026-08-02.md](docs/dogfood-reports/full-application-audit-2026-08-02.md) |
| Live | `rc-full-audit7` |
| Dashboard tools missing | **0** (live probe overlay) |
| SBOM | Syft fallback → **`sbom_generated` / 58 packages** |
| Shellcheck / Trivy | **found** on product scans |
| Grype | Rebuilt DB under container `XDG_CACHE_HOME=/app/data/cache` (was malformed; `$HOME/.cache` rebuild was the wrong path) |
| Wiki | **24 pages** at https://git.commsnet.org/commstech/repository-detective/wiki |

## Live deploy (2026-08-02) — Full UI eval clean pass

| Item | Value |
|------|-------|
| Verdict | **36/36 OK** (light/dark/system) |
| Evidence | [docs/dogfood-reports/ui-flow-eval-2026-08-02.md](docs/dogfood-reports/ui-flow-eval-2026-08-02.md) |
| Harness | `scripts/ui-flow-eval.js` |
| Live | `rc-ui-eval-clean` |
| Follow-up fixed | Configure “missing” secret contrast; table/`details` solid surfaces |

## Live deploy (2026-08-02) — SBOM tools in base image

| Item | Value |
|------|-------|
| Focus | Health no longer reports syft / cyclonedx-gomod as missing binaries |
| Shipped | Image `repository-detective:rc-sbom-tools` (also tagged `all-in-one`); Dockerfile verify gate; install script requires syft |
| Live | `tools_summary` **12/12 available**, `missing: []` |
| Versions | syft **1.18.1**, cyclonedx-gomod **v1.10.0** |

## Live deploy (2026-08-02) — Full WebUI flow evaluation

| Item | Value |
|------|-------|
| Focus | Browser walk of all operator pages in light/dark/system; branding + theme + charts |
| Verdict | **36/36 pages OK** (Playwright headless against live `:8081`) |
| Shipped | Theme contrast (KPI/inset/report solids); theme-aware charts; nav Learning under Intelligence; brand scrub for historical Bugbot text/paths; project group names |
| Live | Hotpatched `rc-ui-flow-eval2` |
| Branding | No Bugbot in UI templates; historical forge/errors/paths scrubbed at display |

## Live deploy (2026-08-02) — Agent / MCP / OpenAPI docs

| Item | Value |
|------|-------|
| Focus | Make Repository Detective usable by OpenClaw-like AI agents |
| Shipped | `docs/AGENT_QUICKSTART.md`, `docs/MCP.md`, `docs/OPENCLAW_INTEGRATION.md`, `docs/openapi.yaml`, MCP stdio bridge `cmd/repository-detective-mcp`, `GET /api/v1/openapi.yaml`, richer `GET /api/v1/about` |
| Agent auth | `X-Repository-Detective-API-Key` or Bearer |

## Live deploy (2026-08-02) — UI responsiveness

| Item | Value |
|------|-------|
| Focus | Benchmark all UI pages; keep only net-faster changes |
| Shipped | Migration 24 indexes; repo-control query rewrites; batched dashboard charts; 2s dashboard summary cache; 30d scanner rollups |
| Evidence | [docs/dogfood-reports/ui-responsiveness-bench-2026-08-02.md](docs/dogfood-reports/ui-responsiveness-bench-2026-08-02.md) |
| Live | Hotpatched `/app/repository-detective` (schema v24 applied on data volume) |
| Gains (cold p50) | `/ui` −58%, `/ui/repos` −73%, `/ui/health` −67%, `/ui/reports` −62%, dashboard API −68% |

## Live deploy (2026-08-02) — Scanner reliability + SBOM + triage export

| Item | Value |
|------|-------|
| Focus | Address external review: parse failures, timeouts, SBOM gap, finding triage volume |
| Shipped | stdout-first command capture; parallel scanner registry; Syft/cyclonedx-gomod in image; focus list + CSV/JSON export; timeout defaults 900s/180s |
| Requires | Image rebuild for Syft on live (`all-in-one`); hotpatch covers parser/parallel/export immediately |
| Deferred | Config struct decomposition, main.go route extract, GitHub forge parity, full LLM prove stage |

## Live deploy (2026-08-02) — Sanitized install base + learning completeness

| Item | Value |
|------|-------|
| Focus | Confirm Gitea is a clean install base; operator DB never published; learning accept path fixed |
| Shipped | Accept bugfix; background job repo-scoped parity; Learning UI actions; LAN IP redaction in tracked docs; privacy/setup clarity |
| Not in git | `.env`, `config/config.yaml`, `data/*.db` (gitignored) |
| Learning | Deterministic loop complete for FP→repo recommendation→accept/reject; global accept blocked; secrets categories blocked |

## Live deploy (2026-08-02) — Prime-time readiness evaluation

| Item | Value |
|------|-------|
| Verdict | **Conditional GO** for private beta; not public prime-time yet |
| Evidence | [docs/dogfood-reports/prime-time-readiness-2026-08-02.md](docs/dogfood-reports/prime-time-readiness-2026-08-02.md) |
| Blockers | 244 critical+high open findings; 251 parse_failed (14d); rebuild image from main |

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
