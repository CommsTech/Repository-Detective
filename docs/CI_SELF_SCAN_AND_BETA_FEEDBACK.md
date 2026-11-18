# Commit-specific executive self-scan and beta feedback

## Why it exists

Every source push to the primary branches triggers a separate CI job that requests a **new Repository Detective scan of the pushed SHA**, retrieves the scan and scanner outcomes, and publishes a Markdown + JSON executive evidence package with SHA-256 manifest. It is not generated from the previous deployed report, nor is it a replacement for release scanning of the final container image.

**Outputs** (Actions artifacts, not source-tree commits):

- `executive-selfscan.md` — readable decision, commit, scan ID, scanner coverage/outcomes, blockers and limitations
- `executive-selfscan.json` — machine-readable evidence (schema_version=1)
- `SHA256SUMS` — checksum for the JSON evidence

## Configuration

Configure secrets on both forges used for CI:

- `RD_SELFSCAN_URL`: HTTPS base of an accessible, isolated Repository Detective instance (no trailing `/api/...`).
- `RD_SELFSCAN_API_KEY`: dedicated least-privilege API key with permission to scan and read evidence. Never print or embed it in reports.

Configure Actions variables:

- `RD_SELFSCAN_OWNER`: project repository owner on the Gitea instance reachable by the scan service.
- `RD_SELFSCAN_REPO`: repository name (defaults to `Repository-Detective`).
- `RD_SELFSCAN_REQUIRED`: scanner list. Default `gitleaks,govulncheck,staticcheck`. Confirm these tools are installed **inside the executing scanner runtime**, and tailor the requirement to the repository languages and policy.

Both CI workflows use the existing API to scan **exactly** the SHA from the push (`GITHUB_SHA`) with `report_only_dry_run=true`, so no forge issues are filed by this CI scan. It checks that the scan's recorded SHA equals the pushed SHA and that evidence from required scanners exists. A failed or unavailable scan is **EVIDENCE_INCOMPLETE**, causing the job to fail and generating an incomplete report when configuration permits.

A completed scan **is not a security certification**: statuses such as `found` may indicate real defects, and scanner counts are per-tool events. Operators must review findings and the actual coverage before promoting a release. Production image digest provenance and cross-workflow release gating remain independent acceptance tasks.

## Feedback submission

Set `REPOSITORY_DETECTIVE_PROJECT_URL=https://your-gitea.example/owner/Repository-Detective` in the application environment and restart. The operator UI now includes **Report a problem**. The user enters reproducible steps, optional version/scan ID, and may opt into a log excerpt. The page performs best-effort local masking for common credentials and produces reviewable Markdown. The user must copy/paste longer drafts, authenticate to Gitea and submit.

No server-side issue-posting account, hidden file upload, or raw support bundle forwarding is added. Credential redaction is **not guaranteed**. For vulnerabilities in Repository Detective, use private security advisories instead of a public issue. Existing Gitea issue templates are still available.

## Local/offline QA

```sh
python3 -m unittest discover scripts/tests -v
node scripts/tests/test_feedback.cjs
node --check ui/static/feedback.js
python3 scripts/ci-selfscan-executive.py --help
```

The final acceptance run is defined in [release/EXTERNAL_AGENT_FULL_ACCEPTANCE_2026-11-18.md](release/EXTERNAL_AGENT_FULL_ACCEPTANCE_2026-11-18.md).
