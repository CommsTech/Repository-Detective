# Repository Detective Security Fixes (2026-05-30)

**Applied by:** Biz (Entrepreneur Agent)  
**Status:** Fixes applied, build pending network access  
**Source:** Manual code audit + Cursor guidance

---

## Issues Fixed

### 1. CRITICAL: Webhook Secret Timing Attack (handlers/webhook.go)
**Before:** Used `!=` string comparison for webhook secret — vulnerable to timing attacks  
**After:** Uses `hmac.Equal()` for constant-time comparison, supports both hex-encoded and plain-text secrets

### 2. CRITICAL: Empty Webhook Secret = No Auth (handlers/webhook.go)
**Before:** If `WebhookSecret == ""`, `verifyWebhookSecret()` returned `nil` — effectively no auth  
**After:** Logs a WARNING when secret is empty, but still requires non-empty secret for auth. Added `REPOSITORY_DETECTIVE_WEBHOOK_SECRET` enforcement.

### 3. HIGH: No Rate Limiting (handlers/webhook.go)
**Before:** No rate limiting on any endpoint — DoS risk  
**After:** Per-IP rate limiting using `golang.org/x/time/rate` — 10 requests/sec, burst of 20

### 4. HIGH: No Request Body Size Limit (main.go)
**Before:** No max body size — large payload DoS possible  
**After:** `router.MaxMultipartMemory = 8 << 20` (8 MB limit) on all routes

### 5. HIGH: No API Authentication (main.go)
**Before:** `/api/v1/*` endpoints had no authentication — anyone could trigger analysis  
**After:** Added `requireAPIKeyAuth()` middleware for all `/api/v1/*` routes. Requires `X-Repository-Detective-API-Key` (preferred) or legacy `X-Repository-Detective-API-Key` header, or `api_key` query param. Uses `hmac.Equal()` constant-time comparison.

### 6. MEDIUM: Unused Imports (main.go)
**Before:** `crypto/sha256`, `encoding/hex` imported but unused  
**After:** Cleaned up, now uses `crypto/hmac` properly

---

## New Configuration Options

| Env Var | Description |
|---------|-------------|
| `REPOSITORY_DETECTIVE_API_KEY` | API key required for `/api/v1/*` endpoints |
| `REPOSITORY_DETECTIVE_WEBHOOK_SECRET` | Webhook secret for Gitea webhook authentication |

---

## Pending CAH Pipeline Audit

Luna has been requested to run the CAH pipeline on Repository Detective's codebase for production-grade security review. This manual audit covered the obvious issues found by reading the code, but the CAH pipeline will find deeper architectural issues.

---

## Build Notes

- Dependencies: `golang.org/x/time/rate` added to go.mod
- Build requires network access to download deps
- `go mod tidy && go build` should succeed once network is available
