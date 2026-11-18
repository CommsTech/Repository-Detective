# Executive self-scan identity & feedback — independent acceptance checklist

**Baseline:** `repository-detective-main(20261118-171045).zip` plus this patch. **Disposition: staging / not production-verified.**

The offline checks below were executed in the artifact workspace; they do **not** prove a real scan or a public release.

## Local checks exercised

- `python3 -m unittest discover scripts/tests -v` — 12 tests passed against simulated API responses and report generation.
- `node scripts/tests/test_feedback.cjs` — issue-composer contract plus valid/invalid scan-ID prefill passed.
- `node --check ui/static/feedback.js` — JavaScript syntax passed.
- GitHub and Gitea CI workflow YAML parsed.

## P0: must be proven on the final release commit

For every item collect: **PASS/FAIL/BLOCKED**, exact full SHA, timestamp, command or UI action, sanitized evidence/log, issue ID if failed. Run on disposable repos and short-lived tokens.

- [ ] Go 1.26.11 build/vet/tests/race and frontend checks pass in Gitea CI and GitHub CI. Run `go mod verify`, `go vet ./...`, `go test -race -count=1 ./...`, and build release image; don't reuse evidence from an older commit.
- [ ] **Gitea push with no self-scan credentials**: self-scan job fails but uploads `executive-selfscan.md`, `executive-selfscan.json`, and `SHA256SUMS`. JSON `decision` is `EVIDENCE_INCOMPLETE`, not a missing artifact or a green pass.
- [ ] **GitHub push with no self-scan credentials**: same evidence and failure. Verify GitHub Actions v4 artifact upload; Gitea Actions v4 compatibility needs separate proof.
- [ ] **Successful Gitea self-scan**: configured service starts a NEW scan for the pushed full SHA with `forge_type=gitea` and `report_only_dry_run=true`; response acknowledges dry-run; separate repository lookup returns the same repository ID, `owner/name`, and `gitea` forge; `scan_id` matches the new job. Scan and scanner statuses are from that job, not a cached result.
- [ ] **Successful GitHub self-scan**: repeat with a connected GitHub mirror and `forge_type=github`, verifying the pushed SHA is available at the mirror. If GitHub integration cannot resolve that exact SHA, the job must report incomplete, not use a Gitea scan from another commit.
- [ ] Negative fixture/API tests: report an incomplete state (and nonzero CI status) for wrong `scan_id`, wrong repository ID, wrong owner/name, wrong forge, wrong/missing commit SHA, missing dry-run acknowledgement, missing required scanner, failed scanner, invalid scanner JSON, scanner timeout, API 5xx, and invalid base URL. Ensure no credentials are printed in logs or artifacts.
- [ ] Download both CI artifacts. From the artifact directory run `sha256sum -c SHA256SUMS`; both Markdown and JSON must verify. Review access control: artifacts of private repositories must not be inadvertently public.
- [ ] Demonstrate that the report is **not** a release approval. `REVIEW_FINDINGS` and `REVIEW_COVERAGE` require human assessment, even when the CI job exits 0; container image provenance and independent testing remain separate.
- [ ] Test scan detail -> **Report a scan problem** in authenticated browser sessions. Only the scan ID may prefill. Validate unauthorized user handling, keyboard navigation, narrow/mobile layout, dark-mode flash, copy/paste, redaction, and manual submission to a **disposable** Gitea issue tracker.
- [ ] Verify local login and account permissions; per-repository multi-tenant isolation, Authentik/Gitea OIDC, AI remediation, external issue filing and runner sandboxing remain separate release blockers unless independently proven.
- [ ] Compare built image digest to pushed source commit and self-scan artifact. Do not promote an image produced from an unverified or different revision.

## Return evidence package

`RESULTS.md`, `P0_BLOCKERS.md`, final full SHA, sanitized CI run URLs, build/test logs, runner versions, container digest, CI `executive-selfscan.md/json/SHA256SUMS`, one negative-fixture artifact, and browser screenshots or trace with secret material removed.

**Decision:** Do not claim production readiness if a P0 acceptance case is failed, blocked, or not tested. Self-scan evidence reduces uncertainty; it is not proof that all code is secure.
