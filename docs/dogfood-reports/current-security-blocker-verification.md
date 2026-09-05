# Current security blocker verification

**Product:** Repository Detective — Inspect. Analyze. Improve.  
**Date:** 2026-06-05  
**Branch verified:** `origin/main` (post `ab97c40` — Auth Slice 1 + Dockerfile fix)  
**Purpose:** Re-check historical P0 security claims against current code — not old audit notes.

---

## Executive summary

| P0 claim | Status | Internet exposure |
|----------|--------|-------------------|
| Webhook HMAC verification | **Fixed** (when secret configured) | Safe if `webhook_secret` set and `allow_insecure_webhooks=false` |
| `/api/v1` unauthenticated | **Fixed** | API requires operator key or runner HMAC |
| Tokens/secrets in logs | **Partial** | Evidence redaction strong; request/access logs not fully sanitized |
| Missing rate limiting | **Partial** | Webhook per-IP limit only; `rate_limit_per_minute` config unused on API |
| Unsafe base64 decode | **Fixed** | Errors returned; no silent ignore |
| Goroutine/context leak | **Partial** | Webhook analysis uses request context in background goroutine |
| Input sanitization gaps | **Partial** | html/template + body limits; API JSON validated per handler |
| Viper/config reliability | **Fixed** | Startup validation including `auth_mode=local` requirements |
| TLS verification issue | **Partial** | TLS 1.2+ default; opt-in `ai_insecure_skip_tls_verify` for homelab only |
| HTTP timeout issue | **Fixed** | Server Read/Write/Idle timeouts configured |

**Verdict:** P0 items **1–3 are not present** when deployed with production-safe defaults (`api_key` set, `webhook_secret` set, `allow_insecure_webhooks=false`). **Do not expose to the Internet** if `allow_insecure_webhooks=true` or webhook secret is empty without that flag being false.

---

## Item-by-item verification

### 1. Webhook HMAC not verified

```text
current_status: fixed
files_checked:
  - handlers/webhook.go (verifyWebhookSecret, HandleWebhook)
  - handlers/webhook_test.go
  - handlers/webhook_auth_test.go
  - main.go (POST /webhook)
evidence:
  - Raw body read before verification (readRequestBody → verifyWebhookSecret → bindJSONBody)
  - HMAC-SHA256 over body vs X-Gitea-Signature (or X-Hub-Signature-256)
  - Query-string secret rejected (TestVerifyWebhookSecretRequiresHMAC)
  - Empty webhook_secret rejects unless allow_insecure_webhooks=true (explicit insecure mode)
  - Per-IP rate limit before body read (10 req/s, burst 20)
tests:
  - TestVerifyWebhookSecretHMACSignature
  - TestVerifyWebhookSecretInvalidHMAC
  - TestVerifyWebhookSecretRequiresHMAC
  - TestVerifyWebhookSecretRejectsMissingSecretConfig
  - TestVerifyWebhookSecretAllowsInsecureMode
remaining_action:
  - Document allow_insecure_webhooks=false as production requirement (done in SECURITY_HARDENING)
  - Optional: webhook timestamp/replay window (not implemented)
```

### 2. `/api/v1` endpoints unauthenticated

```text
current_status: fixed
files_checked:
  - main.go (setupRoutes, registerControlPlaneRoutes, requireAPIKeyAuth)
  - api/handler.go and all api/* RegisterRoutes
  - api/security_test.go
  - main_auth_test.go
  - main_test.go
evidence:
  - setupRoutes: /api/v1 group uses requireComponentsReady + requireAPIKeyAuth
  - registerControlPlaneRoutes: same middleware on control plane group
  - /api/v1/onboard/* requires API key
  - /api/v1/runner/* worker routes use RequireRunnerHMAC (not operator API key)
  - Preferred X-Repository-Detective-API-Key + legacy X-Repository-Detective-API-Key + Bearer
  - Local session auth applies to /ui only, not JSON API
tests:
  - TestRequireAPIKeyAuthAcceptsPreferredAndLegacyHeaders
  - TestAPIKeyStillWorksWhenAuthModeLocal
  - TestControlPlaneRoutesRequireAPIKey
  - TestCSRFNotRequiredForAPIKeyJSON
remaining_action:
  - Add integration test hitting live router for /api/v1/status without key → 401 (optional)
```

**Public routes (intentional, not /api/v1):**

| Route | Auth |
|-------|------|
| `GET /health` | None (orchestrator probe) |
| `GET /`, `GET /onboard` | None (setup wizard) |
| `GET /ui/static/*` | None (assets) |
| `GET/POST /ui/login`, `/ui/bootstrap` | Public when `auth_mode=local` |
| `POST /webhook` | HMAC + rate limit |

### 3. Authorization/API tokens in logs

