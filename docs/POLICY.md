# Per-repo scan policy (Phase 8)

Repository Detective merges **global config** with **per-repo settings** from the local database on every scan. Global `config.yaml` / `REPOSITORY_DETECTIVE_*` (or legacy `REPOSITORY_DETECTIVE_*`) env vars remain the fallback when a repo has no overrides.

> **Naming:** See [NAMING.md](NAMING.md).

## Resolution order

```text
global config snapshot
→ global profile defaults (when repo scan_profile is NULL; see SCAN_PROFILES.md)
→ repo profile defaults (when repo scan_profile is set and not custom)
→ repo_settings DB explicit overrides (non-null fields only)
→ scan execution
```

When `database_enabled=false`, only global config and global profile apply.

See [SCAN_PROFILES.md](SCAN_PROFILES.md) for built-in profile names and defaults.

## Policy levels (Observe / Warn / Enforce)

Operator-facing modes map to stored `policy_level` values:

| Mode | Stored value | Scans | Persist findings | Forge issues | Commit status |
|------|--------------|-------|------------------|--------------|---------------|
| **Observe** | `monitor_only` | Yes | Yes | No | Non-blocking (`OBSERVATION_ONLY`) |
| **Warn** | `issue_only` | Yes | Yes | Yes (per issue policy) | Surfaces `ACTION_REQUIRED` without blocking merge |
| **Enforce** | `gate_pr` | Yes | Yes | Yes | May fail status when branch protection requires the context |

Legacy / reserved: `suggest_fix`, `auto_pr_with_approval`, `auto_pr_low_risk` (status like `gate_pr` for now).

### Policy outcomes (never “safe” / “secure”)

| Outcome | Meaning |
|---------|---------|
| `POLICY_MET` | Required analyzers completed; configured policy conditions were not violated |
| `ACTION_REQUIRED` | One or more configured policy conditions were violated |
| `EVALUATION_INCOMPLETE` | Required scanners failed, timed out, or were unavailable |
| `OBSERVATION_ONLY` | Observe mode — findings may exist; Repository Detective does not block |

Mark the status context as required in Gitea branch protection only when you intentionally want Enforce to participate in merge gates. Outcomes describe **owner policy compliance**, not security assurance.

## Issue policies

| Policy | Behavior |
|--------|----------|
| `off` | No Gitea issue create/update; findings still persist locally |
| `fingerprint` | Fingerprint + semantic dedup (default legacy behavior) |
| `all` | Create/update all gate-passing findings; exact fingerprint dedup still applies |

## Severity and confidence gates

Severity order: `critical > high > medium > low > info`

- Findings below `confidence_gate` do not create issues or fail status.
- Findings below `severity_gate` do not create issues or fail status.
- All findings are still stored in scan results and the local DB at original severity/confidence.

## AI policy

| Value | Behavior |
|-------|----------|
| `allowed` | LLM stages run when global AI is configured and `enable_llm_auditors=true` with depth ≥ 3 |
| `disabled` | No LLM calls (PREPARE attack surface, SCAN auditors, VALIDATE debate, PROVE PoC) |

Repo `ai_policy=disabled` wins over global AI enablement. Repo `ai_policy=allowed` does not enable AI if no provider is configured globally.

## Scanner and workspace overrides

Per-repo toggles override global scanner enablement:

- `enable_trivy`, `enable_grype`, `enable_gitleaks`, `enable_semgrep`, `enable_govulncheck`, `enable_gosec`, `enable_staticcheck`, `enable_hadolint`, `enable_checkov`, `enable_linters`
- `workspace_mode`: `api`, `archive`, or `auto`
- `analysis_depth`: `1`, `2`, or `3`

## Health checks (Phase 10)

Deterministic repository health checks run when **global** `enable_health_checks: true` and effective `analysis_depth >= 2`. They do not require LLM and do not call AI.

| Category | Typical severity | Notes |
|----------|------------------|-------|
| `tech_debt` | low–medium | TODO/FIXME/HACK markers |
| `reliability` | medium | Ignored errors, missing HTTP timeouts |
| `maintainability` / `code_quality` | low–medium | Large files/functions, deep nesting |
| `test_gap` | medium | Missing `_test.go` or test scripts |
| `performance` | low–medium | Regex-in-loop, long sleeps |
| `ai_generated_risk` | low–medium | Optional; off by default |

Health findings use the same severity/confidence gates as security findings for issue creation. All findings persist regardless of gates.

Global toggles: `enable_tech_debt_checks`, `enable_reliability_checks`, `enable_maintainability_checks`, `enable_test_gap_checks`, `enable_performance_checks`, `enable_ai_risk_checks` (default `false`).

Per-repo overrides use the same field names in `repo_settings` (nullable = inherit global). Configure via `PUT /api/v1/repos/{id}/settings` or `/ui/repos/:id/settings`.

Pre-install audits use global pre-install/health config only — per-repo health overrides apply to connected repository scans.

See [HEALTH_CHECKS.md](HEALTH_CHECKS.md).

## Code graph / repository map (Phase 11B)

Deterministic code graphs run when effective `analysis_depth >= 2` and effective `enable_code_graph: true`. Graph settings resolve from global defaults plus nullable per-repo overrides on `repo_settings`.

| Field | Range / type |
|-------|----------------|
| `enable_code_graph` | bool |
| `graph_max_nodes` | 100–50000 |
| `graph_max_edges` | 100–200000 |
| `graph_timeout_seconds` | 5–1800 |
| `graph_include_functions` | bool |
| `graph_include_findings` | bool |

Scan policy snapshots include resolved graph settings. Pre-install audits use **global** graph config only.

Disconnected-code graph findings use cautious wording and may false-positive on dynamic/reflection-heavy code — see [CODE_GRAPH.md](CODE_GRAPH.md).

