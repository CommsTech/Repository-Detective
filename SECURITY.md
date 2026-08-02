# Security policy

## Supported versions

| Stream | Support |
|--------|---------|
| `main` (Community public beta) | Best-effort fixes |
| Private / commercial builds | Per agreement |

## Reporting a vulnerability

**Do not** file a public GitHub/Gitea issue with exploit details, secrets, or customer code.

Prefer one of:

1. Gitea private/security triage channel for the project maintainers (if you have forge access)
2. Email the maintainers listed on the org profile for **CommsTech** / Repository Detective
3. A high-level public issue titled “Security contact requested” with **no** PoC payload — we will follow up privately

Include:

- Affected version / image tag / commit SHA
- Impact summary (auth bypass, secret leak, RCE, SSRF, etc.)
- Minimal reproduction **without** real secrets
- Whether you are already running a patched workaround

## Operator hardening

See [docs/SECURITY_HARDENING.md](docs/SECURITY_HARDENING.md) and [docs/SECURITY_MODEL.md](docs/SECURITY_MODEL.md).

## Scope notes

- Scanner false positives / noisy rules → use issue templates (`scanner_false_positive`), not this policy
- Dependency CVEs in scanned *target* repos are product findings, not RD vulnerabilities
- Untrusted-code scanners run with a minimal subprocess environment — report env/secret leakage if you observe it
