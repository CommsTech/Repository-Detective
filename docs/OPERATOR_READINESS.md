# Operator readiness checklist

Repository Detective — **Inspect. Analyze. Improve.**

Use this checklist before your first real deployment (Track A: dogfooding on your own Gitea repos).

## Docker image targets

Repository Detective publishes three images from one `Dockerfile` (see [DOCKER.md](DOCKER.md)):

| Image | When to use |
|-------|-------------|
| `repository-detective:all-in-one` | Homelab / single host — core + scanners (default in root `docker-compose.yml`) |
| `repository-detective:core` | Split deploy — scanners only on runners |
| `repository-detective:runner` | Gitea Actions workers — `repository-detective-runner` + toolchain |

Persistent data: mount host `./data` → `/app/data`. Config: `./config` → `/app/config/config.yaml` (read-only). **Never** bake `.env` into images.

```bash
docker build --target all-in-one -t repository-detective:all-in-one .
docker compose up -d --build
./scripts/docker-build-verify.sh   # optional CI smoke
```

## Required binaries by feature

| Feature | Binary | Notes |
|---------|--------|-------|
| All scans | **git** | Clone and diff; required for connected repos and remediation PRs |
| Dependency CVEs | **trivy** | When `enable_trivy: true` |
| Dependency CVEs (alt) | **grype** | When `enable_grype: true` |
| Secret detection | **gitleaks** | When `enable_gitleaks: true` |
| SAST | **semgrep** | When `enable_semgrep: true` |
| Go module CVEs | **govulncheck** | When `enable_govulncheck: true` |
| Go security | **gosec** | When `enable_gosec: true` |
| Go static analysis | **staticcheck** | When `enable_staticcheck: true`; also used by safe remediation PR patchers |
| Dockerfile lint | **hadolint** | When `enable_hadolint: true`; also used by remediation PR patchers |
| IaC security | **checkov** | When `enable_checkov: true` |

Repository Detective **does not fail startup** when optional scanner binaries are missing. Check `GET /api/v1/status` or the dashboard **Scanner binaries** table for configured vs available tools.

## Optional binaries

| Binary | Use |
|--------|-----|
| golangci-lint | General Go linting when `enable_linters: true` |
| ruff | Python linting |
| shellcheck | Shell script linting |
| staticcheck | Already listed; optional unless enabled or used for PR validation |

## Recommended runner binaries

When using [RUNNERS.md](RUNNERS.md) delegation, use the **`repository-detective:runner`** image or install the same scanner set on runner hosts. Runners need:

- `repository-detective-runner` (included in the runner image)
- Scanner binaries matching the scan profile assigned to repos they serve (bundled in `repository-detective:runner` / `all-in-one`)
- Network egress to Gitea and the Repository Detective control plane (not to arbitrary operator networks unless configured)

**core-only** deployments must delegate scans to runners — the core image does not ship Trivy/Semgrep/etc.

## Required Gitea configuration

| Setting | Purpose |
|---------|---------|
| `gitea_url` | Base URL of your Gitea instance |
| `gitea_token` | Personal access token with repo read (+ write for issues/PRs when remediation enabled) |
| `webhook_secret` | HMAC secret for inbound webhooks |
| `public_url` | **Strongly recommended** — external URL for webhooks, UI links, runner callbacks |

Webhook events: `push`, `pull_request` (see `config.yaml`).

For remediation PRs the token needs permission to create branches and pull requests on target repos.

## Required runner configuration

When `runner_delegation_enabled: true`:

| Setting | Purpose |
|---------|---------|
| `runner_shared_secret` | HMAC for runner callback auth |
| `public_url` | Runners call back to `{public_url}/api/v1/runner/*` |
| Per-repo `runner_policy` | `local`, `runner`, or `auto` |

See [RUNNERS.md](RUNNERS.md).

## SQLite backup guidance

