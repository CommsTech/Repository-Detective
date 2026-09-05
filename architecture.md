# Repository Detective Architecture

## Tech-debt / duplicate paths (Phase 8A)

- Inventory only (no code deleted): [docs/TECH_DEBT_AUDIT.md](docs/TECH_DEBT_AUDIT.md) (RD-030).
- Classifications: ACTIVE_CANONICAL | ACTIVE_COMPATIBILITY | DEPRECATED | DEAD_PROVEN | UNKNOWN.
- Primary scan pipeline remains `analyzers.Engine`; multiple entry points (webhook, API, UI, scheduler, runner delegate) are intentional.

## Public trust / release supply chain (Phase 7)

- Public README identity + limitations: root `README.md`.
- Screenshots / DEMO: `docs/assets/screenshots/`, `docs/DEMO.md` (disposable synthetic only).
- Release mirror tags: `docs/RELEASE_MIRROR.md`, `scripts/publish-github-release-snapshot.sh`.
- Container SBOM (SPDX + CycloneDX) for exact digest: `docs/release/sbom/` via `scripts/generate-release-sbom.sh`.
- Verify / signing: `docs/VERIFY_RELEASE.md` — beta.3 = **CHECKSUM_ONLY**.

## Gitea E2E acceptance (Phase 6A/6B)

- Disposable topology: `docker-compose.e2e.yml` (Gitea **1.22.3** + RD all-in-one).
- Harness: `scripts/e2e-gitea-acceptance.sh` → `e2e/results/<run-id>/acceptance.json`.
- Clean install: `scripts/e2e-clean-install.sh` (git archive + `.env.example`).
- Phase 6B: prove **published** digest (`RD_E2E_IMAGE=@sha256:…`), not a local overlay — see [docs/release/ACCEPTANCE_v0.1.0-beta.3.md](docs/release/ACCEPTANCE_v0.1.0-beta.3.md).
- Operator proofs: `store/operator_evidence` (migration 25) — `webhook.last_delivery`, `proof.first_scan`.
- Finding resolution: [docs/FINDING_RESOLUTION_SEMANTICS.md](docs/FINDING_RESOLUTION_SEMANTICS.md) (no naive absence-close on partial scans).
- Docs: [docs/E2E_GITEA_ACCEPTANCE.md](docs/E2E_GITEA_ACCEPTANCE.md), [docs/DOC_TRUTH_AUDIT.md](docs/DOC_TRUTH_AUDIT.md).

## Privacy / security (Phase 3)

- `privacy_mode`: `local_only` | `hybrid` (default) | `external_ai_enabled` — see `internal/privacy` and [docs/PRIVACY_MODES.md](docs/PRIVACY_MODES.md).
- Threat model honesty: [docs/SECURITY_MODEL.md](docs/SECURITY_MODEL.md) (PROVEN / PARTIAL / NOT_PROVEN / NOT_IMPLEMENTED).
- Auth: runtime default `api_key_only`; recommended new install `auth_mode=local` ([docs/AUTH_LOCAL.md](docs/AUTH_LOCAL.md)).
- PR policy summaries: idempotent upsert via `<!-- repository-detective-policy-summary -->` ([docs/PR_SUMMARY_IDEMPOTENCY.md](docs/PR_SUMMARY_IDEMPOTENCY.md)).

## Onboarding + doctor (Phase 4)

- Wizard: Connect→Select→Protect→Verify→Ready (`/onboard/`, `handlers/onboarding*.go`, `web/static`).
- Shared diagnostics: `doctor` package; CLI `repository-detective doctor`; API `/api/v1/doctor`; UI `/ui/doctor`.
- Class-B gate: [docs/RD-008B_CLASS_B_EXECUTION.md](docs/RD-008B_CLASS_B_EXECUTION.md) (Option C).

## Scan profiles

Operator-facing profiles are **Light**, **Standard**, **Deep**, and **Custom** (`store/profiles.go`). Legacy IDs (`beta_standard`, `fast`, `maintainer_deep`, …) normalize to these. UI pickers show Label — Summary; display helpers use `profileLabel` / `profileDesc`.

Profile-required scanners cannot be removed by disabling the scanner (RD-012A); see [docs/SCAN_PROFILES.md](docs/SCAN_PROFILES.md).

## Issue deduplication

Forge issue filing dedups via **finding fingerprints** and local SQLite `external_issues` mappings (plus forge issue search). There is no external vector / Qdrant integration.


## Scan failure classification

Dashboard/health treat scan `.error` text via `store.ClassifyScanFailure`:
- `stale_reaped` — restart cleanup noise (`IsNoiseScanFailure`); demoted from primary failed lists
- `forge_unavailable` — ResolveRef could not definitively probe refs (API/transport outage)
- `invalid_ref` — missing/default branch after definitive probes
- Plus clone/auth, timeout, prepare, scanner, config, other

**Actionable failed** = non-noise failures in the last **14 days** (not lifetime).  
**Unhealthy repos** = repositories whose **latest** scan failed (excluding restart noise).  
Lifetime `FailedScansCount` remains for historical totals. Parse failures are windowed to 14 days.

