# Security policy

## Supported versions (public beta)

| Stream | Support |
|--------|---------|
| `v0.1.0-beta.3` (accepted public-beta baseline) | Best-effort fixes |
| Newer tagged `v*` betas | Best-effort for the latest announced tag |
| `main` | Best-effort; may move ahead of the proven image |
| Private / commercial builds | Per agreement |

Verify images by **digest**, not tag alone — [docs/VERIFY_RELEASE.md](docs/VERIFY_RELEASE.md).

## Reporting a vulnerability in Repository Detective

**Do not** file a normal public GitHub or Gitea issue with exploit details, secrets, customer code, or a working PoC.

### Preferred — private disclosure

1. **GitHub private vulnerability reporting** (recommended for public reporters):  
   [Report a vulnerability](https://github.com/CommsTech/Repository-Detective/security/advisories/new)
2. If you already have access to the canonical forge, contact maintainers through a **private** Gitea channel (do not attach exploit payloads to public issues).

### If private reporting is unavailable

Open a **high-level** public issue titled **“Security contact requested”** with:

- Affected version / image digest / commit SHA
- Impact category only (for example: auth bypass, secret leak, RCE, SSRF)
- **No** PoC, payload, secret material, or customer repository contents

Maintainers will follow up privately. Response timing is best-effort during public beta (no paid SLA unless under a commercial agreement).

### What to include (private report)

- Affected version / image digest / commit SHA
- Impact summary
- Minimal reproduction **without** real secrets
- Whether you already have a workaround

## Out of scope for this policy

| Topic | Where to go |
|-------|-------------|
| Scanner false positives / noisy rules | Public issue templates |
| Dependency CVEs in *scanned target* repositories | Normal product findings — not RD vulnerabilities |
| Installation / docs / UX bugs | [GitHub Issues](https://github.com/CommsTech/Repository-Detective/issues/new/choose) |

## Operator hardening

See [docs/SECURITY_HARDENING.md](docs/SECURITY_HARDENING.md) and [docs/SECURITY_MODEL.md](docs/SECURITY_MODEL.md).

## Scope notes

- Untrusted-code scanners run with a minimal subprocess environment — report env/secret leakage if you observe it.
- Do not assume automated analysis proves a repository is “safe” or “secure”; policy outcomes describe compliance with an owner-defined policy only.
- Class-B remediation execution is **disabled by default** and sandboxing is **NOT_PROVEN** (RD-008B Option C).
