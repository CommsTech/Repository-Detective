# Dogfooding Repository Detective

Repository Detective — **Inspect. Analyze. Improve.**

This guide prepares you to scan **this repository first** before onboarding wider Gitea repos. Track A is docs-and-config only: no new features, no auto-close, no broad auto-remediation.

After dogfooding, fill in [DOGFOOD_REPORT_TEMPLATE.md](DOGFOOD_REPORT_TEMPLATE.md) to decide go/no-go for wider rollout.

## Prerequisites

### Required

| Item | Notes |
|------|-------|
| **Go 1.22+** | Build and run from source, or use your deployment image |
| **git** | On PATH — clones and diffs |
| **Gitea** | Instance where this repo is hosted; token with repo + issue read (write for issue creation) |
| **SQLite** | Enabled via `database_enabled: true` (default) |
| **API key** | Set `api_key` / `REPOSITORY_DETECTIVE_API_KEY` — never commit real values |

### Recommended scanner binaries

For the dogfood profile (`standard_deterministic`), install what you can. Missing optional tools do **not** block startup; they appear as missing in status.

| Binary | Purpose |
|--------|---------|
| trivy, grype | Dependency CVEs |
| gitleaks, semgrep | Secrets / SAST (strict profile) |
| govulncheck, gosec, staticcheck | Go scanner trio |
| hadolint, checkov | IaC / Dockerfiles (if Dockerfiles or IaC present) |
| golangci-lint | General Go linting |

See [OPERATOR_READINESS.md](OPERATOR_READINESS.md) for the full binary matrix.

### Network

- `public_url` must be reachable **from Gitea** for webhooks (push/PR triggers).
- For first manual scan only, webhooks are optional if you use `POST /api/v1/analyze`.

---

## Recommended config

Copy keys from [examples/dogfood-repository-detective.yaml](examples/dogfood-repository-detective.yaml) into your `config.yaml` or environment.

### Safe first-run settings

```yaml
scan_profile: standard_deterministic
database_enabled: true
ui_enabled: true

evidence_closure_enabled: true
evidence_closure_close_issues: false

remediation_planner_enabled: true
remediation_pr_enabled: false

notifications_enabled: false
runner_delegation_enabled: false

enable_llm_auditors: false
enable_ai_risk_checks: false
```

> **Note:** The valid profile name is `standard_deterministic` (not `deterministic-standard`). The example file [deterministic-standard.yaml](examples/deterministic-standard.yaml) uses the same profile with a descriptive filename.

**Keep remediation PRs disabled** (`remediation_pr_enabled: false`) until you have:

1. Completed at least one full manual scan of this repo
2. Reviewed findings and false positives
3. Run the planner and manually approved at least one plan
4. Filled a dogfood report with go/no-go for PR automation

AI is intentionally off for dogfood so results are reproducible and reviewable. Enable LLM auditors only after deterministic findings look sane.

---

## Start Repository Detective

Build and run (adjust paths and secrets):

```bash
go build -o bin/repository-detective .
./bin/repository-detective -config config/config.yaml
```

Or use Docker Compose / your existing deployment. Set at minimum:

```text
REPOSITORY_DETECTIVE_GITEA_URL
REPOSITORY_DETECTIVE_GITEA_TOKEN
REPOSITORY_DETECTIVE_API_KEY
REPOSITORY_DETECTIVE_PUBLIC_URL
```

Legacy `REPOSITORY_DETECTIVE_*` env vars remain supported.

---

## Verify `/health`

No authentication required:

```bash
curl -s http://localhost:8080/health | jq .
```

Expect when ready:

```json
{
  "status": "healthy",
  "service": "repository-detective",
  "product_name": "Repository Detective",
  "version": "...",
  "public_url_configured": true,
  "features": { "database_healthy": true, ... },
  "tools_summary": { "configured_count": N, "available_count": M, "missing": [...] }
}
```

If `status` is `starting`, wait for migrations and component init. If `503`, check logs and database path permissions.