## Learning / calibration

Deterministic learning records lifecycle events (`learning_events`), builds per-rule stats, and proposes **repo-scoped** calibration recommendations (false-positive heavy rules → `report_only`).

- Accept (UI `/ui/learning` or API) creates a repo suppression + `repo_calibration_rules` entry; refreshes the suppression matcher.
- Global recommendation accepts are blocked in community beta.
- Secrets/security categories cannot be accepted via calibration; high/critical are never auto-downgraded at persist time.
- Background calibration job mirrors manual recompute (global + per-repo recommendation generation).
- Optional LLM sanity gate remains advisory / off by default — not required for the core learning loop.

## Scanner execution

External scanners run **concurrently** via `scanners.Registry.RunAll` (results keep registry order). Each scanner still has its own timeout (`scanner_timeout_seconds` default 180s; analysis envelope default 900s). Command capture prefers **stdout** for JSON parsers so stderr progress logs no longer cause `parse_failed`.

## UI / dashboard performance

SQLite stays single-writer (`SetMaxOpenConns(1)`), so page latency is dominated by query plans on large tables (`finding_instances`, `findings`, `scanner_results`). Migration **24** adds hot-path indexes. `DashboardSummary` uses a **2s** in-process TTL cache (shared by dashboard, health, reports, API). Repo control metrics reuse latest scan IDs instead of re-aggregating the full `scans` table per count. Scanner platform rollups are windowed to **30 days**.

## SBOM

Go modules prefer `cyclonedx-gomod`; other ecosystems use **Syft**. Both are installed in all-in-one/runner images when `INSTALL_EXTERNAL_TOOLS=true`.

## Install base vs operator data

Published Gitea content is application + docs. Operator `.env`, `config/config.yaml`, and `data/*.db` are gitignored and must not be pushed.


## Core Components

### 1. Control plane (`main.go`, `api/`, `ui/`)
- Authenticated JSON API under `/api/v1` and operator Web UI under `/ui`
- Dashboard / health / findings / learning / reports / configure
- OpenAPI served at `GET /api/v1/openapi.yaml`; MCP bridge in `cmd/repository-detective-mcp`

### 2. Webhook & onboarding (`handlers/`, `web/`)
- Webhook auth, rate limit, include/exclude patterns
- Onboarding wizard at `/onboard` for forge + API key setup

### 3. Scan orchestration (`analyzers/engine.go`, `scanners/`)
- Primary path: external **scanner registry** (trivy, grype, gitleaks, semgrep, Go/IAC/linters) run concurrently
- SBOM generation (`sbom/`) via cyclonedx-gomod / Syft
- Optional LLM CAH stages (prepare → scan → validate → prove) when AI policy enables them

### 4. Persistence & learning (`store/`, calibration UI/API)
- SQLite findings, scans, suppressions, external issue mappings
- Repo-scoped calibration recommendations (`/ui/learning`, `/api/v1/calibration/*`)

### 5. Forge & remediation (`gitea/`, `issues/`, remediation/closure packages)
- Issue filing, reconciliation, optional remediation PRs and evidence closure

### 6. Runners & containers
- Optional HMAC runner delegation and container image scanning routes

## Data Flow

```
Webhook / Analyze API / scheduled Scan Now
    → clone workspace
    → scanners.Registry.RunAll (parallel external tools)
    → sbom.Generate (cyclonedx-gomod / Syft)
    → optional LLM CAH stages when AI policy enabled
    → persist findings + scanner_results + scan summary
    → optional forge issues / remediation / closure
```

## Module Path

```
git.commsnet.org/commstech/repository-detective
```

## Configuration

- Forge: `gitea_url`, `gitea_token`, `webhook_secret`, `public_url`
- Auth: `api_key` / `REPOSITORY_DETECTIVE_API_KEY`
- Per-scanner enables (`enable_trivy`, `enable_gitleaks`, …) and scan profiles (Light/Standard/Deep/Custom)
- Timeouts: `analysis_timeout_seconds`, `scanner_timeout_seconds`
- Optional AI: `ai_provider`, `ai_base_url`, `ai_api_key`, `ai_model` (off by default; `needsAIProvider()` only when LLM auditors enabled at depth ≥ 3)
- Include/exclude and skip patterns for repositories and files

### Public beta support model

- **GitHub Issues** — public bug/feature/install/scanner feedback
- **Gitea** — canonical development forge (CI, wiki, maintainers)
- **SECURITY.md** — product vulnerabilities (private advisory preferred)

### Recommended deployment

`docker-compose.yml` + published all-in-one image, port **8081**, bridge networking (host-network overlay optional).

See [docs/CONFIGURATION.md](docs/CONFIGURATION.md), [docs/ONBOARDING.md](docs/ONBOARDING.md), [docs/AI_PROVIDERS.md](docs/AI_PROVIDERS.md), and [docs/DOC_TRUTH_AUDIT.md](docs/DOC_TRUTH_AUDIT.md).
