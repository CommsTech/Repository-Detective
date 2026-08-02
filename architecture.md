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

## SBOM

Go modules prefer `cyclonedx-gomod`; other ecosystems use **Syft**. Both are installed in all-in-one/runner images when `INSTALL_EXTERNAL_TOOLS=true`.

## Install base vs operator data

Published Gitea content is application + docs. Operator `.env`, `config/config.yaml`, and `data/*.db` are gitignored and must not be pushed.


## Core Components

### 1. Webhook Handler (`handlers/webhook.go`)
- Rate limiting per client IP
- Webhook secret verification (HMAC-safe)
- Repository include/exclude pattern filtering
- Dispatches push and pull request events to the analysis processor

### 2. Onboarding UI (`web/`, `handlers/onboarding.go`)
- Embedded static wizard at `/onboard`
- Tests Gitea and AI connections
- Lists repositories and registers webhooks
- Exports environment variables for deployment

### 3. Analysis Engine (`analyzers/engine.go`)
- **Prepare** — repository structure and attack surface mapping
- **Scan** — static pattern rules, then LLM auditors with full file content
- **Validate** — advocate/counsel debate (high-confidence static hits skip debate)
- **Dedup** — merge findings by location
- **Prove** — generate proof-of-concept for validated findings

### 4. Static Scanner (`analyzers/static.go`)
- Deterministic regex rules for SQL injection, secrets, XSS, command injection, debug logging
- Runs before any LLM call
- LLM auditors target files flagged by static analysis when possible

### 5. AI Integration (`ai/`)
- Provider abstraction: OpenAI-compatible and Anthropic Messages APIs
- Supported: OpenAI, Anthropic, OpenRouter, Ollama, OpenWebUI, OpenClaw
- Auditor prompts include fetched source code

### 6. Gitea Client (`gitea/`)
- Repository file listing and content fetch (base64 decode)
- Webhook creation for onboarding
- Label resolution and creation for issues

### 7. Issue Manager (`issues/`)
- Creates labeled Gitea issues from analysis results
- Issue body includes severity, file, line, code snippet, and PoC

## Data Flow

```
Gitea webhook
    → handlers.WebhookHandler (auth, rate limit, repo filter)
    → main.webhookProcessor (concurrency limiter)
    → analyzers.Engine.RunCAHPipeline
        → Prepare (structure + attack surface)
        → Scan (static → LLM on flagged files)
        → Validate → Dedup → Prove
    → issues.Manager.CreateIssuesFromAnalysis
    → Gitea issues with labels
```

## Module Path

```
git.commsnet.org/commstech/repository-detective
```

## Configuration

- `gitea_url`, `gitea_token`, `webhook_secret`
- `public_url` — required for webhook registration via onboarding UI
- `api_key` — protects API and onboarding endpoints
- `ai_provider`, `ai_base_url`, `ai_api_key`, `ai_model`
- `enable_security`, `enable_quality`
- `repository_include_patterns`, `repository_exclude_patterns`
- `skip_patterns`, `max_file_size`, `max_concurrent_analyses`

See [docs/ONBOARDING.md](docs/ONBOARDING.md) and [docs/AI_PROVIDERS.md](docs/AI_PROVIDERS.md).
