# Documentation audit

**Date:** 2026-06-04  
**Phase:** Test and documentation hardening (private beta)

Legend: **Present** · **Current** (accurate for beta) · **Examples** · **Safety** · **Needs update**

---

## Minimum beta documentation set

| Doc | Present | Current | Examples | Safety | Needs update |
|-----|:-------:|:-------:|:--------:|:------:|:------------:|
| [QUICKSTART.md](QUICKSTART.md) | ✅ | ✅ | ✅ | ✅ | No |
| [DOCKER.md](DOCKER.md) | ✅ | ✅ | ✅ | ✅ | Minor — rebuild after pull |
| [CONFIGURATION.md](CONFIGURATION.md) | ✅ | ✅ | ✅ | ✅ | No |
| [SCANNERS.md](SCANNERS.md) | ✅ | ✅ | ✅ | ✅ | No |
| [POLICY.md](POLICY.md) | ✅ | ✅ | ⚠️ | ✅ | Add beta_standard pointer |
| [SCAN_PROFILES.md](SCAN_PROFILES.md) | ✅ | ✅ | ✅ | ✅ | No |
| [PREINSTALL_AUDIT.md](PREINSTALL_AUDIT.md) | ✅ | ⚠️ | ✅ | ✅ | Default now `false` in beta |
| [REMEDIATION.md](REMEDIATION.md) | ✅ | ✅ | ⚠️ | ✅ | No |
| [EVIDENCE_CLOSURE.md](EVIDENCE_CLOSURE.md) | ✅ | ✅ | ⚠️ | ✅ | No |
| [BACKUP_RESTORE.md](BACKUP_RESTORE.md) | ✅ | ✅ | ✅ | ✅ | No |
| [TROUBLESHOOTING.md](TROUBLESHOOTING.md) | ✅ | ✅ | ✅ | ✅ | Expanded this phase |
| [BETA_READINESS.md](BETA_READINESS.md) | ✅ | ✅ | ✅ | ✅ | Link test matrix |

---

## Core operator docs

| Doc | Present | Current | Examples | Safety | Needs update |
|-----|:-------:|:-------:|:--------:|:------:|:------------:|
| [README.md](../README.md) | ✅ | ✅ | ✅ | ⚠️ | License TBD (AGPL proposed) |
| [docs/README.md](README.md) | ✅ | ⚠️ | ✅ | — | Add QUICKSTART, TEST_MATRIX links |
| [ONBOARDING.md](ONBOARDING.md) | ✅ | ✅ | ✅ | ✅ | No |
| [DEPLOYMENT.md](DEPLOYMENT.md) | ✅ | ✅ | ✅ | ✅ | New index this phase |
| [SETUP.md](SETUP.md) | ✅ | ⚠️ | ✅ | ✅ | Prefer REPOSITORY_DETECTIVE_* in examples |
| [../DEPLOYMENT.md](../DEPLOYMENT.md) | ✅ | ✅ | ✅ | ✅ | Root quick deploy |
| [UI.md](UI.md) | ✅ | ✅ | ⚠️ | ⚠️ | API key in URL homelab risk |
| [DATABASE.md](DATABASE.md) | ✅ | ✅ | ⚠️ | ✅ | repository-detective.db name documented |
| [RUNNERS.md](RUNNERS.md) | ✅ | ✅ | ✅ | ✅ | Off by default beta |
| [NOTIFICATIONS.md](NOTIFICATIONS.md) | ✅ | ✅ | ✅ | ✅ | Preferred header updated |
| [REMEDIATION_PRS.md](REMEDIATION_PRS.md) | ✅ | ✅ | ✅ | ✅ | Off by default |
| [ISSUE_RECONCILIATION.md](ISSUE_RECONCILIATION.md) | ✅ | ✅ | ⚠️ | ✅ | No |
| [CALIBRATION.md](CALIBRATION.md) | ✅ | ✅ | ⚠️ | ✅ | No |
| [QDRANT.md](QDRANT.md) | ✅ | ✅ | ✅ | ✅ | Disabled by default |
| [PRIVACY.md](PRIVACY.md) | ✅ | ✅ | ⚠️ | ✅ | Also PRIVACY_AND_DATA_PROTECTION.md |

---

## Planning / beta docs

| Doc | Present | Current | Examples | Safety | Needs update |
|-----|:-------:|:-------:|:--------:|:------:|:------------:|
| [AUTH_RBAC_PLAN.md](AUTH_RBAC_PLAN.md) | ✅ | ✅ | ✅ | ✅ | Design only |
| [EDITIONS.md](EDITIONS.md) | ✅ | ✅ | — | — | No enforcement yet |
| [LICENSING_STRATEGY.md](LICENSING_STRATEGY.md) | ✅ | ✅ | — | — | Legal review pending |
| [TEST_MATRIX.md](TEST_MATRIX.md) | ✅ | ✅ | ✅ | ✅ | New this phase |
| [BETA_SMOKE_TEST.md](BETA_SMOKE_TEST.md) | ✅ | ✅ | ✅ | ✅ | New this phase |
| [RELEASE_NOTES_0.1.0_BETA.md](RELEASE_NOTES_0.1.0_BETA.md) | ✅ | ✅ | ✅ | ✅ | New this phase |
| [KNOWN_LIMITATIONS.md](KNOWN_LIMITATIONS.md) | ✅ | ⚠️ | — | ✅ | Cross-link beta docs |

---

## Gaps / follow-ups (non-blocking)

| Item | Action |
|------|--------|
| `SETUP.md` still shows `REPOSITORY_DETECTIVE_*` first in places | Update to preferred env prefix |
| Single `PRIVACY.md` vs `PRIVACY_AND_DATA_PROTECTION.md` | Cross-link; merge later |
| VPAT / full accessibility | [ACCESSIBILITY.md](ACCESSIBILITY.md) checklist only |
| GitHub scanning doc | [GITHUB_SCANNING.md](GITHUB_SCANNING.md) — not beta-critical |

---

## Operator path (documented end-to-end)

```text
QUICKSTART → CONFIGURATION → DEPLOYMENT/DOCKER → operator-smoke-test.sh
→ BETA_SMOKE_TEST → BACKUP_RESTORE → TROUBLESHOOTING
```

---

## Related

- [BETA_READINESS.md](BETA_READINESS.md)
- [TEST_MATRIX.md](TEST_MATRIX.md)
