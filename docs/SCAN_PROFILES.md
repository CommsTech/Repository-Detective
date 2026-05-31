# Scan profiles (Phase 14)

Repository Detective **scan profiles** replace a wall of per-scanner toggles with named presets. Profiles are **deterministic-first**; AI remains advisory unless explicitly enabled.

See also: [POLICY.md](POLICY.md), [SCANNERS.md](SCANNERS.md), [UI.md](UI.md).

## Profile names

| Profile | Purpose |
|---------|---------|
| `fast` | Quick feedback, low noise |
| `standard_deterministic` | Default useful deterministic scan |
| `strict_security` | Strong PR / security gate |
| `maintainer_deep` | Scheduled deep repo maintenance |
| `preinstall_cautious` | Third-party repo trust assessment |
| `custom` | Manual per-toggle control |

## Global config

```yaml
scan_profile: standard_deterministic
```

Environment (preferred / legacy):

```text
REPOSITORY_DETECTIVE_SCAN_PROFILE=standard_deterministic
BUGBOT_SCAN_PROFILE=standard_deterministic
```

**Default:** `custom` in code/viper — preserves legacy behavior for existing deployments that rely on individual scanner toggles in `config.yaml`.

## Per-repo setting

Nullable DB column `scan_profile`:

- `NULL` — inherit global profile name (defaults apply at resolve time)
- `custom` — use explicit repo toggles only; no profile override layer
- any other allowed name — apply that profile’s defaults before repo explicit overrides

## Resolution order

```text
global config snapshot
→ global profile defaults (when repo scan_profile is NULL and global profile ≠ custom)
→ repo profile defaults (when repo scan_profile is set and ≠ custom)
→ repo explicit overrides (non-null stored fields always win)
```

Explicit repo overrides set `profile_modified: true` in API/UI. Changing any advanced toggle via UI/API auto-switches the repo to `custom`.

Selecting a non-custom profile without advanced fields **clears** stored repo overrides so the profile applies cleanly.

## Profile defaults (summary)

### `fast`

- `analysis_depth: 2`, AI disabled
- gitleaks + trivy on; semgrep, grype, Go, IaC, linters off
- minimal health (master switch on, sub-checks off), graph off

### `standard_deterministic`

- `analysis_depth: 2`, AI disabled
- trivy, grype, gitleaks, semgrep, Go, IaC, linters on
- health + graph on

### `strict_security`

- `analysis_depth: 2`, AI allowed but LLM auditors off by default
- all deterministic scanners on
- `policy_level: gate_pr`, `severity_gate: medium`, `confidence_gate: 0.85`

### `maintainer_deep`

- `workspace_mode: auto`, `analysis_depth: 3`
- all deterministic scanners, full health, test gaps + performance
- graph with functions, `runner_policy: auto`

### `preinstall_cautious`

- AI off, `issue_policy: off`, `policy_level: monitor_only`
- gitleaks, trivy, semgrep, Go, IaC on; grype/linters off
- health standard, graph on

### `custom`

- No profile layer; global config + repo explicit toggles only

## API / scan snapshot fields

Settings and scan summaries include:

```json
{
  "scan_profile": "standard_deterministic",
  "profile_modified": false,
  "profile_source": "global",
  "effective_profile_summary": {
    "security_scanners": true,
    "go_scanners": true,
    "iac_scanners": true,
    "health_checks": true,
    "code_graph": true,
    "ai": "disabled",
    "runner_eligible": false
  }
}
```

`profile_source`: `repo`, `global`, or `default` (custom / inherited baseline).

## When to use each profile

| Scenario | Recommended profile |
|----------|---------------------|
| PR webhook quick feedback | `fast` |
| Default connected repos | `standard_deterministic` |
| Security gate on merge | `strict_security` |
| Nightly cron / runner jobs | `maintainer_deep` |
| Pre-install audit alignment | `preinstall_cautious` (audit path keeps its own cautious runtime) |
| Fine-grained operator control | `custom` |

## Runner recommendations

| Profile | Runner |
|---------|--------|
| `fast`, `standard_deterministic` | `core` (default) |
| `strict_security` | `core` or `gitea_actions` for isolated PR scans |
| `maintainer_deep` | `auto` or `gitea_actions` for full workspace |
| `preinstall_cautious` | core read-only audit path |

## Rollback

Set `scan_profile: custom` globally and per-repo (or NULL repo profile). Deploy previous binary if needed; migration v9 is additive (`scan_profile` column nullable).
