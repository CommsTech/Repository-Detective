# Security model (overview)

Repository Detective — **Inspect. Analyze. Improve.**

High-level trust boundaries. See [SECURITY_HARDENING.md](SECURITY_HARDENING.md) for OWASP-oriented checklist.

## Trust zones

```text
[Gitea] <--webhook/token--> [Repository Detective API/UI]
                                |
                    +-----------+-----------+
                    |           |           |
               [SQLite]    [Scanners]   [Optional AI]
                    |           |           |
              operator DB    subprocess   external API
```

## Authentication

| Surface | Mechanism |
|---------|-----------|
| `/api/v1/*` | API key header/query (when configured) |
| `/ui/*` | API key (when configured) |
| Webhooks | HMAC `webhook_secret` |
| Runners | `runner_shared_secret` |

`/health` is unauthenticated (liveness only; no secrets in response when healthy).

## Authorization

- Gitea token scopes limit repo/issue/PR actions
- Repository Detective does not implement multi-tenant RBAC — one deployment serves configured org/repos

## Input handling

- Repository content is **untrusted** — scanners run with minimal env ([internal/security/env.go](../internal/security/env.go))
- HTML templates auto-escape Go template variables
- Issue bodies sanitize evidence snippets ([issues/fingerprint.go](../issues/fingerprint.go))

## Output handling

- Gitea issues: sanitized evidence, fingerprints for dedup
- Notifications: redacted payloads ([notify/templates.go](../notify/templates.go))
- Logs: heuristic redaction for scanner detail fields (expanding; see backlog)

## Failure modes

- Missing scanner → degraded coverage, not full outage
- DB unavailable → features depending on store fail; `/health` reports degraded
- AI unavailable → deterministic scans continue if configured

## Non-goals

- Not a SIEM, not a WAF, not a secrets manager
- Not certified Common Criteria / FedRAMP / HIPAA
