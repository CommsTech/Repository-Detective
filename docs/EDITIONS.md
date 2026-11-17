# Repository Detective editions

**One product. One binary. Feature gates by edition.**

Repository Detective ships as a single codebase and Docker image. Edition determines which capabilities are enabled at runtime.

**Commercial MVP status (RD-COMMERCIAL-001):** Community vs Commercial gates are **implemented** for connected-repo limits, multi-user RBAC roles, operator audit trail UI, and license-key unlock. Enterprise remains reserved (treated like Commercial capability set until a separate Enterprise license path exists). SSO/SAML/HA are **not** in Commercial v1.

---

## Editions

| Edition | Audience | Goal |
|---------|----------|------|
| **Community** | Homelab, OSS adopters, small teams | Adoption, trust, feedback, Gitea visibility |
| **Commercial** | Self-hosted companies | Scale, teams, RBAC, support, commercial license |
| **Enterprise** | Regulated / large orgs | SSO, governance, HA, compliance, SLA *(roadmap)* |

---

## Feature matrix

| Feature | Community | Commercial | Enterprise |
|---------|:---------:|:----------:|:----------:|
| Gitea / Forgejo connected repo scans | ✅ | ✅ | ✅ |
| Deterministic scanners | ✅ | ✅ | ✅ |
| Finding Detail 2.0 operator brief | ✅ | ✅ | ✅ |
| Noise reduction dashboard proof | ✅ | ✅ | ✅ |
| Suppressions / calibration | ✅ basic | ✅ advanced reporting | ✅ governed *(roadmap)* |
| Issue creation (connected Gitea) | ✅ | ✅ | ✅ |
| Remediation planner | ✅ | ✅ | ✅ |
| Safe remediation PRs | limited / manual | ✅ when enabled | ✅ with approvals |
| Evidence closure | ✅ | ✅ | ✅ |
| Single API-key auth | ✅ | ✅ legacy | ✅ legacy |
| Multi-user login (`auth_mode=local`) | single operator | ✅ | ✅ |
| RBAC (owner / admin / security / developer / viewer) | ❌ | ✅ | ✅ |
| Operator audit trail UI | basic / local writes | ✅ | ✅ advanced *(roadmap)* |
| Connected repo limit | **10** (configurable) | unlimited | unlimited |
| Custom branding / reports | ❌ | ✅ gated | ✅ |
| OIDC / SAML | ❌ | ❌ | ✅ *(not built)* |
| Postgres / HA | ❌ | optional *(not built)* | ✅ *(not built)* |
| Support / SLA | community | paid support | SLA |
| License | AGPL | Commercial license + key | Commercial + enterprise terms |

---

## Community limits (enforced)

```text
~10 connected repos (community_max_repos, default 10)
Single operator (no multi-user create UI)
No Commercial audit trail UI
No branded management reports
```

**Never limited:** security findings, scanners, local false-positive suppressions, pre-install safety checks.

---

## Runtime model

```yaml
edition: community          # community | commercial | enterprise
license_key: ""             # required to unlock commercial/enterprise
community_max_repos: 10
```

Environment:

```bash
REPOSITORY_DETECTIVE_EDITION=commercial
REPOSITORY_DETECTIVE_LICENSE_KEY=your-license-key
REPOSITORY_DETECTIVE_COMMUNITY_MAX_REPOS=10
```

Empty `license_key` always forces Community limits even if `edition` says commercial.

UI shows the edition chip in the sidebar. Doctor reports the active edition.

See [LICENSING_STRATEGY.md](LICENSING_STRATEGY.md) and [COMMERCIAL_ENTERPRISE.md](COMMERCIAL_ENTERPRISE.md).

---

## Compatibility

Legacy **Repository-Detective** naming remains supported in all editions for env vars, API headers, labels, and fingerprints. See [BRANDING_MIGRATION.md](BRANDING_MIGRATION.md).
