# Security policy

## Supported versions

| Stream | Support |
|--------|---------|
| `main` (Community public beta) | Best-effort fixes |
| Tagged releases (`v*`) | Best-effort for the latest tag |
| Private / commercial builds | Per agreement |

## Reporting a vulnerability in Repository Detective

**Do not** file a normal public GitHub or Gitea issue with exploit details, secrets, customer code, or a working PoC.

### Preferred — private disclosure

1. **GitHub private vulnerability reporting** (recommended for public reporters):  
   [Report a vulnerability](https://github.com/CommsTech/Repository-Detective/security/advisories/new)
2. If you already have access to the canonical forge, contact project maintainers through a **private** Gitea channel (do not attach exploit payloads to public issues).

### If private reporting is unavailable

Open a **high-level** public issue titled **“Security contact requested”** with:

- Affected version / image tag / commit SHA
- Impact category only (for example: auth bypass, secret leak, RCE, SSRF)
- **No** PoC, payload, secret material, or customer repository contents

Maintainers will follow up privately.

### What to include (in the private report)

- Affected version / image tag / commit SHA
- Impact summary
- Minimal reproduction **without** real secrets
- Whether you already have a workaround

## Out of scope for this policy

| Topic | Where to go |
|-------|-------------|
| Scanner false positives / noisy rules | Public issue templates (`scanner_false_positive`, `scanner_problem`) |
| Dependency CVEs in *scanned target* repositories | Normal product findings — not RD vulnerabilities |
| Installation / docs / UX bugs | [GitHub Issues](https://github.com/CommsTech/Repository-Detective/issues/new/choose) |

## Operator hardening

See [docs/SECURITY_HARDENING.md](docs/SECURITY_HARDENING.md) and [docs/SECURITY_MODEL.md](docs/SECURITY_MODEL.md).

## Scope notes

- Untrusted-code scanners run with a minimal subprocess environment — report env/secret leakage if you observe it.
- Do not assume automated analysis proves a repository is “safe” or “secure”; policy outcomes describe compliance with an owner-defined policy only.