---

## Verify `/api/v1/status`

Requires API key header (prefer new name; legacy alias still works):

```bash
export KEY="your-api-key"

curl -s -H "X-Repository-Detective-API-Key: $KEY" \
  http://localhost:8080/api/v1/status | jq .
```

Confirm before onboarding:

| Field | Expected for dogfood |
|-------|----------------------|
| `features.database_enabled` / `database_healthy` | `true` |
| `features.ui_enabled` | `true` |
| `features.remediation_planner_enabled` | `true` |
| `features.remediation_pr_enabled` | **`false`** |
| `features.evidence_closure_enabled` | `true` |
| `features.runner_delegation_enabled` | `false` |
| `features.scan_profile` | `standard_deterministic` |
| `tools[]` | Required tools `available: true` where configured |

Also verify `/api/v1/about` (no auth):

```bash
curl -s http://localhost:8080/api/v1/about | jq .
```

---

## Status API examples (no CLI)

Repository Detective has no operator CLI. Use curl:

```bash
export KEY="your-api-key"
export BASE="http://localhost:8080"

curl -s -H "X-Repository-Detective-API-Key: $KEY" "$BASE/api/v1/status" | jq .
curl -s -H "X-Repository-Detective-API-Key: $KEY" "$BASE/api/v1/repos" | jq .
curl -s -H "X-Repository-Detective-API-Key: $KEY" "$BASE/api/v1/dashboard/summary" | jq .
```

Legacy header (equivalent):

```bash
curl -s -H "X-Repository-Detective-API-Key: $REPOSITORY_DETECTIVE_API_KEY" "$BASE/api/v1/status" | jq .
```

---

## Onboard the Repository Detective repo

### Option A — Onboarding wizard

1. Open `{public_url}/onboard`
2. Enter Gitea URL, token, public URL, webhook secret, API key
3. Select the repository that hosts **Repository-Detective** (or your fork name)
4. Register webhooks (push + pull request)

See [ONBOARDING.md](ONBOARDING.md).

### Option B — Manual webhook

In Gitea → Repository → Settings → Webhooks:

- URL: `{public_url}/webhook`
- Secret: matches `webhook_secret` in config
- Events: Push, Pull request

Ensure the repo appears in the UI under **Repositories** after the first webhook or manual scan.

### Per-repo profile

In UI → Repository → Settings, set **Scan profile** to:

- **`standard_deterministic`** — recommended first dogfood pass, or
- **`strict_security`** — second pass after baseline looks good

Save settings. Confirm effective summary badges show Security, Go, IaC, Health, Graph, and AI **disabled**.

---

## Run a manual full scan

### API trigger

Replace `owner`, `repo`, and `ref` with your Gitea coordinates:

```bash
curl -X POST "$BASE/api/v1/analyze" \
  -H "X-Repository-Detective-API-Key: $KEY" \
  -H "Content-Type: application/json" \
  -d '{"owner":"your-org","repo":"Repository-Detective","ref":"main"}'
```

### Webhook trigger

Push an empty commit or open a PR to trigger a scan if webhooks are registered.

### Watch progress

- UI: **Dashboard** → recent scans; **Repositories** → repo → scan list
- API: `GET /api/v1/scans/:scan_id`, `GET /api/v1/scans/:scan_id/scanner-results`
- Logs: look for `[SCANNER:...]` lines

---

## Inspect findings

1. UI → **Findings** or repo detail → findings list
2. Sort by severity; open **critical** and **high** first
3. Note **source** (trivy, staticcheck, health, etc.) and **fingerprint**
4. API: `GET /api/v1/findings?severity=high&limit=50`
5. Check **scanner failures** on scan detail before trusting absence of findings

For health / tech-debt items see [HEALTH_CHECKS.md](HEALTH_CHECKS.md). For code graph output see scan/repo graph pages ([CODE_GRAPH.md](CODE_GRAPH.md)).

---

## Approve a remediation plan

