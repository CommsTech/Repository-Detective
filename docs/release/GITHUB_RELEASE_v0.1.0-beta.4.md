# Repository Detective v0.1.0-beta.4

**Public Beta** — AI harness hardening + cookie security closeout.

## Identity

| Field | Value |
|-------|-------|
| Version | `v0.1.0-beta.4` |
| Source commit (Gitea canonical) | `63672b5a` (product tip `0c4ceed4` + release docs + Go 1.25 govulncheck image pin) |
| Container digest | `sha256:8d6f224b66870c761c4b1b3f87b74bad98dae215edaba3a381c871d7ba1264ea` |
| E2E forge | Gitea **1.22.3** (same baseline as beta.3) |
| License | AGPL-3.0-or-later |

## What's new since v0.1.0-beta.3

### AI harness hardening (ponytail-influenced)
- Cross-auditor shared cancel + abort-on-first-timeout + per-scan failure budget
- `RunAuditor` single-flight and immediate cancel respect
- Lean defaults: `auto_after_scan=false`, CAH max candidates **6**, token budget **1200**
- `ai_recommendations_value_mode=actionable_security` — prefer security/workflow findings; skip GRAPH-ORPHAN / HEALTH / LINT noise
- Preflight skip when recent `llm_auditor_timeout` / `ai_advisory_failed` counts are high or the AI endpoint probe fails
- Harness contract **`rd-cah-v3`**
- Global `ENABLE_LLM_AUDITORS=false` kill-switch still wins over deep/profile overrides; cost lockdown remains the safe default

### Learning / cost lockdown (included)
- Disposition-only FP rates; AI advisory calibrate ingest; auditor timeout learning events
- 24h auditor timeout circuit-breaker

### Security closeout
- Fixed gosec **G124** on UI API-key cookies (`ui/api_key_auth.go`, `ui/api_key_cookie.go`) via gin `SetCookie` / `SetSameSite`
- `Secure` still follows `public_url` (HTTPS → secure; HTTP homelab unlock still works)
- Product forge issues **#478** / **#479** closed as resolved-verified

## Install (binaries)

Download assets from this release, verify `SHA256SUMS`, then run the binary for your OS/arch with your existing `.env` / compose setup.

```bash
sha256sum -c SHA256SUMS
```

## Container

All-in-one image published to Gitea Package Registry and GHCR:

```bash
docker pull git.commsnet.org/commstech/repository-detective:v0.1.0-beta.4
docker pull ghcr.io/commstech/repository-detective:v0.1.0-beta.4
# digest pin (preferred)
docker pull ghcr.io/commstech/repository-detective@sha256:8d6f224b66870c761c4b1b3f87b74bad98dae215edaba3a381c871d7ba1264ea
```

Also tagged `:v0.1.0-beta.4-all-in-one`, `:all-in-one`, and `:latest` on both registries.

## Known limitations (honest)

- Same product limitations as beta.3 unless noted above
- Remediation PR execution remains **disabled by default**
- LLM auditors / auto-after-scan AI remain **off by default** (intentional cost control)
- Image signing remains **CHECKSUM_ONLY** (no cosign yet)

## Security

Report privately — see [SECURITY.md](https://github.com/CommsTech/Repository-Detective/blob/main/SECURITY.md).

## Mirror note

Canonical development history lives on Gitea. GitHub is the public mirror; tag SHAs may differ while trees match.
