# Security model

Repository Detective — **Inspect. Analyze. Improve.**

Trust boundaries, data flows, and honesty about what is PROVEN vs NOT_PROVEN.
See also [SECURITY_HARDENING.md](SECURITY_HARDENING.md), [PRIVACY_MODES.md](PRIVACY_MODES.md), [AUTH_LOCAL.md](AUTH_LOCAL.md).

## Classifications

| Label | Meaning |
|-------|---------|
| **PROVEN** | Enforced in code with tests or repeated operational evidence |
| **PARTIAL** | Present for some paths; gaps remain |
| **NOT_PROVEN** | Documented intent only, or untested |
| **NOT_IMPLEMENTED** | Not built |

Proof levels for capabilities: `CODE_PRESENT` / `WIRED` / `UNIT_TESTED` / `INTEGRATION_TESTED` / `E2E_PROVEN`.

## Trust zones

```text
[Operator browser / API clients]
        |
[Repository Detective control plane]  ← API key / optional local session
        |
   +----+----+------------------+
   |         |                  |
[SQLite] [Scanner subprocesses] [Optional AI endpoint]
   |         |                  |
 local DB  untrusted repo     LOCAL or EXTERNAL
             content              (privacy_mode)
        |
[Configured forge Gitea/Forgejo]  ← issues, PR summaries, commit status
        |
[Optional runners / notifications / MCP host]
```

## Privacy modes (RD-007)

| Mode | Contract | Proof |
|------|----------|-------|
| `local_only` | Blocks EXTERNAL AI + EXTERNAL notification channels; fails closed when invokable | CODE_PRESENT + WIRED + UNIT_TESTED |
| `hybrid` (default) | External AI/notify allowed when configured; disclosed in status | WIRED + UNIT_TESTED |
| `external_ai_enabled` | External AI intentional | WIRED |

**LOCAL_ONLY does not mean “local LLM only.”** If `GITEA_URL` points at a remote forge, issue bodies and PR summaries still egress there by design. Use a local forge for a fully air-gapped loop.

Endpoint locality uses IP classification (loopback, RFC1918, ULA, link-local, CGNAT) and DNS resolution for hostnames. Provider name alone (`ollama`) is **never** sufficient.

## Authentication (RD-010)

| Surface | Mechanism | Status |
|---------|-----------|--------|
| `/api/v1/*` | API key **header** (preferred) | PROVEN / UNIT_TESTED |
| Query `?api_key=` | Compatibility; warn by default; reject optional | PARTIAL — `.env.example` recommends reject |
| `/ui/*` `api_key_only` | API key (cookie hop after unlock) — **runtime default** | PROVEN |
| `/ui/*` `local` | Session cookie + CSRF + bootstrap — **recommended new install** | PROVEN (UNIT/INTEGRATION) |
| Webhooks | HMAC secret | PROVEN |
| Runners | Shared secret HMAC | PROVEN |

**Recommended new install:** `auth_mode=local` + `session_secret`. Runtime default remains `api_key_only` so existing installs are not silently flipped.

## Untrusted repository execution (RD-008)

### Class A — analysis/parsing tools (threat notes)

| Tool | Passive? | Side effects / risk notes | Isolation today |
|------|---------|---------------------------|-----------------|
| Gitleaks | Mostly | Filesystem read of workspace; `--redact` | Subprocess + timeout + MinimalSubprocessEnv — **PARTIAL** |
| Trivy | No | May fetch vuln DBs / network | Same — network **NOT_ISOLATED** |
| Grype | No | DB updates / network | Same |
| Semgrep | Mostly | Can run custom rules; generally parse | Same |
| Checkov | Mostly | Policy-as-code parse | Same |
| gosec / govulncheck / staticcheck | Mostly | Go toolchain may touch module cache | Same |
| Hadolint / shellcheck / linters | Mostly | File parse | Same |
| Syft / cyclonedx SBOM | Mostly | Package inventory; MinimalSubprocessEnv applied | PARTIAL |

**Do not assume scanners are passive.** Treat repository content as hostile input to these tools.

### Class B — repository code execution

| Path | Risk | Status |
|------|------|--------|
| Remediation validation / builds / repo scripts | High | Prefer ephemeral runners; **NOT_PROVEN** as enforced sandbox on control plane |
| Remediation PR auto-merge | Critical | **NOT_IMPLEMENTED** (intentional) |
| Runner forbidden tasks | Medium | **PROVEN** policy denylist |
| Dedicated runner VM | High isolation | Operational guidance — **NOT_PROVEN** as product-enforced |

### Process / container controls

| Control | Status |
|---------|--------|
| Minimal subprocess env (strip operator secrets) | **PROVEN** scanners/exec; **PROVEN** SBOM + runner clone (aligned) |
| Per-scanner timeouts + output caps | PROVEN |
| Non-root container user (UID 1001) | PARTIAL (image-dependent) |
| seccomp / AppArmor / no-new-privileges on scanner procs | NOT_IMPLEMENTED |
| Network namespace isolation per scan | NOT_IMPLEMENTED |
| No Docker socket by default | PROVEN for container-scan policy |
| Resource / PID limits per scan job | PARTIAL / NOT_PROVEN |
| Workspace cleanup | PARTIAL |

## Data flows involving repository content

| Flow | Content | Gate |
|------|---------|------|
| Forge issues / PR summary | Snippets (redacted) | Operator forge URL |
| AI auditors | File contents / prompts | `enable_llm_auditors` + `privacy_mode` |
| OpenClaw advisory | Findings ± snippets | Off by default; privacy gate |
| Notifications | Titles/summaries | Off by default; LOCAL_ONLY disables EXTERNAL |
| Runner clone | Full tree on worker | Delegation off by default |
| MCP | Findings via API | Local stdio host trust |

## Credential handling (RD-009)

| Control | Status |
|---------|--------|
| Access-log redaction of `api_key` | PROVEN / UNIT_TESTED |
| Issue/OpenClaw secret evidence redaction | PROVEN / UNIT_TESTED |
| Prefer `X-Repository-Detective-API-Key` header | PROVEN |
| Default reject query API keys | NOT default (compat); **recommended** for new installs |
| UI templates still can append `?api_key=` in api_key_only | PARTIAL — discouraged |
| Config validation errors must not print secrets | PARTIAL — patterns redacted in logs |

## Failure modes

- Missing required scanner → `EVALUATION_INCOMPLETE`, never silent `POLICY_MET`
- AI unavailable / privacy block → deterministic scans continue when auditors off
- DB unavailable → store-backed features degrade; `/health` reports

## Remaining NOT_PROVEN / NOT_IMPLEMENTED (honest backlog)

- Full Gitea E2E for PR summary upsert (RD-017)
- Per-scan network namespace / seccomp / AppArmor
- Ephemeral Class-B worker sandbox as default Community path
- DNS-rebinding-proof dialer (validate-at-connect pinning)
- Silent flip of existing installs to `auth_mode=local` (intentionally not done)

## Non-goals

- Not a SIEM, WAF, or secrets manager
- Not certified Common Criteria / FedRAMP / HIPAA
- Does not claim repositories are “safe” or “secure”
