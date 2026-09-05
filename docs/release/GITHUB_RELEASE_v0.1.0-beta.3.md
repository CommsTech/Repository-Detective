# Repository Detective v0.1.0-beta.3

**Public Beta** — accepted baseline.

## Identity

| Field | Value |
|-------|-------|
| Version | `v0.1.0-beta.3` |
| OCI digest | `sha256:6a615548c8a1fc2494140e73f1c3bd3f78f0ed54a7b15eaa7a1025e83e308727` |
| E2E forge | Gitea **1.22.3** only |
| Source commit (in image) | `e130bfb` |
| License | AGPL-3.0-or-later |

Same digest is published to:

- `git.commsnet.org/commstech/repository-detective`
- `ghcr.io/commstech/repository-detective` (public mirror)

## What this release is

Self-hosted, Gitea-first repository investigation: deterministic analysis, canonical finding lifecycle, owner-defined policy evaluation, and remediation **planning**. AI is optional. Remediation PR execution remains **disabled by default**.

## Proof (links)

- [Acceptance evidence](https://github.com/CommsTech/Repository-Detective/blob/main/docs/release/ACCEPTANCE_v0.1.0-beta.3.md) — clean-install + published-image E2E on this digest
- [Scanner coverage](https://github.com/CommsTech/Repository-Detective/blob/main/docs/SCANNERS.md)
- [Verify release](https://github.com/CommsTech/Repository-Detective/blob/main/docs/VERIFY_RELEASE.md) — digest + SBOM checksums
- [Container SBOM](https://github.com/CommsTech/Repository-Detective/blob/main/docs/release/sbom/) (SPDX + CycloneDX for the digest above)

## Install

```bash
docker pull ghcr.io/commstech/repository-detective@sha256:6a615548c8a1fc2494140e73f1c3bd3f78f0ed54a7b15eaa7a1025e83e308727
# or clone and use docker compose (pin RD_IMAGE to the digest above)
```

See [QUICKSTART](https://github.com/CommsTech/Repository-Detective/blob/main/docs/QUICKSTART.md).

## Known limitations (honest)

- Gitea **1.22.3** is the only E2E-proven forge/version
- Forgejo not proven; GitLab not implemented; GitHub issue-provider experimental
- Multi-user/RBAC not production-ready
- Class-B remediation sandbox **NOT_PROVEN**; remediation PR execution disabled by default
- Upgrade E2E **NOT_PROVEN**
- Image signing: **CHECKSUM_ONLY** (digest + SBOM checksums; no cosign yet)

## Security

Report privately — see [SECURITY.md](https://github.com/CommsTech/Repository-Detective/blob/main/SECURITY.md). Do not post exploit details in public issues.

## Mirror note

This GitHub tag is a **sanitized public snapshot** of the Gitea release source. Canonical development history lives on Gitea. See [GITHUB_MIRROR.md](https://github.com/CommsTech/Repository-Detective/blob/main/docs/GITHUB_MIRROR.md).
