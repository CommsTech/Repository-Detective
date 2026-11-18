# Repository Detective — independent full-acceptance handoff

**Scope:** run on an isolated staging instance with access to the actual Gitea/GitHub Actions runners, credentials, a disposable Gitea repository, a controlled AI provider, and a browser. **Do not promote to public beta until every P0 gate is evidenced.** This checklist is deliberately explicit about work not possible in the artifact-building environment.

## Handoff instructions to the testing agent

1. Check out the exact proposed commit. Record the full 40-character SHA, source tree checksum, running build version, container image digest, runner version, and start/end time in UTC.
2. Run **read-only preflight first**; use a disposable test repository and disposable webhook, user, and forge tokens. Do not use the production repository for destructive tests, secret-leak simulation, or automatic PR merging.
3. For every test below record **PASS / FAIL / BLOCKED / NOT TESTED**, the actual command or browser action, timestamp, output/trace ID, sanitized evidence, root cause, and any new Gitea issue.
4. Never claim a test passed because the source parsed, a browser screenshot looked good, or a mock succeeded. Track the real running build and scanner coverage.
5. Re-run the full suite on the final fixed SHA. A test on an earlier commit does not certify the current release.

## P0 — required release gates (untested here)

- [ ] **Go 1.26.11**: `go version`, `go mod verify`, `go mod tidy && git diff --exit-code go.mod go.sum`, `go vet ./...`, `go test -race -count=1 ./...`, `go build ./...` all exit zero. Record execution time and peak RSS.
- [ ] **Container reproducibility**: build the `core` and supported release images; start without personal paths, private forge names, or nonportable credentials. Check readiness, filesystem restrictions, and scanner versions in the resulting image. Scan the built image for known CVEs; review all critical/high issues.
- [ ] **Self-scan on push**: set CI secrets `RD_SELFSCAN_URL` (HTTPS) and `RD_SELFSCAN_API_KEY` (read/scan-only), variables `RD_SELFSCAN_OWNER`, `RD_SELFSCAN_REPO`, optionally `RD_SELFSCAN_REQUIRED`. Push a harmless test commit. Confirm the workflow actually **requests a new scan of that full SHA**, waits for completion, saves `executive-selfscan.md`, `executive-selfscan.json`, `SHA256SUMS`, and points to the commit and scan ID. Do not treat prior scan history as evidence.
- [ ] **Self-scan fail closed**: simulate (a) missing API credentials, (b) wrong SHA, (c) failed scan, (d) one missing required scanner, (e) bad result JSON, (f) timeout, (g) service 5xx, (h) secret-free clean scan. Require nonzero status for a–g, explicit incomplete report for failures, and no green 'secure' verdict for h.
- [ ] **Gitea and GitHub artifacts**: on both providers confirm uploaded Markdown/JSON/manifest can be downloaded and validated using `sha256sum -c SHA256SUMS`; artifact retention, permissions, and runner-action version compatibility verified. If Gitea artifact upload action v4 is incompatible, pin a vetted compatible action or native runner facility and rerun. Verify report access doesn't inadvertently reveal private repository metadata.
- [ ] **Supply-chain provenance**: compare checkout SHA, scanned SHA, built image labels, digest, release tag, and report scan commit. No mixed-version reports. Review signed provenance/attestation requirements before advertising release trust.
- [ ] **Feedback UI**: sign in as an owner, developer, security, and read-only user. Navigate to `UI > Report a problem`, complete form, add sample scrubbed logs, preview and copy, open Gitea issue draft, submit from the **user's own Gitea identity**, and confirm issue appears in the correct project with title, reproduction, expected/actual, version and scan ID.
- [ ] **Feedback privacy**: paste synthetic API tokens, JWTs, passwords, private key blocks, credentials in URLs, code snippets with dummy secrets, and long logs. Verify common secrets are masked, review-before-publish language is clear, no unexpected POST or upload leaves the browser, and nothing silently enters the application database. Since redaction is heuristic, **manually inspect the draft before submission**.
- [ ] **Feedback abuse/security**: test missing project URL, non-HTTPS URL, invalid/oversized input, links to an unauthorized forge, no login to Gitea, XSS payload in title/logs, mobile form usability, and browser pop-up blocking. Verify public reports cannot post with a server administrator token and no raw support bundle is exposed.
- [ ] **Authentication/RBAC**: local username/password logins, session expiry, CSRF, security/admin/developer/viewer roles, *repository-level* isolation if advertised, request-object authorization, pagination/export object isolation, and session invalidation. If repo isolation or SSO is not implemented, state beta limitation and do not expose shared tenant data.
- [ ] **Actual scanner lifecycle**: run a clean fixture, a known positive for each configured scanner, a deliberately bad dependency, a harmless fake secret, and a failed tool. Validate evidence completeness, false-negative/false-positive labeling, scan status, and actionable remediation guidance. Scan must not be 'verified' on missing tools.
- [ ] **Delegated sandbox**: run untrusted fixture on a disposable worker with no host mounts, restricted network, no server credentials, commit pinning, bounded CPU/RAM/disk, cleanup, and evidence transfer. Observe crash/timeout/duplicate-callback and ingestion retries; no dual issue creation.
- [ ] **AI remediation**: small local model (4B), medium and large model under identical budgets; verify deterministic prompts, token ceilings, parse-repair, tool error handling, human approval, sandbox tests, vulnerability rescan, PR creation, rollback and independent closure. Mark 'not implemented' instead of PASS where required.
- [ ] **Forge integrations**: issue assignment and duplicate detection across Gitea, GitHub, and Forgejo; use disposable repositories. False or unreviewed external disclosures must not auto-post to unrelated projects.
- [ ] **Export parity**: CSV and JSON must honor the same status/severity/category/owner/search/priority/suppression filters, include results beyond 200 rows, avoid leakage across users, protect spreadsheet formula cells, and show precise errors on server failure.
- [ ] **UI full coverage**: authenticated user journeys at 390, 768, 1024, 1440 widths, dark/light/system, keyboard-only, screen-reader labels, responsive tables, disabled actions, charts, dialogs, form submission, errors, all buttons, and deep links. Check dark-mode flash under slow and repeated scans.
- [ ] **Performance**: measure cold and warm p50/p95 for dashboard, repos, scan detail, finding detail, reports and exports with 1, 100, 1000 repositories. Measure throughput, peak RSS, disk temp growth and total storage IO under concurrent scans. Compare same hardware/data to previous build; investigate regressions.
- [ ] **Documentation and public packaging**: README installation from a fresh machine, quick start, config examples, no private hosts/tokens/datasets, accurate supported feature matrix, SECURITY.md private disclosure instructions, LICENSE/NOTICE, tests and changelog. Test all links and startup commands.

