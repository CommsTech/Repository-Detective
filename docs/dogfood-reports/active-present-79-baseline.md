# Active-present 79 burn-down baseline

Recorded: 2026-06-09
Latest commit: `d350b5f`
Latest scan: `4f8617f80f1ef1e8`

## Product repo

| Metric | Value |
|---|---:|
| Open Gitea issues | 1 (#48) |
| Active-present | 79 |
| High/critical | 0 |

### Severity

| Severity | Count |
|---|---:|
| info | 4 |
| low | 63 |
| medium | 12 |

### Top rules

| Count | Source | Rule |
|---:|---|---|
| 33 | reliability | HEALTH-IGNORED-ERROR |
| 13 | maintainability | HEALTH-MANY-PARAMS |
| 8 | performance | HEALTH-READ-ALL |
| 4 | maintainability | HEALTH-DEEP-NEST |
| 3 | tech_debt | HEALTH-DEPRECATED |
| 3 | tech_debt | HEALTH-TECH-PHRASE |
| 3 | maintainability | HEALTH-LARGE-FILE |
| 2 | tech_debt | HEALTH-COMMENT-BLOCK |
| 2 | test_gap | HEALTH-GO-NO-TEST |
| 2 | maintainability | HEALTH-LARGE-FUNC |
| 1 | performance | HEALTH-LONG-SLEEP |
| 1 | performance | HEALTH-REGEX-IN-LOOP |
| 1 | static | OPT-NESTED-LOOP |
| 1 | reliability | HEALTH-EMPTY-CATCH |
| 1 | reliability | HEALTH-HTTP-NO-TIMEOUT |
| 1 | static | REL-INTERNAL-INFRA-REF |

## Runner / remediation state

| Setting | Value |
|---|---|
| Runner delegation | disabled |
| Worker running | no |
| Remediation PR | disabled |
| Gitea Actions backend | disabled |
| All-repo scan | off |

## Stop conditions

- Do not enable runner delegation by default
- Do not create PRs or enable remediation PR
- Do not globally suppress from product-only evidence
- High/critical protected from automatic downgrade
- Target: active-present near 0 after fix + repo-scoped calibration + rescan
