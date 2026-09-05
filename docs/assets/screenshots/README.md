# Documentation screenshots (public beta)

These images are captured from a **disposable** demo environment only.

**Do not** publish screenshots from production forges, private hostnames, personal usernames, live credentials, or real customer findings.

Synthetic names used in captures:

- `demo/repository-detective-test`
- `example/app`
- `demo-api`

## Required set (RD-020)

| File | Subject |
|------|---------|
| `01-onboarding-connect.png` | Onboarding — Connect stage |
| `02-onboarding-protect.png` | Onboarding — Protect stage |
| `03-doctor.png` | Doctor / diagnostics |
| `04-dashboard.png` | Repository overview / dashboard |
| `05-finding-evidence.png` | Canonical finding with evidence |
| `06-policy-evaluation.png` | Policy evaluation (Observe / outcome) |
| `07-pr-compact-summary.png` | PR compact policy summary (disposable Gitea; format matches product marker) |
| `08-privacy-local-only.png` | Privacy **LOCAL_ONLY** state |
| `09-remediation-plan-preview.png` | Finding page remediation area (plan preview; **no** PR execution) |

### Capture environment (Phase 7)

- Local disposable RD binary on `127.0.0.1:18081` with seeded synthetic SQLite (`demo/repository-detective-test`)
- Disposable Gitea `1.22.3` on `127.0.0.1:13000` (`rd-e2e-gitea`) for PR summary only
- Playwright `mcr.microsoft.com/playwright:v1.55.0-jammy` headless Chromium

Never used production `:8081` / private fleet data for these captures.

Optional legacy filenames may exist for older docs; prefer the numbered set above in README.

## Capture

```bash
# Disposable stack (not production :8081)
export RD_E2E_IMAGE=repository-detective:v0.1.0-beta.3
export RD_SCREENSHOT_BASE=http://127.0.0.1:18081
export REPOSITORY_DETECTIVE_API_KEY=e2e-acceptance-api-key-not-a-secret
./scripts/e2e-gitea-acceptance.sh   # or compose up + DEMO.md seed
./scripts/capture-phase7-screenshots.sh
```

Privacy review before commit: no `commsnet`, private IPs, tokens, or real repo names in PNG metadata/`strings`.

## Quarantine note

Older June 2026 captures under this directory that showed private fleet data were **removed from the public doc set** during Phase 7 and must not be re-linked.