## P1 — desirable before broad public beta

- [ ] Add issue report button from failed scan, finding and system-health contexts, prepopulated with nonsecret IDs; audit that context is limited to authorized repository access.
- [ ] Add screenshots/videos documenting first-run onboarding, self-scan executive report, feedback issue, scan failure, remediation approval and verified closure.
- [ ] Add baseline metrics for independent learning/calibration precision and recall using pinned known-positive and known-negative corpora. A lower finding count alone is not a success metric.
- [ ] Confirm executive self-scan reports are surfaced from release notes or public release pages, not merely available in private CI artifacts. Link to exact commit-scoped report, and never claim 'safe' from a passing scan.
- [ ] Confirm `Gitea Actions -> Docker publish`, `Release`, and GitHub mirror cannot publish untested code; decide whether an executive selfscan is a **required release gate**, and wire cross-workflow gating with an auditable artifact identity rather than rescanning incompatible commits.

## Evidence package to return

Provide one folder containing `RESULTS.md` (test results and release recommendation), `env-versions.txt` (no secrets), sanitized `go-test.log`, `go-vet.log`, CI workflow run IDs, `executive-selfscan.md/json/SHA256SUMS`, container image digest, runner isolation proof, UI screenshots/traces, baseline latency and resource CSVs, issue URLs, and `OPEN_BLOCKERS.md` (each blocker with priority, reproducer, owner, and candidate fix).

**Disposition rule:** If any P0 security/data-isolation/scan-integrity gate is unverified or fails, report **CONTROLLED BETA ONLY / NO PUBLIC PROMOTION**. Do not change production policy, forge access, or release tags without operator authorization.