## Runner delegation (Phase 12)

When `runner_delegation_enabled: true` and global `runner_mode` is not `core`, scheduled and manual full-repo scans may create `runner_jobs` instead of running the in-process analyzer.

| `runner_policy` | Behavior |
|-----------------|----------|
| `core` | Always in-process |
| `gitea_actions` | Queue runner job when global mode allows |
| `auto` | Queue runner job; fall back to core on capacity/errors |

Webhook push/PR scans remain on core in Phase 12. Runners never create issues or commit statuses — see [RUNNERS.md](RUNNERS.md).

## Notifications (Phase 15)

Per-repo notification overrides (`notifications_enabled`, `notification_min_severity`, `notification_events`, `notification_cooldown_seconds`) inherit global defaults when NULL. A repo can disable notifications even when global notifications are enabled. Channel credentials remain global — see [NOTIFICATIONS.md](NOTIFICATIONS.md).

Notification fields do **not** switch the scan profile to `custom` when changed alone.

## Remediation planner (Phase 16)

See [REMEDIATION.md](REMEDIATION.md). Generates structured fix plans only by default. `remediation_policy` on repo settings remains stored; planner uses global toggles and severity/confidence gates.

## Safe remediation PRs (Phase 17)

See [REMEDIATION_PRS.md](REMEDIATION_PRS.md). **Disabled by default.** When enabled, only approved low-risk plans with deterministic patchers may open a branch + PR on connected Gitea repos. No auto-merge, no issue close, no secret or dependency auto-fix.

### Owned-repo safe fix rollout (beta)

Start fixing **owned/connected repos only** through the approved loop:

```text
finding → issue → remediation plan → approval → safe PR → tests → merge → rescan → verified closure
```

PRs are created only when **all** of the following hold:

| Gate | Requirement |
|------|-------------|
| Repository | Connected/owned Gitea repo |
| Plan | `approved`, `safe_for_auto_pr=true`, `requires_human_review=false` |
| Risk | `regression_risk=low`, `fix_complexity=small` |
| Validation | At least one allowlisted command passes |
| Scope | Not secret, graph/orphan deletion, architecture rewiring, dependency major upgrade, or high-risk gosec/checkov/trivy |

**Third-party pre-install audits:** never auto-file issues or PRs — generate copy/paste reports and disclosure drafts only.

UI copy: *Repository Detective creates PRs only for approved low-risk plans. It never auto-merges.*

## Evidence-based closure (Phase 18)

See [EVIDENCE_CLOSURE.md](EVIDENCE_CLOSURE.md). **Enabled by default** for evidence tracking; **auto-close disabled by default**. Closes or marks resolved only after PR merge + rescan + fingerprint gone + scanner success.

Finding auto-resolution after a fix push is **not** implied by webhook scans alone — see [FINDING_RESOLUTION_SEMANTICS.md](FINDING_RESOLUTION_SEMANTICS.md).

## Issue reconciliation (Phase 19)

See [ISSUE_RECONCILIATION.md](ISSUE_RECONCILIATION.md). Inspects already-filed Gitea issues against latest scans. Never deletes issues; closes only when `issue_reconciliation_close_verified: true` and verification evidence exists.

## Deterministic calibration (Phase 19)

See [CALIBRATION.md](CALIBRATION.md). Local rule statistics and recommendations from suppressions, false positives, and verified fixes. **No auto-apply by default.**

## AI token efficiency (Phase 19)

See [AI_TOKEN_EFFICIENCY.md](AI_TOKEN_EFFICIENCY.md). **AI startup chat tests disabled by default**; use metadata-only or manual test endpoints.

## Example: deterministic strict repo

```yaml
policy_level: gate_pr
workspace_mode: auto
analysis_depth: 2
ai_policy: disabled
enable_trivy: true
enable_grype: true
enable_gitleaks: true
enable_semgrep: true
enable_linters: true
issue_policy: fingerprint
severity_gate: medium
confidence_gate: 0.85
```

Set via operator UI or `PUT /api/v1/repos/{id}/settings`.

## Scan audit trail

Each completed scan stores an `effective_settings` snapshot in scan summary JSON so you can see which policy was active for that run.

## Rollback

1. Clear per-repo overrides in UI/API (set fields to inherit).
2. Or set `database_enabled=false` to revert to global-only behavior.
3. No schema migration required.

## Limitations (Phase 8)

Pre-install audit mode (Phase 9) uses separate tables and does **not** use per-repo policy from `repo_settings`. Third-party audits always run with deterministic scanners only, `issue_policy=off`, and no Gitea issue creation.

- `remediation_policy` is stored but not enforced on scan paths.
- `runner_policy` is **enforced** for scheduled and manual full scans when global runner delegation is enabled (Phase 12). See [RUNNERS.md](RUNNERS.md).
- Manual analyze of unknown repos uses global config only.
- No `.repository-detective.yaml` in-repo config yet.

## Private beta defaults (issue closeout calibration)

Evidence from the 253-issue closeout sprint (see `docs/dogfood-reports/issue-closeout-calibration-report.md`):

| Default | Rationale |
|---------|-----------|
| Global profile `beta_standard` | Graph + QUAL-DEBUG findings report-only; reduces Gitea noise |
| Graph rules report-only | 82 open graph issues were legacy noise; keep on dashboard |
| `standard_deterministic` issue min severity **high** for lint/health | Ruff F401 and low health findings deferred — not auto-closed |
| Critical/high security unchanged | SEC-* and dependency findings remain actionable |
| Global suppressions for graph/debug rules | See `docs/dogfood-reports/closeout-suppressions.sql` |

**Do not** suppress critical/high security findings without documented false-positive review.