```text
current_status: partial
files_checked:
  - redact/secrets.go, issues/fingerprint.go, store/recorder.go
  - scanners/gitleaks.go (gitleaks --redact)
  - memory/qdrant/redaction.go
  - patcher/patcher_test.go (git output redaction)
  - handlers/webhook.go (logs repo name, not body/secrets)
evidence:
  - SecretEvidence redacts api_key/password/token/Bearer/AKIA patterns in findings and reports
  - Gitleaks runs with --redact
  - Webhook verification failures log error string only, not signature or secret
  - Gin default access logger may log ?api_key= query strings if used (homelab anti-pattern)
tests:
  - redact/secrets_test.go (added 2026-06-05)
  - TestRenderIssueBodyRedactsSecrets, TestRedactSummaryRemovesSecrets
remaining_action:
  - **P1:** Add middleware to strip/redact Authorization and API key headers from access logs; deprecate `?api_key=` in UI URLs for internet-facing deployments (preferred header: `X-Repository-Detective-API-Key`)
  - Document: never pass API key in URL for production UI
```

### 4. Missing rate limiting

```text
current_status: partial
files_checked:
  - handlers/webhook.go (per-IP limiter)
  - main.go (rate_limit_per_minute config — unused on API routes)
  - notify/rate_limit.go (notification cooldown only)
evidence:
  - Webhook: 10 req/s per IP with bounded map
  - API /api/v1: no global rate limit middleware wired to rate_limit_per_minute
tests:
  - Webhook rate limit not unit-tested (implicit via limiter.Allow())
remaining_action:
  - Wire rate_limit_per_minute to API group or document as notification-only (backlog)
```

### 5. Unsafe base64 decode / input parsing

```text
current_status: fixed
files_checked:
  - gitea/client.go GetFileContent
  - github/client.go (mirror)
evidence:
  - base64.StdEncoding.DecodeString with error return on failure
  - JSON bind errors return 400 on API handlers
  - security.MiddlewareMaxBody (1 MiB default)
tests:
  - gitea/github client tests cover content decode paths
remaining_action: none critical
```

### 6. Goroutine/context cancellation leak

```text
current_status: partial
files_checked:
  - handlers/webhook.go handlePushEvent, handlePullRequestEvent
  - main.go analysis goroutines
evidence:
  - Webhook spawns: go h.processor.ProcessPush(c.Request.Context(), payload)
  - Request context cancels when HTTP response completes — long scans may see early cancel
  - Analysis uses context.WithTimeout for manual scans
tests: none specific for webhook goroutine lifetime
remaining_action:
  - Use context.WithoutCancel or detached context with timeout for webhook-triggered work (backlog)
```

### 7. Input sanitization gaps

```text
current_status: partial
files_checked:
  - ui/templates/* (html/template auto-escape)
  - internal/security/middleware.go (headers, max body)
  - store parameterized SQL
  - ui/handler requireCSRF for form POSTs
evidence:
  - UI output encoding via html/template
  - CSRF on UI POST (API-key or session mode)
  - No global HTML sanitizer for API JSON responses (JSON only)
tests:
  - internal/security/middleware_test.go
  - ui/auth_handlers_test.go (CSRF)
remaining_action:
  - Continue per-handler validation; no mass user HTML input today
```

### 8. Viper/config reliability issue

```text
current_status: fixed
files_checked:
  - main.go loadConfig, validateAuth
  - internal/config/envcompat/envcompat.go
evidence:
  - Required forge tokens validated at startup
  - auth_mode=local requires session_secret and database_enabled
  - envcompat merges REPOSITORY_DETECTIVE_* (legacy prefixes removed)
tests:
  - TestValidateAuthLocalRequiresSessionSecret
  - TestValidateAuthDefaultsAPIKeyOnly
  - envcompat tests
remaining_action: none critical
```

### 9. TLS verification issue

```text
current_status: partial (by design for homelab)
files_checked:
  - ai/httpclient.go
evidence:
  - Default TLS min version 1.2
  - InsecureSkipVerify only when ai_insecure_skip_tls_verify=true (documented homelab-only)
  - Gitea/GitHub clients use default transport unless configured otherwise
tests:
  - ai client tests
remaining_action:
  - Document: never enable ai_insecure_skip_tls_verify on Internet-facing deployments
```

### 10. HTTP timeout issue

```text
current_status: fixed
files_checked:
  - main.go http.Server ReadTimeout/WriteTimeout/IdleTimeout (120s/120s/60s)
  - ai/httpclient.go Client Timeout 120s
evidence:
  - Server timeouts set at startup
tests: implicit via deployment stability
remaining_action: none critical
```

---

## Production deployment checklist

Before Internet exposure:

- [ ] `REPOSITORY_DETECTIVE_API_KEY` set (strong random)
- [ ] `REPOSITORY_DETECTIVE_WEBHOOK_SECRET` set
- [ ] `allow_insecure_webhooks: false` (default)
- [ ] `auth_mode: api_key_only` or validated `local` with `session_secret`
- [ ] `ai_insecure_skip_tls_verify: false`
- [ ] TLS termination at reverse proxy for HTTPS
- [ ] Do not use `?api_key=` in shared UI URLs

---

## Related

- [API_ROUTES.md](../API_ROUTES.md)
- [SECURITY_HARDENING.md](../SECURITY_HARDENING.md)
- [AUTH_LOCAL.md](../AUTH_LOCAL.md)
