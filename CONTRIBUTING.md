# Contributing to Repository Detective

Thanks for helping.

| Host | Role | URL |
|------|------|-----|
| **Gitea** | Canonical CI, wiki, maintainer development | https://git.commsnet.org/commstech/Repository-Detective |
| **GitHub** | Sanitized public mirror + **public issue feedback** | https://github.com/CommsTech/Repository-Detective |

## Where to report

| Kind | Where |
|------|--------|
| Bug, install, scanner, feature | **[GitHub Issues](https://github.com/CommsTech/Repository-Detective/issues/new/choose)** |
| Security vulnerability in RD | **[SECURITY.md](SECURITY.md)** — private advisory; never paste exploit detail publicly |

## Pull requests — where to send them

- **Preferred for maintainers:** open PRs against **Gitea** (canonical history + CI).
- **External contributors without Gitea access:** open a **GitHub Issue** with a clear proposal / patch description, or a GitHub PR against the public mirror for discussion. Maintainers will port accepted changes to Gitea.
- Do **not** assume a GitHub PR is automatically merged into the canonical tree — GitHub is a snapshot mirror ([GITHUB_MIRROR.md](docs/GITHUB_MIRROR.md)).

If you need direct Gitea write access for ongoing contribution, ask maintainers via a public GitHub issue titled “Contributor access requested” (no secrets).

## Before you open an issue

1. Note your image tag / digest (prefer `v0.1.0-beta.3` or the published digest).
2. Never paste API keys, forge tokens, `.env`, or real secrets.
3. Prefer: scan ID, finding fingerprint, rule ID, severity, minimal repro.
4. `curl -s http://127.0.0.1:8081/health`

## Local development

```bash
git clone https://github.com/CommsTech/Repository-Detective.git
cd Repository-Detective
cp .env.example .env
cp config/config.yaml.example config/config.yaml
docker compose pull && docker compose up -d
```

Tests (match `go.mod`; exclude local `e2e/results` clones):

```bash
docker run --rm -v "$PWD":/src -w /src golang:1.25-bookworm \
  sh -c 'pkgs=$(go list ./... | grep -v /e2e/results/); go test -count=1 $pkgs'
```

## Coding standards

- Prefer small, focused changes with a clear “why”
- Do not commit `.env`, `config/config.yaml`, `data/*.db`, or credentials
- Update docs when operator-facing behavior changes
- Do not expand Class-B remediation execution or claim sandboxing that is NOT_PROVEN
- By contributing, you agree your changes are licensed under **AGPL-3.0-or-later**

## Security-sensitive contributions

Auth, webhook HMAC, privacy egress, reconcile/close paths, and scanner fail-closed behavior need tests and honest documentation updates (DOC_TRUTH / SECURITY_MODEL). See [SECURITY.md](SECURITY.md).

## Code of conduct

Be respectful. Treat findings and operator data carefully.
