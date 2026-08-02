# Public community beta

**Audience:** Self-hosters and invitees moving from private trials onto the public GitHub mirror.  
**Edition:** Repository Detective Community (AGPL-3.0-or-later)  
**Status:** Public beta — expect rough edges; please file issues.

## What you get

- Deterministic scanning first (Trivy, Grype, gitleaks, linters, …)
- Operator UI + API (`X-Repository-Detective-API-Key`)
- Gitea-first webhooks and issue filing
- Docker all-in-one / minimal compose
- Docs + wiki source under `docs/` / `docs/wiki/`

## What to expect (honest)

| Area | Reality |
|------|---------|
| Forge | **Gitea is first-class**; GitHub issue filing is present but not as proven |
| Scale | Single-operator / SQLite — not multi-tenant SaaS |
| LLM | **Off by default** (`enable_llm_auditors: false`) |
| Image | Prefer a build that matches `go.mod` (Go **1.25**); older toolchains degrade some SBOM/linter paths |
| Backlog | Busy fleets accumulate findings — use Learning / suppressions / focus export |

Full constraints: [KNOWN_LIMITATIONS.md](KNOWN_LIMITATIONS.md).

## 15-minute try

```bash
git clone https://github.com/CommsTech/Repository-Detective.git
cd Repository-Detective
cp .env.example .env
cp config/config.yaml.example config/config.yaml
# Set REPOSITORY_DETECTIVE_API_KEY (+ Gitea URL/token if you want webhooks)
docker compose -f docker-compose.minimal.yml up -d --build
curl -s http://localhost:8080/health
open http://localhost:8080/onboard
```

Deeper install: [QUICKSTART.md](QUICKSTART.md) · [SETUP.md](SETUP.md) · smoke: [BETA_SMOKE_TEST.md](BETA_SMOKE_TEST.md).

## Feedback we want most

1. Install friction (compose, tools missing, docs gaps)
2. Scanner false positives / parser failures (rule ID + fingerprint)
3. UI/workflow confusion on first scan → first triage
4. Gitea webhook / issue-filing surprises

Never paste secrets. Prefer scan ID + fingerprint.

- GitHub: https://github.com/CommsTech/Repository-Detective/issues  
- Gitea (canonical templates): https://git.commsnet.org/commstech/Repository-Detective/issues/new  

## License

Community builds are **AGPL-3.0-or-later** — see root [LICENSE](../LICENSE) and [LICENSING_STRATEGY.md](LICENSING_STRATEGY.md). Commercial terms: [EDITIONS.md](EDITIONS.md).
