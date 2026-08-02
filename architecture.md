# Repository Detective Architecture

## Scan profiles

Operator-facing profiles are **Light**, **Standard**, **Deep**, and **Custom** (`store/profiles.go`). Legacy IDs (`beta_standard`, `fast`, `maintainer_deep`, …) normalize to these. UI pickers show Label — Summary; display helpers use `profileLabel` / `profileDesc`.

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
- Optional AI: `ai_provider`, `ai_base_url`, `ai_api_key`, `ai_model` (off by default for beta)
- Include/exclude and skip patterns for repositories and files

See [docs/CONFIGURATION.md](docs/CONFIGURATION.md), [docs/ONBOARDING.md](docs/ONBOARDING.md), and [docs/AI_PROVIDERS.md](docs/AI_PROVIDERS.md).
