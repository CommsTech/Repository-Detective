# CVE remediation worktree verification — 2026-11-16

**Status:** `PARTIAL`; not a public release acceptance receipt.

## Inputs

- User-supplied `repository-detective-main(1).zip` (untracked export; archive has no `.git` metadata).
- Code adjusted for CVE signals, AI advisory planning, remediation-plan preservation, and finding-page UI.
- Public CISA KEV / FIRST EPSS format checked against published documentation; **no live upstream or deployment service invoked**.

## Executed checks and results

| Check | Result |
|---|---|
| New `vulnintel` tests via local Go 1.23.2 in GOPATH/stdlib-only mode | PASS |
| `remediation` package tests via temporary isolated copy using local Go 1.23.2 | PASS |
| JavaScript `node --check ui/static/app.js` | PASS |
| Go `gofmt` for modified source | PASS |
| Go `html/template` parse of `ui/templates/finding_detail.html` with registered helpers | PASS |
| Full module `go test ./...` | NOT RUN (project `go.mod` requests Go 1.26.10; toolchain download unreachable) |
| Docker Compose build, image digest, container readiness | NOT RUN (Docker not present) |
| Staging Gitea webhook/scan/patch/closure E2E | NOT RUN (no staging services/credentials available) |
| External CISA/FIRST live network behavior | NOT RUN (mock-server tests only) |
| Browser UX and accessibility, CSRF/auth integration | NOT RUN (no running application) |

## Release blockers

1. Run `scripts/check-cve-workflow.sh` with Go 1.26.10 and module dependencies available, then investigate any failing tests.
2. Build an immutable image in staging and validate clean install, migrations, local UI auth and `/health/ready`.
3. Run a disposable Gitea/Forge test: scanner fixture, plan generation with local AI on/off, restart/re-scan plan stability, safe-PR rejection/approval, merge/redeploy and closure verification.
4. Run security review for new on-demand external requests (SSRF/redirect behavior, auth, privacy), and performance test CISA cache under concurrent load.
5. Complete responsive browser and accessibility checks for light/dark/mobile layouts.
6. Independently verify the resulting image digest and publish a separate acceptance report. Do not re-label `v0.1.0-beta.6` as containing these changes.

## Fail-safe behavior expected

- CVE intelligence off by default; source failure yields `Unknown`, not safe.
- AI opt-in; provider failure yields deterministic guidance plus explicit human-review blocker.
- Safe patcher cannot auto-change vulnerable dependencies; no auto-merge.
- Existing remediation plan is preserved across routine repeated scans.
