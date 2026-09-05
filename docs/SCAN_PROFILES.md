# Scan profiles

Repository Detective uses four operator-facing scan profiles. Names say what they do.

| Profile | ID | What it does |
|---------|----|----------------|
| **Light** | `light` | Fast read-only scan (secrets + vulns). **No forge issue submissions.** |
| **Standard** | `standard` | Full deterministic scanners (security, Go, IaC, health, graph). **Files forge issues** when policy allows. |
| **Deep** | `deep` | Heavy workspace scan with **AI cross-checks**. Slower, highest coverage. |
| **Custom** | `custom` | Manual toggles only — no preset overrides. |

### Required vs optional scanners (RD-012 / RD-012A)

| Concept | Meaning |
|---------|---------|
| **Profile-required** | Declared by the profile; cannot be removed by disabling the scanner |
| **Operator-enabled optional** | Extra scanners turned on; for Standard/Deep these join the required set when enabled |
| **Applicability** | `NOT_APPLICABLE` (e.g. no matching manifests) — complete for required only when the tool legitimately decides N/A |
| **Disabled** | `SKIPPED_BY_POLICY` — **incomplete** when the scanner is REQUIRED |

| Profile | Always required (even if disabled) | Also required |
|---------|--------------------------------------|---------------|
| Light | `gitleaks`, `trivy` | — |
| Standard / Deep | `gitleaks`, `trivy`, `grype`, `semgrep` | Union of all currently enabled scanners |
| Custom | — | All enabled scanners; **empty enabled set is incomplete** (never silent `0/0` → `POLICY_MET`) |

Missing, disabled, failed, timed-out, or unavailable **required** analyzers produce `EVALUATION_INCOMPLETE` — never `POLICY_MET`.

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
