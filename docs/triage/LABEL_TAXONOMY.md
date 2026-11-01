# Label taxonomy

Recommended labels for Repository Detective product issues during private beta.

Create these in Gitea as needed; YAML issue templates pre-select subsets.

## Type

| Label | Use |
|-------|-----|
| `type/bug` | Incorrect behavior, regression |
| `type/feature` | Enhancement request |
| `type/docs` | Documentation gap |
| `type/false-positive` | Scanner noise / miscalibration |
| `type/missed-detection` | Should have been found |
| `type/security-review` | Triage security finding accuracy |
| `type/operator-task` | Internal operator work |

## Area

| Label | Use |
|-------|-----|
| `area/ui` | Operator UI |
| `area/docs` | Docs/wiki/onboarding |
| `area/scanner` | Static/secret/dependency scanners |
| `area/sbom` | SBOM generation/inventory |
| `area/container-scan` | Image scanning |
| `area/preinstall` | Pre-install audit |
| `area/ai-recommendations` | Advisory AI layer |
| `area/runner` | Runner delegation |
| `area/issue-filing` | Forge issue creation/sync |
| `area/graph` | Repository map / graph heuristics |
| `area/learning` | Calibration / learning loop |

## Severity

`severity/critical` · `severity/high` · `severity/medium` · `severity/low` · `severity/info`

## Confidence (review)

`confidence/high` · `confidence/medium` · `confidence/low`

## Status

| Label | Meaning |
|-------|---------|
| `status/needs-triage` | New, unassigned |
| `status/needs-repro` | Missing reproduction |
| `status/accepted` | Valid, queued |
| `status/blocked` | External dependency |
| `status/fixed` | Resolved in main |
| `status/wontfix` | Intentional / out of scope |
| `status/duplicate` | Duplicate of existing issue |

## Beta

| Label | Use |
|-------|-----|
| `beta/feedback` | Private beta tester feedback |
| `beta/private` | Internal-only context |

## Legacy compatibility

Existing issues may use `repository-detective/*` labels. New templates prefer the taxonomy above where Gitea labels exist.
