# Public community beta

**Audience:** Self-hosters discovering Repository Detective via the public GitHub mirror.
**Edition:** Community (AGPL-3.0-or-later)
**Status:** Public beta — expect rough edges.
**Accepted baseline:** [`v0.1.0-beta.3`](release/ACCEPTANCE_v0.1.0-beta.3.md). Current public-release readiness is tracked in [`CURRENT_PUBLIC_RELEASE_READINESS.md`](release/CURRENT_PUBLIC_RELEASE_READINESS.md).

## Start here

| Resource | Link |
|----------|------|
| Screenshots | [assets/screenshots/README.md](assets/screenshots/README.md) |
| Demo walkthrough | [DEMO.md](DEMO.md) |
| Doctor / diagnostics | [DOCTOR.md](DOCTOR.md) · `/ui/doctor` · `/api/v1/doctor` |
| Acceptance evidence | [release/ACCEPTANCE_v0.1.0-beta.3.md](release/ACCEPTANCE_v0.1.0-beta.3.md) |
| Verify release | [VERIFY_RELEASE.md](VERIFY_RELEASE.md) |
| Known limitations | [KNOWN_LIMITATIONS.md](KNOWN_LIMITATIONS.md) |
| Bug reports | [GitHub Issues](https://github.com/<your-org>/Repository-Detective/issues/new/choose) |
| Security reports | [SECURITY.md](../SECURITY.md) |

## What you get

- Deterministic scanning first (Trivy, Grype, gitleaks, linters, …)
- Operator UI + API (`X-Repository-Detective-API-Key`)
- Gitea-first webhooks and issue filing
- Docker all-in-one compose (port **8081**)
- Optional local LLM path; AI off by default

## What to expect (honest)

| Area | Reality |
|------|---------|
| Forge | **Gitea 1.22.3** is E2E-proven; Forgejo not proven; GitHub issue filing experimental |
| Scale | Single-operator / SQLite — not multi-tenant SaaS / RBAC |
| LLM | **Off by default** |
| Remediation PRs | **Disabled by default**; Class-B sandbox **NOT_PROVEN** |
| Upgrade E2E | **NOT_PROVEN** |
| Image | Prefer `v0.1.0-beta.3` or its immutable digest |

### Recommended installation

```bash
git clone https://github.com/<your-org>/Repository-Detective.git
cd Repository-Detective
cp .env.example .env
# REPOSITORY_DETECTIVE_API_KEY required; forge vars for webhooks
docker compose up -d --build
curl -s http://127.0.0.1:8081/health
# open http://127.0.0.1:8081/onboard
```

Deeper: [QUICKSTART.md](QUICKSTART.md) · [SETUP.md](SETUP.md).

## Feedback we want most

1. Install friction
2. Scanner false positives (rule ID + fingerprint)
3. First-scan → first-triage confusion
4. Gitea webhook / issue surprises

Never paste secrets.

**Canonical forge:** [Gitea](https://git.example.com/<your-org>/Repository-Detective).
**GitHub** is a sanitized snapshot — [GITHUB_MIRROR.md](GITHUB_MIRROR.md).

## License

**AGPL-3.0-or-later** — [LICENSE](../LICENSE) · [LICENSING_STRATEGY.md](LICENSING_STRATEGY.md). Commercial: [EDITIONS.md](EDITIONS.md).