1. **Location:** `database_path` (default `./data/repository-detective.db` — legacy path name; data is Repository Detective state).
2. **When to backup:** Before upgrades, before enabling remediation PRs on production repos, and on a regular schedule (daily for active homelabs).
3. **How:** Stop the process or use SQLite online backup:
   ```bash
   sqlite3 /path/to/repository-detective.db ".backup '/path/to/backup-$(date +%F).db'"
   ```
4. **Restore:** Stop Repository Detective, replace the DB file, restart. Migrations run automatically on startup.
5. **Permissions:** Restrict file mode to the service user (`chmod 600`).

See [DATABASE.md](DATABASE.md).

## Notification setup

| Setting | Notes |
|---------|-------|
| `notifications_enabled: true` | Master toggle |
| `notification_webhook_url` | Outbound webhook (Slack-compatible JSON) |
| `notification_min_severity` | Filter noise |
| `notification_events` | Include closure events: `fix_pr_merged`, `closure_verified`, `closure_blocked`, `remediation_still_present` |

Payloads are **redacted** — no raw secrets, tokens, or full diffs. See [NOTIFICATIONS.md](NOTIFICATIONS.md).

Repository Detective does **not** send email.

## Remediation PR safety settings

Recommended for first deployment:

```yaml
remediation_planner_enabled: true
remediation_pr_enabled: true          # enable only after reviewing allowlists
remediation_pr_require_approval: true
remediation_pr_max_files_changed: 3
remediation_pr_max_diff_lines: 100
remediation_pr_branch_prefix: repository-detective/fix
```

See [REMEDIATION_PRS.md](REMEDIATION_PRS.md). No auto-merge, no protected-branch push, no secret auto-fix.

## Closure settings

Recommended defaults (already the product default):

```yaml
evidence_closure_enabled: true
evidence_closure_close_issues: false   # comment + label only until you trust the loop
evidence_closure_comment: true
evidence_closure_require_scanner_success: true
```

Closure requires: **PR merged + post-merge scan + fingerprint absent + original scanner succeeded**. See [EVIDENCE_CLOSURE.md](EVIDENCE_CLOSURE.md).

## Recommended first repo onboarding

1. Deploy with `scan_profile: standard_deterministic` (or `homelab-minimal` — see [examples/](examples/)).
2. Confirm `GET /api/v1/status` — database healthy, required tools available.
3. Connect **one non-critical repo** with webhooks enabled.
4. Run a manual scan; confirm issues created with fingerprints.
5. Enable remediation planner; generate and **manually approve** one low-risk plan.
6. Enable remediation PRs; create one PR; **merge manually** in Gitea.
7. Trigger rescan (push or scheduled); confirm `resolved-verified` label/comment.
8. Expand to more repos after one full loop succeeds.

See [ONBOARDING.md](ONBOARDING.md).

## Safe default profile recommendation

For homelab / first dogfood:

| Profile | When |
|---------|------|
| **`standard_deterministic`** | Default for connected repos — Trivy + linters, LLM auditors optional |
| **`homelab-minimal`** | Resource-constrained host; fewer scanners |
| **`strict-security-gate`** | After baseline works; enable gitleaks/semgrep/gosec |
| **`preinstall_cautious`** | Third-party repo audit only |

Copy-paste examples: [examples/deterministic-standard.yaml](examples/deterministic-standard.yaml).

## Pre-flight commands

```bash
curl -s http://localhost:8080/health | jq .
curl -s -H "X-Repository-Detective-API-Key: $REPOSITORY_DETECTIVE_API_KEY" http://localhost:8080/api/v1/status | jq .
curl -s http://localhost:8080/api/v1/about | jq .
```

## Go / no-go

**Go** when:

- Database healthy, Gitea webhook delivers events
- Required scanner binaries available for your profile
- One test repo completes the full safe loop manually
- Backups configured

**No-go** when:

- Missing `public_url` while using runners or external webhooks
- Remediation PR enabled without approval gate
- Closure enabled with `evidence_closure_require_scanner_success: false` without explicit operator acceptance
