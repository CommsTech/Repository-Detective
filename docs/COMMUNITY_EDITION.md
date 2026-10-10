# Repository Detective Community Edition

**Purpose:** Drive adoption, trust, issue reports, and feature requests on Gitea/GitHub — without crippling core security value.

Community Edition is the **default** when no commercial license is configured.

---

## Included

```text
Gitea connected repo scanning
Manual and scheduled scans (limited concurrency)
Deterministic scanners (Trivy, Grype, gitleaks, semgrep, Go tools, hadolint, checkov)
Pre-install audit (on-demand / manual)
Repository Map and dashboard
Single admin / API-key mode
Local SQLite storage
Suppression and false-positive handling
Calibration recommendations (basic)
Issue creation in connected Gitea repos
Remediation planner
Evidence closure tracking
Docker all-in-one image
```

---

## Limits (proposed — not enforced yet)

Scale and convenience caps only:

```text
~10 connected repos (example default)
Limited schedule frequency / concurrent scans
No multi-user RBAC
No SSO / OIDC
No multi-tenant isolation
Community support only (issues/docs)
No customer-ready PDF/export branding
No commercial managed-service rights
```

---

## Must remain free (non-negotiable)

```text
Core scanner results — never paywalled
Pre-install local safety checks
Suppressions for false positives
Evidence closure visibility
Local dashboard access
```

---

## Auth model (current + future)

**Today:** global API key (`REPOSITORY_DETECTIVE_API_KEY`; legacy `REPOSITORY_DETECTIVE_API_KEY`).

**After Auth/RBAC ships:** Community stays single-operator or optional single local admin; multi-user RBAC moves to Commercial.

Preferred API header: `X-Repository-Detective-API-Key`. Legacy `X-Repository-Detective-API-Key` accepted.

---

## Private beta

Community Edition is sufficient for **homelab and public community beta** on self-hosted Gitea (GitHub forge optional). See [PUBLIC_BETA.md](PUBLIC_BETA.md) and [BETA_READINESS.md](BETA_READINESS.md).

---

## License

AGPL-3.0-or-later — root [LICENSE](../LICENSE) and [NOTICE](../NOTICE). Strategy notes: [LICENSING_STRATEGY.md](LICENSING_STRATEGY.md).

---

## Related docs

- [EDITIONS.md](EDITIONS.md)
- [LICENSING_STRATEGY.md](LICENSING_STRATEGY.md)
- [BRANDING_MIGRATION.md](BRANDING_MIGRATION.md)
