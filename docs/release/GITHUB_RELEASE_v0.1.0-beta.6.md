# Repository Detective v0.1.0-beta.6

**Public beta** — CI restored, webhook replay claims no longer fail closed on duplicates, and advisory AI preflight follows the operator TLS setting.

## Identity

| Field | Value |
|-------|-------|
| Version | `v0.1.0-beta.6` |
| E2E forge baseline | Gitea **1.22.3** (acceptance recorded for `v0.1.0-beta.3`; this tag inherits that baseline and does not add a new clean-install proof) |
| License | AGPL-3.0-or-later |
| Release posture | Public beta prerelease. Forge issue filing defaults **on** (`auto_create_issues: true`). Severity and confidence gates still apply. |

## What's new since v0.1.0-beta.5

### Forge and startup
- Durable issue-filing claims so concurrent workers do not open duplicate forge issues for the same fingerprint
- Webhook replay protection across process restarts
- A repeated delivery of an in-flight or already processed webhook is acknowledged without a SQLite unique-constraint error
- Startup exposes a truthful readiness route before background work finishes
- `X-API-Key` is accepted again alongside `X-Repository-Detective-API-Key`

### AI advisory
- The preflight health probe uses the same explicit insecure-TLS opt-in as the AI client, and that opt-in does not leak into later strict probes
- Advisory reviews stay off unless you enable them. In-scan LLM auditors stay off by default
- Optional first-run rule tuning applies `report_only` only after AI confirms a false positive, and it will not quiet protected secret or security rules

### Scanning
- Scheduled scans can skip work when the repository tip matches the last completed scan
- Scanner workspace skips match path segments, so a directory named like a build artifact is not dropped by accident
- Go 1.26 is the CI and container toolchain. Gitea CI runs staticcheck `v0.8.2`, which understands that toolchain
- Beta, rc, and alpha tags are published as prereleases

## Install

Use the image published with this tag, or build from the tag yourself.

```bash
docker pull git.commsnet.org/commstech/repository-detective:v0.1.0-beta.6
```

Public mirror, when the release workflow has pushed it:

```bash
docker pull ghcr.io/commstech/repository-detective:v0.1.0-beta.6
```

Point Compose at that image with `RD_IMAGE`, then set your own forge URL, token, organization, and API key. The shipped examples use placeholders. Do not copy an operator's `.env` or database into this tree.

## Onboarding

1. Copy `.env.example` to `.env` and set `REPOSITORY_DETECTIVE_API_KEY`.
2. Set `REPOSITORY_DETECTIVE_GITEA_URL` and `REPOSITORY_DETECTIVE_GITEA_TOKEN` for your Gitea or Forgejo.
3. Optional: set GitHub URL and token if you also file against GitHub.
4. Optional: set an AI base URL. Leave AI disabled to scan deterministically only.
5. Open `/onboard` after `/health` returns ready.

## Known limitations

- Remediation pull-request execution stays **disabled by default**
- LLM auditors stay **off by default**
- Cross-process issue idempotency and multi-process restart recovery remain **NOT_PROVEN**
- Image signing remains checksum-only
- Forgejo and a fresh upgrade E2E from beta.3 are **NOT_PROVEN** for this tag

## Security

Report privately. See [SECURITY.md](https://github.com/CommsTech/Repository-Detective/blob/main/SECURITY.md).

Canonical history is on Gitea. GitHub is the public mirror.
