# Repository Detective Commercial & Enterprise

**Status:** Planning only — no license enforcement or edition gates in code yet.

---

## Commercial Edition

**Goal:** Small teams and companies running serious self-hosted Repository Detective.

### Unlocks (over Community)

```text
Higher or unlimited connected repos
Multi-user local auth
Roles and RBAC
Teams
API token management
Audit log
Advanced scheduled scans
Runner delegation
Notifications
Remediation PR approvals workflow
Custom report branding
Exportable assessment reports
Advanced calibration recommendations
Backup/restore operator tooling (documented workflows)
Priority support
Commercial-use license (non-AGPL terms)
```

### Typical buyers

- AppSec team on private Gitea/Forgejo
- Consultancies doing **manual paid audits** on customer infra (self-hosted)
- Companies needing RBAC before wider rollout

---

## Enterprise Edition

**Goal:** Regulated environments and larger organizations.

### Unlocks (over Commercial)

```text
SSO / OIDC / SAML
SCIM / user provisioning (later)
Org and team isolation policies
Advanced audit log and export
Policy packs and approval workflows
Multiple Gitea/Forgejo instances
GitHub / GitLab connected adapters (when shipped)
External runner pools
High availability / Postgres backend
Tenant or customer separation
Compliance-oriented reports
Custom suppression governance
Offline / license-server activation
Support SLA
Professional services
```

---

## One binary / feature gates

Editions share one build artifact. Capabilities loaded at startup:

```go
type Edition string

const (
    EditionCommunity  Edition = "community"
    EditionCommercial Edition = "commercial"
    EditionEnterprise Edition = "enterprise"
)

type Capabilities struct {
    MaxRepos                 int
    MultiUserAuth            bool
    RBAC                     bool
    OIDC                     bool
    AuditLog                 bool
    RunnerDelegationAdvanced bool
    CustomBranding           bool
    ReportExports            bool
    MultiForge               bool
    AdvancedCalibration      bool
    EnterprisePolicies       bool
}
```

Example config (future):

```yaml
edition: community
license_key: ""
```

Valid Commercial/Enterprise license → expanded `Capabilities`. UI shows locked features with badges:

```text
SSO/OIDC        — Enterprise
RBAC            — Commercial
Custom reports  — Commercial
Policy packs    — Enterprise
```

**Community must not break** when license is absent — only gates premium surfaces.

---

## Future implementation plan

| Phase | Deliverable |
|-------|-------------|
| 1 | `internal/license` package — edition enum, capability defaults |
| 2 | License file + env key parsing (stub validation) |
| 3 | Handler-level capability checks (return 402/403 with edition hint) |
| 4 | UI locked-feature banners |
| 5 | Offline signed license files |
| 6 | License server integration (Enterprise) |
| 7 | Tests — community defaults, commercial unlock matrix |

**Do not implement until private beta week completes** unless blocking paid pilot.

---

## Monetization alignment

| Offering | Edition |
|----------|---------|
| Homelab / OSS adoption | Community |
| Self-hosted team license | Commercial |
| Regulated enterprise | Enterprise |
| Managed service for clients | Commercial+ contract |

See [MONETIZATION_READINESS.md](MONETIZATION_READINESS.md).

---

## Auth/RBAC dependency

Commercial requires [AUTH_RBAC_PLAN.md](AUTH_RBAC_PLAN.md) phases 1–5. Enterprise adds OIDC and advanced audit on top.

---

## Branding compatibility

All editions honor legacy Repository-Detective env vars, API headers, labels, and fingerprints. Public docs prefer Repository Detective naming — see [BRANDING_COMPATIBILITY_AUDIT.md](BRANDING_COMPATIBILITY_AUDIT.md).

---

## Related docs

- [EDITIONS.md](EDITIONS.md)
- [LICENSING_STRATEGY.md](LICENSING_STRATEGY.md)
- [COMMUNITY_EDITION.md](COMMUNITY_EDITION.md)
- [AUTH_RBAC_PLAN.md](AUTH_RBAC_PLAN.md)
