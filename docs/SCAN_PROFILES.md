# Scan profiles

Repository Detective uses four operator-facing scan profiles. Names say what they do.

| Profile | ID | What it does |
|---------|----|----------------|
| **Light** | `light` | Fast read-only scan (secrets + vulns). **No forge issue submissions.** |
| **Standard** | `standard` | Full deterministic scanners (security, Go, IaC, health, graph). **Files forge issues** when policy allows. |
| **Deep** | `deep` | Heavy workspace scan with **AI cross-checks**. Slower, highest coverage. |
| **Custom** | `custom` | Manual toggles only — no preset overrides. |

Legacy IDs still work and map automatically:

| Legacy | Maps to |
|--------|---------|
| `fast`, `preinstall_cautious` | `light` |
| `beta_standard`, `standard_deterministic`, `homelab_infra`, `strict_security` | `standard` |
| `maintainer_deep` | `deep` |

## Config

```yaml
scan_profile: standard
```

```text
REPOSITORY_DETECTIVE_SCAN_PROFILE=standard
REPOSITORY_DETECTIVE_SCAN_PROFILE=standard
```

## Resolution order

```text
global config snapshot
→ global profile defaults (when profile ≠ custom)
→ repo profile defaults (when repo profile set and ≠ custom)
→ repo explicit overrides (non-null stored fields always win)
```

Changing advanced toggles via UI/API switches the repo to `custom`.

See also: [POLICY.md](POLICY.md), [SCANNERS.md](SCANNERS.md), [UI.md](UI.md).
