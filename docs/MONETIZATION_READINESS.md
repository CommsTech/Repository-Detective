# Monetization readiness

**Product:** Repository Detective — Inspect. Analyze. Improve.  
**Date:** 2026-06-04

This document tracks what revenue paths are open today versus what requires future engineering.

---

## Ready now

| Offering | Mode | Notes |
|----------|------|-------|
| **Manual paid audits** | Operator-led | Pre-install audit + human triage + private disclosure package (RuView-style workflow) |
| **Single-operator managed assessments** | Homelab / dedicated host | API-key auth; operator runs scans and delivers reports |
| **Private beta with trusted users** | Gitea-first | Beta config in [BETA_READINESS.md](BETA_READINESS.md); limitations documented |

---

## Not ready

| Offering | Blocker |
|----------|---------|
| **Multi-tenant SaaS** | Auth/RBAC, tenant isolation, billing — see [AUTH_RBAC_PLAN.md](AUTH_RBAC_PLAN.md) |
| **Self-service paid onboarding** | User signup, payment, provisioning |
| **Teams with separate permissions** | Repo grants, roles — phase 3–4 of auth plan |
| **Unsupervised customer remediation PRs** | Policy + legal + RBAC; intentionally disabled in beta |

---

## Next unlock for monetization

**Auth/RBAC** (Commercial) and **edition feature gates** (Commercial/Enterprise) — not more scanners.

| Tier | Path |
|------|------|
| Community | Default today — API key, AGPL proposed |
| Commercial | Auth/RBAC + license — [COMMERCIAL_ENTERPRISE.md](COMMERCIAL_ENTERPRISE.md) |
| Enterprise | SSO, HA, policy packs — same binary, expanded capabilities |

See [EDITIONS.md](EDITIONS.md) and [LICENSING_STRATEGY.md](LICENSING_STRATEGY.md).

Recommended sequence:

1. Auth/RBAC design (complete — [AUTH_RBAC_PLAN.md](AUTH_RBAC_PLAN.md))
2. Single-admin local login + sessions
3. API token management for automation
4. Roles + repo permissions for multi-analyst paid audits
5. Org/team model + audit log
6. Billing integration (SaaS only)

---

## Pricing model placeholders (design only)

| Tier | Audience | Includes |
|------|----------|------------|
| **Private beta** | Trusted operators | Single tenant, API key or local login, manual support |
| **Managed audit** | Per-engagement | Pre-install report, human review, optional private disclosure |
| **Team** | Small AppSec team | Multi-user RBAC, repo grants, audit log |
| **SaaS** | Self-service | OIDC, billing, tenant isolation, SLA |

No billing or **license enforcement** code exists. Do not implement until Auth/RBAC phases 1–5 are stable and beta week completes.

---

## Related docs

- [BETA_READINESS.md](BETA_READINESS.md)
- [AUTH_RBAC_PLAN.md](AUTH_RBAC_PLAN.md)
- [EDITIONS.md](EDITIONS.md)
- [LICENSING_STRATEGY.md](LICENSING_STRATEGY.md)
- [COMMUNITY_EDITION.md](COMMUNITY_EDITION.md)
- [COMMERCIAL_ENTERPRISE.md](COMMERCIAL_ENTERPRISE.md)
- [POLICY.md](POLICY.md)
- [PRIVACY.md](PRIVACY.md)
