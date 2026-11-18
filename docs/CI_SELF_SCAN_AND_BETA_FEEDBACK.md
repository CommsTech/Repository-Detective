# Commit-specific executive self-scan and beta feedback

## Why it exists

Every source push to `main` or `develop` triggers a separate CI job that requests a **new Repository Detective scan of the pushed SHA**, retrieves the scan and scanner outcomes, and publishes a Markdown + JSON executive evidence package with SHA-256 manifest. **If configuration is missing or malformed, it still publishes an explicit incomplete report and fails the job**; a skipped job without an artifact is not acceptable evidence. It is not generated from the previous deployed report, nor is it a replacement for release scanning of the final container image.

The job is **not opt-in**. There is no `RD_SELFSCAN_ENABLED` switch. On a push with no self-scan credentials, the scanner script writes an `EVIDENCE_INCOMPLETE` report and exits with failure; the checksum verification and artifact-upload steps still run. CI should preserve the incomplete artifact for reviewers. The separate source-test job may still pass; self-scan success is **not yet an enforced release-publishing gate**. A skipped job, missing artifact, or incomplete decision cannot be described as a successful self-scan.

**Outputs** (Actions artifacts, not source-tree commits):

- `executive-selfscan.md` — readable decision, commit, scan ID, scanner coverage/outcomes, blockers and limitations
- `executive-selfscan.json` — machine-readable evidence (schema_version=1)
- `SHA256SUMS` — checksums for both the JSON evidence and Markdown executive report

## Configuration

Configure secrets on both forges used for CI:

- `RD_SELFSCAN_URL`: HTTPS base of an accessible, isolated Repository Detective instance (no trailing `/api/...`).
- `RD_SELFSCAN_API_KEY`: dedicated least-privilege API key with permission to scan and read evidence. Never print or embed it in reports.

Configure Actions variables (the job runs even when the values are absent):

- `RD_SELFSCAN_OWNER`: project repository owner on the Gitea instance reachable by the scan service.
- `RD_SELFSCAN_REPO`: repository name (defaults to `Repository-Detective`).
- `RD_SELFSCAN_REQUIRED`: nonempty scanner list. Default `gitleaks,govulncheck,staticcheck`. An empty list fails with `EVIDENCE_INCOMPLETE`. Confirm these tools are installed **inside the executing scanner runtime**, and tailor the requirement to the repository languages and policy.

Gitea CI selects forge `gitea`; GitHub CI selects forge `github`. The corresponding forge repository must be connected and reachable by the scan service for its own source push. Both workflows call the existing API with the pushed SHA (`GITHUB_SHA`) and `report_only_dry_run=true` (acknowledgement required), so no forge issues are intentionally filed by this scan.

Before accepting the result, the job compares **the newly requested scan ID**, the persisted scan commit, and the scan's repository ID with an **authenticated repository metadata lookup** (`GET /api/v1/repos/:id`). The repository owner/name and forge must agree with the CI configuration. A matching commit from the *wrong repository* is not valid evidence. Incomplete scanner coverage, an invalid configuration, wrong forge or SHA, a missing dry-run acknowledgement, and API timeouts produce a failed job with a report artifact. A failed or unavailable scan is **EVIDENCE_INCOMPLETE**, causing the job to fail and generating an incomplete report when configuration permits.

A completed scan **is not a security certification**: statuses such as `found` may indicate real defects, and scanner counts are per-tool events. Operators must review findings and the actual coverage before promoting a release. Production image digest provenance and cross-workflow release gating remain independent acceptance tasks.

## Feedback submission

Set `REPOSITORY_DETECTIVE_PROJECT_URL=https://your-gitea.example/owner/Repository-Detective` in the application environment and restart. The operator UI now includes **Report a problem**. The user enters reproducible steps, optional version/scan ID, and may opt into a log excerpt. The page performs best-effort local masking for common credentials and produces reviewable Markdown. The user must copy/paste longer drafts, authenticate to Gitea and submit.

Scan Details now links to the local composer with the **scan ID only**. No repository source or raw evidence is copied into the draft. No server-side issue-posting account, hidden file upload, or raw support bundle forwarding is added. Credential redaction is **not guaranteed**. For vulnerabilities in Repository Detective, use private security advisories instead of a public issue. Existing Gitea issue templates are still available.

## Local/offline QA

```sh
python3 -m unittest discover scripts/tests -v
node scripts/tests/test_feedback.cjs
node --check ui/static/feedback.js
python3 scripts/ci-selfscan-executive.py --help
```

The final acceptance run is defined in [release/EXTERNAL_AGENT_FULL_ACCEPTANCE_2026-11-18.md](release/EXTERNAL_AGENT_FULL_ACCEPTANCE_2026-11-18.md).

## Staging handoff for CI self-scan

1. Push a harmless commit to `main` in a disposable mirror. Run Gitea and GitHub source CI separately and confirm each uses its own `--forge` selection.
2. Without `RD_SELFSCAN_URL`, confirm `executive-selfscan.md`, `executive-selfscan.json`, and `SHA256SUMS` are uploaded **despite the job failing**. Inspect `release_blockers` for missing configuration.
3. Configure the scan service. Confirm it returns the same new `scan_id`, a full commit SHA, matching repository ID/name/forge from `/api/v1/repos/:id`, and explicit dry-run acknowledgement. Verify `sha256sum -c SHA256SUMS` within the downloaded artifact.
4. Deliberately scan a different repository at the same SHA (or use a fixture API); require `EVIDENCE_INCOMPLETE`. Also simulate wrong forge, missing scanner, non-object scanner response, failed execution and timeouts.
5. Inspect the result and ensure that **REVIEW_FINDINGS / REVIEW_COVERAGE are not release approvals**. If you want this to block image publication, add an explicit tested cross-job and cross-workflow gate tied to this exact source commit and image digest.
6. Verify the `Report a scan problem` link opens the feedback page with an opaque scan ID, no attached logs/source, and a user-reviewed Gitea draft. Use temporary accounts and fake logs only.

Live CI artifact upload on Gitea Actions, release gating, and multi-forge identity mapping still require verification on the actual runners.

For the commit-specific expanded acceptance matrix, see [CI self-scan identity acceptance](release/CI_SELFSCAN_IDENTITY_ACCEPTANCE_2026-11-18.md). For this patch, see [always-on self-scan acceptance](release/SELF_SCAN_ALWAYS_ON_ACCEPTANCE_2026-11-18.md).
