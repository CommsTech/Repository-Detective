# First tester feedback sprint verification

Date: 2026-06-02  
Commit: `97aaafb` (+ staticcheck cleanup)

## Tests

| Command | Result |
|---------|--------|
| `go test ./...` | **PASS** |
| `go vet ./...` | **PASS** (via test container) |
| `staticcheck ./...` | **PASS** (`GOFLAGS=-buildvcs=false`) |
| `./scripts/operator-smoke-test.sh` | **PASS** (live homelab — pre-redeploy image; rebuild recommended for new UI) |

## Docker

| Item | Status |
|------|--------|
| `docker-build-verify.sh` | Not re-run (~23 min); prior pass documented |
| Live redeploy for sprint UI | **Recommended** — `docker build --target all-in-one` + recreate container |

## Beta package

| Item | Status |
|------|--------|
| `make beta-release` | **PASS** at `97aaafb` |
| Secrets check | **PASS** |

## Feature verification

| Feature | Status |
|---------|--------|
| Manual Scan Now UI + API `scan_id` | Implemented + unit tests |
| Report-only default when filing off | Enforced in form + handler |
| Repo settings sections + badges | Implemented + unit test |
| Favicon matches logo (`?v=2`, `/favicon.ico` redirect) | Implemented + unit test |
| Issue/finding reconciliation panel | Implemented + store/API tests |
| AMMBER-like report-only pattern | Covered by `TestReconciliationSummaryReportOnly` |

## UI routes (after redeploy)

| Route | Expected |
|-------|----------|
| `/ui/repos/:id/scan` | Scan confirmation form |
| `/ui/repos/:id` | Reconciliation panel + Scan now |
| `/ui/scans/:id` | Reconciliation panel scoped to scan |
| `/favicon.ico` | Redirect to SVG icon |
| `/ui/repos/:id/settings` | Sectioned policy reference |

## Manual scan safety (verification design)

Report-only manual scan via API/UI must:

- Set `trigger_type=manual`
- Return `scan_id` immediately
- Skip issue filing when `report_only_dry_run=true`
- Create **0** forge issues

Live report-only run deferred to post-redeploy operator check (same pattern as prior validation scans).

## Product repo / safety gates

| Gate | Status |
|------|--------|
| Open issues | 1 (#48 operator task) |
| Active-present | 0 |
| Non-product issue filing | Disabled by default |
| All-repo scan | Not started |
| LLM sanity gate | Disabled by default |

## Recommendation

**Ready for continued private beta** with redeployed UI. Tester feedback items addressed; no issue filing enabled by default.

## Remaining UX follow-ups (non-blocking)

- Live redeploy to pick up Scan now + reconciliation panel
- Optional: inline scan modal without separate page
- Optional: forge live open-issue count (currently DB mappings only)
