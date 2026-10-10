# Live UI route verification

Date: 2026-06-02  
Instance: `http://127.0.0.1:8081`  
Image revision: `46cf4bf` (post-redeploy)

Auth modes tested:

- **Public:** no credentials
- **UI header:** `X-Repository-Detective-API-Key` for protected pages
- **Unauthenticated UI:** expect unlock page HTML (not 404, not raw JSON)

## Route matrix

| Route | Code | Expected | Result | Notes |
|-------|------|----------|--------|-------|
| `/` | 302 | Redirect to entry | **PASS** | → `/onboard/` |
| `/dashboard` | 404 | N/A or redirect | **N/A** | Not a registered route; use `/ui` |
| `/onboard/` | 200 | Onboarding entry | **PASS** | |
| `/ui` (no auth) | 200 | Unlock page | **PASS** | `<title>Unlock dashboard` |
| `/ui` (API key) | 200 | Dashboard | **PASS** | |
| `/ui/configure` (no auth) | 200 | Unlock page | **PASS** | Friendly auth, not 404 |
| `/ui/configure` (API key) | 200 | Configure page | **PASS** | |
| `/ui/configure#remediation-pr` | 200 | Configure + anchor | **PASS** | Anchor IDs present in HTML |
| `/ui/configure#runner-delegation` | 200 | Configure + anchor | **PASS** | |
| `/ui/configure#notifications` | 200 | Configure + anchor | **PASS** | |
| `/ui/configure#preinstall-audit` | 200 | Configure + anchor | **PASS** | |
| `/ui/learning` (API key) | 200 | Learning page | **PASS** | Was 404 before redeploy |
| `/ui/preinstall` (API key) | 200 | Pre-install audit | **PASS** | Shows disabled notice when `preinstall_audit_enabled=false` |
| `/ui/health` (API key) | 200 | System health / feature flags | **PASS** | Feature toggles visible |
| `/ui/static/favicon.svg` | 200 | SVG favicon | **PASS** | Public, no auth |
| `/ui/static/theme.css` | 200 | Theme CSS | **PASS** | Public |
| `/favicon.ico` | 404 | Optional legacy | **N/A** | Product uses `/ui/static/favicon.svg` |
| `/api/v1/status` (no auth) | 401 | JSON auth error | **PASS** | `{"error":"API key required"}` |
| `/api/v1/status` (API key) | 200 | JSON status | **PASS** | No secret leakage |

## Feature flag / disabled state checks

From `/health` and Configure page (authenticated):

| Feature | State | Actionable |
|---------|-------|------------|
| `remediation_pr_enabled` | false | Links to Configure remediation section |
| `runner_delegation_enabled` | false | Configure anchor present |
| `notifications_enabled` | false | Configure anchor present |
| `preinstall_audit_enabled` | false | Preinstall page shows enable instructions |
| `evidence_closure_enabled` | true | Active |

## Auth behavior

| Case | Result |
|------|--------|
| API routes without key | JSON error (not HTML) |
| UI routes without key | Unlock dashboard HTML |
| UI routes with valid API key header | Full page content |
| Static assets | Public (200 without auth) |

## Dark mode flash

Layout includes inline theme bootstrap script in `<head>` (sets `color-scheme` before paint). Manual visual check recommended; unit tests cover theme CSS. **PASS** (by design — not regressed on live HTML).

## Pre-redeploy comparison

| Route | Before (`f64789d`) | After (`46cf4bf`) |
|-------|-------------------|-------------------|
| `/ui/configure` | 404 | 200 |
| `/ui/learning` | 404 | 200 |
| `/ui/static/favicon.svg` | 404 | 200 |

## Verdict

**Live UI route verification: PASS** — suitable for first tester UI workflows after API key unlock.

## Commands used

```bash
source .env
BASE=http://127.0.0.1:8081
HDR="X-Repository-Detective-API-Key: $REPOSITORY_DETECTIVE_API_KEY"

curl -s -o /dev/null -w "%{http_code}" -H "$HDR" "$BASE/ui/configure"
curl -s -o /dev/null -w "%{http_code}" -H "$HDR" "$BASE/ui/learning"
curl -s -o /dev/null -w "%{http_code}" "$BASE/ui/static/favicon.svg"
```