Only after findings look accurate:

1. Open a finding detail page in the UI
2. Click **Generate / regenerate plan**
3. Review plan summary, risk, complexity, blocked reasons
4. If appropriate, click **Approve plan**

Planner creates **plans only** — no code changes until you explicitly attempt a PR (still disabled in dogfood phase).

API:

```bash
curl -X POST -H "X-Repository-Detective-API-Key: $KEY" \
  "$BASE/api/v1/findings/{id}/remediation/generate"

curl -X POST -H "X-Repository-Detective-API-Key: $KEY" \
  "$BASE/api/v1/remediation/{plan_id}/approve"
```

---

## Safe remediation PR (only if eligible — after dogfood gate)

**Not for first run.** When `remediation_pr_enabled: true` and a plan is approved + eligible:

1. UI shows eligibility checklist on finding detail
2. **Create remediation PR** creates branch + PR only — no merge, no auto-close
3. Merge manually in Gitea, then trigger rescan

During Track A dogfood, keep `remediation_pr_enabled: false` and document eligible plans in the dogfood report instead.

---

## Verify closure evidence

With `evidence_closure_enabled: true` and `evidence_closure_close_issues: false`:

1. After PR merge + rescan, open finding → **Closure evidence**
2. Expect: merge commit, verification scan ID, scanner status, fingerprint absent
3. Verified closure adds comment + `resolved-verified` label — does **not** auto-close issues by default

Manual verify:

```bash
curl -H "X-Repository-Detective-API-Key: $KEY" \
  "$BASE/api/v1/findings/{id}/closure-evidence"

curl -X POST -H "X-Repository-Detective-API-Key: $KEY" \
  "$BASE/api/v1/findings/{id}/verify-closure"
```

See [EVIDENCE_CLOSURE.md](EVIDENCE_CLOSURE.md).

---

## Self-scan checklist

Follow this exact flow for the first dogfood pass:

```text
1. Start Repository Detective with dogfood config.
2. Open /health and confirm ready.
3. Open /api/v1/status and confirm DB, scanner tools, UI, graph, planner, closure.
4. Add Repository Detective repo as connected Gitea repo.
5. Set repo profile to standard_deterministic or strict_security.
6. Run manual full scan.
7. Review scanner failures first.
8. Review critical/high findings.
9. Review generated issues.
10. Review remediation plans.
11. Do not enable remediation PR until findings look accurate.
12. Export or save dogfood report.
13. Fix config/docs/code issues manually if needed.
14. Run second scan.
15. Only then consider enabling remediation_pr_enabled.
```

Copy [DOGFOOD_REPORT_TEMPLATE.md](DOGFOOD_REPORT_TEMPLATE.md) and fill it in after step 12.

---

## Rollback

| Goal | Action |
|------|--------|
| Stop scanning | Disable repo in UI settings or remove webhook |
| Disable issues | Set `auto_create_issues: false` and restart |
| Disable planner | `remediation_planner_enabled: false` |
| Disable closure | `evidence_closure_enabled: false` |
| Full reset | Stop process, restore SQLite backup ([DATABASE.md](DATABASE.md)), restart |
| Undo bad config | Revert `config.yaml` to dogfood example; restart |

No automatic rollback of Gitea issues or PRs — triage manually in Gitea.

---

## Related docs

| Doc | Purpose |
|-----|---------|
| [OPERATOR_READINESS.md](OPERATOR_READINESS.md) | Pre-flight binary and config checklist |
| [DOGFOOD_REPORT_TEMPLATE.md](DOGFOOD_REPORT_TEMPLATE.md) | Report template after first scan |
| [examples/dogfood-repository-detective.yaml](examples/dogfood-repository-detective.yaml) | Copy-paste dogfood config |
| [TESTING.md](TESTING.md) | Unit tests and analyze API |

---

## After dogfood

The next “feature” should come **only after** the dogfood report shows where real friction is. Wider Gitea onboarding follows a **go** decision in the report template.
