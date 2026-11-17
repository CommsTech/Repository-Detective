# Runner result and scanner evidence integrity — 2026-11-17

**Scope:** source-only development update based on `repository-detective-main(10).zip`.
**Status:** proposed for CI/staging; **not deployed, not full-suite verified**.

## Problem and correction

Delegated workers already supplied a per-scanner `findings_count` and optional `error` in the runner contract. The core's `JobResult.ToAnalysisResult` discarded both, and `finishPersistedScan` calculated scanner counts from a separate, unpopulated `RunResult.Findings` list. As a result, successful delegated scans could persist **zero scanner findings** even if the worker reported matches. Individual scanner errors also could be lost before the operator viewed coverage evidence.

- The core now preserves worker-reported counts independently from normalized issue records and persists the greater of the actual per-scanner findings list and the reported count. No finding records or issue counts are synthesized from scanner totals.
- Any scanner outcome containing an `error` becomes `failed` for coverage reporting; error/detail text is redacted and truncated before persistence.
- Custom declarative rules emit the standard `clean` or `found` scanner status rather than `completed`, which is not a normal scanner coverage status.
- Runner result validation rejects unknown or empty job statuses, mismatched result-contract versions, negative counts, and reversed nonzero timestamps. Error-bearing completed jobs still fail closed as before.
- The runner ingestion path records the job's immutable pinned SHA when available instead of relying exclusively on the mutable branch ref; the worker must independently prove the checkout matches the SHA.

## Regression tests added

`runner/result_integrity_contract_test.go` checks malformed callback envelopes, count/error propagation and redaction, and `custom_rules` success/clean outcome vocabulary. Receiver test fixtures now send the explicit version-1 contract.

## Required release acceptance

1. With matching core and worker builds, run `GOTOOLCHAIN=local go test ./runner ./scanners ./store -count=1`, `go vet ./...`, and `go test ./... -count=1` on the required Go **1.26.11** toolchain.
2. Run a disposable Gitea/native-worker scan against a fixture producing multiple gosec findings; compare `runner_jobs` result summary, `scanner_results.findings_count`, the scan details page and the repository PDF. All must show the same per-scanner count. The total persisted issue count may differ due to deduplication and is **not** a substitute for scanner counts.
3. Simulate a tool returning `status=clean` with an `error`; the operator must see a **failed scanner**, not a clean coverage indicator. Verify credential-like error strings are redacted.
4. Run the custom-rule worker twice, once with a matching line and once without. Coverage statuses must be `found` and `clean` respectively and must agree with stored findings.
5. Verify the scan's SHA matches the checkout's `git rev-parse HEAD` and the reported forge issue commit. Test a ref movement during cloning; it must fail rather than silently scan a different commit.
6. Check that unsupported runner-result statuses and versions return a client error without a completed job row. Exercise cancellation/expiry and error cases, then review any failed ingestions.

## Known unresolved issues

- This change is **not** a disposable VM sandbox, native Actions dispatcher, validated AI remediation/PR engine, or proof of full scanner coverage.
- The result ingestion callback currently records the runner job final state before executing all downstream analysis/persistence side effects. A separate transactional/idempotent workflow is needed to guarantee retry-safe exactly-once ingestion across failures and concurrent callbacks.
- Scanner counts are worker-reported evidence, not independent certification. A compromised worker can lie; worker isolation, credential scoping, and independent attestations remain mandatory.
- This environment has Go 1.23.2, so the full module tests are **not executable here**. The maintainer's earlier Go 1.26 CI success applies to a prior deployed baseline, not this source patch.
