# Runner and remediation verification: fail-closed corrections

**Status:** unreleased working-tree change; not a production verification receipt. **Date:** 2026-11-17.

## Operator impact

A green job status must not be inferred from the absence of errors on the screen. This update closes four fail-open paths:

1. **Remediation verification scan depth.** `remediation_verify` previously forced `AnalysisDepth=1`, preventing the external scanner stage from running. A nominally successful job could therefore have no actual external scanner evidence. It now runs depth 2 and returns the scanner outcomes and findings for review.
2. **Coverage gate.** A remediation verification job fails visibly if no source files were analyzed; if no applicable external scanner finished; if an external scanner timed out, failed to parse, is missing or otherwise incomplete; or if findings remain. Disabled and inapplicable scanners are not credited as successful verification evidence.
3. **Worker callback integrity.** A `completed` status accompanied by an error list is normalized to `failed` before status and summary persistence or ingestion callbacks.
4. **Worker scope.** Unknown job types fail rather than triggering a default scan, and an empty `JobTypeAllowed` list grants no permission.

The scan-detail screen now marks coverage review as **not automatically proven** when the pipeline completes; highlights missing/unavailable scanners; and displays a warning if no scanner results exist.

## Limits that still block automatic trust claims

- **This is NOT an implemented dependency-upgrade or AI-patch engine.** It neither generates nor merges a fix.
- A completed scan does **not** prove the intended issue was fixed. Before a fix is considered verified, implement target-specific fingerprint/CVE/commit matching, isolated build and test execution, PR diff review, and scanner regression gates. The current conservative verification may fail on unrelated pre-existing findings; that is a safety bias, not target-aware remediation.
- The native Docker worker is not a strong boundary for adversarial code. Disposable VMs or equivalent isolation, bounded credentials, and restrictive egress are necessary for untrusted repositories. Do not mount the host Docker socket or supply the main application's AI/forge credentials to the worker.
- Container/image SBOM, graph, Gitea Actions/GitHub Actions and automatic PR workflows have separate acceptance criteria; this patch does not certify them.

## Regression tests added

- `TestRemediationVerificationRejectsMissingAndFailedScannerEvidence`
- `TestRemediationVerificationCleanEvidenceIsOnlyScanEvidence`
- `TestRemediationVerificationDoesNotSucceedWithScannersDisabled`
- `TestUnknownJobTypeNeverExecutesScan`
- `TestEmptyWorkerAllowlistDeniesAllJobs`
- `TestReceiverCompletedStatusWithErrorsFailsClosed`

Recommended verification on Go **1.26.10**:

```bash
GOTOOLCHAIN=local go test ./runner -count=1
GOTOOLCHAIN=local go test ./... -count=1
```

Then perform a disposable Gitea + native worker E2E with missing scanner binaries, scanner errors, a vulnerable fixture, a clean fixture, and malicious repository fixtures. Check that UI status, DB status, job result artifacts, webhook notification, and policy/closure semantics all agree. Avoid production rollouts until both paths are evidenced.

## Local build environment

The author's current environment provides **Go 1.23.2**, below the repository's `go 1.26.0` + `toolchain go1.26.10` requirements. The module tests are **NOT RUN** here. Go formatting and file-level static checks are not replacements for Go unit, integration, or full system acceptance.
