# Repository Detective editions

**One product. One binary. Feature gates by edition.**

Repository Detective ships as a single codebase and Docker image. Edition determines which capabilities are enabled at runtime.

---

## Editions

| Edition | Audience | Goal |
|---------|----------|------|
| **Community** | Homelab, OSS adopters, small teams | Adoption, trust, feedback, Gitea visibility |
| **Commercial** | Self-hosted companies | Scale, teams, RBAC, support, commercial license |
| **Enterprise** | Regulated / large orgs | SSO, governance, HA, compliance, SLA |

---

## Feature matrix

| Feature | Community | Commercial | Enterprise |
|---------|:---------:|:----------:|:----------:|
| Gitea connected repo scans | ✅ | ✅ | ✅ |
| Deterministic scanners | ✅ | ✅ | ✅ |
| Pre-install audit | ✅ limited | ✅ | ✅ |
| Repository Map | ✅ | ✅ | ✅ |
| Suppressions / calibration | ✅ basic | ✅ advanced | ✅ governed |
| Issue creation (connected Gitea) | ✅ | ✅ | ✅ |
| Remediation planner | ✅ | ✅ | ✅ |
| Safe remediation PRs | limited / manual | ✅ | ✅ with approvals |
| Evidence closure | ✅ | ✅ | ✅ |
| Single API-key auth | ✅ | ✅ legacy | ✅ legacy |
| Multi-user login | ❌ | ✅ | ✅ |
| RBAC | ❌ | ✅ | ✅ |
| Teams / orgs | ❌ | ✅ | ✅ |
| OIDC / SAML | ❌ | ❌ / OIDC optional | ✅ |
| Audit log | basic / local | ✅ | ✅ advanced |
| Runner delegation | limited | ✅ | ✅ advanced pools |
| Notifications | basic | ✅ | ✅ |
| Custom branding / reports | ❌ | ✅ | ✅ |
| Multiple forge connections | Gitea only | Gitea / Forgejo | GitHub / GitLab / Gitea |
| Postgres / HA | ❌ | optional | ✅ |
| Support / SLA | community | paid support | SLA |
| Managed service rights | ❌ | by contract | by contract |

---

## Community limits (proposed)

Good limits — preserve credibility:

```text
~10 connected repos (configurable)
Limited schedules / concurrency
No teams / RBAC / SSO
No multi-tenant
No hosted support SLA
No customer PDF/export branding
No commercial managed-service rights without license
```

Bad limits — **never**:

```text
Hiding security findings
Blocking scanner results
Blocking local pre-install safety checks
Blocking false-positive suppressions
```

---

## Runtime model (future)

```yaml
edition: community          # community | commercial | enterprise
license_key: ""             # empty = community defaults
```

Startup loads license file, env key, or defaults to Community. UI shows locked features with edition badges (not implemented yet).

See [LICENSING_STRATEGY.md](LICENSING_STRATEGY.md) and implementation plan in [COMMERCIAL_ENTERPRISE.md](COMMERCIAL_ENTERPRISE.md).

---

## Compatibility

Legacy **Repository-Detective** naming remains supported in all editions for env vars, API headers, labels, and fingerprints. See [BRANDING_MIGRATION.md](BRANDING_MIGRATION.md).

---

## Related docs

- [COMMUNITY_EDITION.md](COMMUNITY_EDITION.md)
- [COMMERCIAL_ENTERPRISE.md](COMMERCIAL_ENTERPRISE.md)
- [LICENSING_STRATEGY.md](LICENSING_STRATEGY.md)
- [MONETIZATION_READINESS.md](MONETIZATION_READINESS.md)
- [AUTH_RBAC_PLAN.md](AUTH_RBAC_PLAN.md)
