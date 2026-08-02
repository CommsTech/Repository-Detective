# Licensing strategy

**Status:** Community edition published under **AGPL-3.0-or-later** (see root [LICENSE](../LICENSE) and [NOTICE](../NOTICE)).  
Commercial / Enterprise remain separate paid terms.  
**Not legal advice** — consult a lawyer for distribution and dual-licensing questions.

---

## Model

**One codebase, three editions**, differentiated by **feature gates** and **license terms**:

```text
Repository Detective Community     → AGPL-3.0-or-later
Repository Detective Commercial    → paid commercial license
Repository Detective Enterprise    → paid commercial license + enterprise features
```

---

## Recommended path: AGPL Community + commercial license

### Community (AGPL-3.0-or-later)

- Source available; modify and self-host
- Network use triggers copyleft obligations for modified versions offered as a service
- Suitable for homelab, evaluators, and contributors who accept AGPL terms
- Reference: [GNU AGPL v3.0 (SPDX)](https://spdx.org/licenses/AGPL-3.0.html) · root [LICENSE](../LICENSE)

### Commercial / Enterprise (proprietary license)

- No AGPL obligations for licensees
- Private modifications allowed under contract
- Support, SLA, enterprise features
- Managed service rights by agreement

**Simple message:**

```text
Free under AGPL if you comply with its terms.
Paid if you need commercial terms, enterprise features, support, or closed modifications.
```

---

## Alternatives considered

| Option | Pros | Cons |
|--------|------|------|
| **Elastic License 2.0** | Blocks hosted reselling | Not OSI open source; adoption friction |
| **Business Source License** | Converts to OSS later | Complex; restricted period |
| **MIT (historical draft)** | Maximum adoption | No SaaS protection; weak commercial lever |

AGPL fits a **networked self-hosted security product** while preserving a commercial exception path.

---

## What stays free (Community)

Core security value must remain credible:

- Deterministic scanners
- Local pre-install safety checks
- Dashboard and repository map
- Suppressions and false-positive handling
- Evidence closure tracking
- Calibration recommendations (basic)
- Docker all-in-one

---

## What is likely paid (Commercial / Enterprise)

Monetize **scale, teams, governance, support** — not basic findings:

- RBAC and multi-user auth
- SSO / OIDC / SAML
- Teams and org model
- Audit log (advanced)
- Custom report branding and exports
- Advanced runner pools
- Enterprise policy packs
- Priority support / SLA
- Managed service rights
- HA / Postgres multi-tenant (Enterprise)

---

## Managed service rights

Operating Repository Detective as a **managed service for third parties** is reserved for **Commercial/Enterprise licensees** unless AGPL compliance is met for modified/network-deployed versions.

Community operators may run it for **their own** repos and community beta users on their infrastructure.

---

## Community contribution model

Encourage:

- GitHub/Gitea issues and feature requests
- Scanner recipes and false-positive reports
- Documentation improvements
- Repro steps for bugs

Contribution license should align with chosen OSS license (CLA or DCO — TBD with counsel).

---

## Implementation status

| Item | Status |
|------|--------|
| License file in repo | **Done** — root `LICENSE` (AGPL-3.0) + `NOTICE` |
| `edition` config key | Not implemented |
| Capability struct / gates | Not implemented |
| License key validation | Not implemented |
| UI locked-feature banners | Not implemented |
| Offline license file | Not implemented |

Future plan: see [COMMERCIAL_ENTERPRISE.md](COMMERCIAL_ENTERPRISE.md) § Implementation plan.

---

## Branding compatibility

All editions support legacy `REPOSITORY_DETECTIVE_*` env vars and `X-Repository-Detective-API-Key` alongside preferred Repository Detective names. See [BRANDING_COMPATIBILITY_AUDIT.md](BRANDING_COMPATIBILITY_AUDIT.md).

---

## Related docs

- [EDITIONS.md](EDITIONS.md)
- [COMMUNITY_EDITION.md](COMMUNITY_EDITION.md)
- [COMMERCIAL_ENTERPRISE.md](COMMERCIAL_ENTERPRISE.md)
- [MONETIZATION_READINESS.md](MONETIZATION_READINESS.md)
