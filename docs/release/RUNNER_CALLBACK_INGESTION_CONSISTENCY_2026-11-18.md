# Runner callback ingestion consistency — 2026-11-18

## Problem and scope

Before this change, the runner receiver marked an accepted result `completed` and ignored artifact-write errors **before** invoking the scan persistence callback. If the callback failed, the API returned an error but the persisted job still looked successful. A second callback could race the first, or a network retry after a successful callback could receive a misleading duplicate error.

This change addresses the **callback acceptance and job-state consistency boundary**. It is not a complete transactional pipeline for all downstream issues, notifications, and SBOM persistence.

## Lifecycle

1. Only `running` or `dispatched` jobs may submit results. Jobs that have not been claimed cannot self-complete.
2. The receiver validates the payload and computes a SHA-256 digest of its original canonical JSON representation.
3. A SQLite compare-and-swap transition moves the job to `ingesting`. Only one callback may claim this stage; another callback cannot execute downstream ingestion concurrently.
4. The complete result JSON, size, and digest are saved as an artifact. Artifact write errors are no longer ignored.
5. The ingestion callback must succeed, including recording the scan completion, findings, and any requested forge-issue links. Only then can the job transition from `ingesting` to `completed` or `failed` (for an actual failed worker result).
6. If artifact storage or ingestion fails, the receiver records `ingestion_failed` and an error. The API returns an error. Retries cannot automatically rerun partially applied side effects; an operator must reconcile the incomplete scan and any forge issues.
7. A later repeat of a **successfully finalized** callback is acknowledged if its canonical JSON digest matches. A different result is rejected.
8. An abandoned `ingesting` job can expire at its configured expiry time; an expiration does not turn the job into a successful result.

`ingesting`, `ingestion_failed`, and `expired` are **not verified scan outcomes**. Operators and downstream automation must not interpret them as a clean scan or a completed remediation.

## Implementation boundaries

- Atomicity is limited to runner job state changes. Scan persistence, finding reconciliation, SBOM writes, notifications, issue filing, and external forge calls are not a single SQLite transaction.
- If a process terminates after a scan side effect but before the final runner transition, it may leave an incomplete job with some downstream records already written. The system explicitly **does not** auto-replay this result.
- The callback digest compares the structured JSON representation of the received result, not a cryptographic signature of scanner accuracy or a trusted sandbox attestation. Existing runner HMAC verification remains required at the HTTP boundary.
- `finishPersistedScan` now returns the scan recorder's error to delegated ingestion. Other existing callers still use the original best-effort behavior and require separate reliability review.
- Artifact write success alone does not prove scanner coverage, valid patches, or safe execution. Runner isolation, verified AI remediation, and Actions delegation remain separate gates.

## CI and staging acceptance

Run with Go **1.26.11**, per `go.mod`:

```bash
go test ./runner ./store ./api ./... -count=1
go test -race ./runner ./store -count=1
go vet ./...
```

Test an authenticated native worker against disposable Gitea and SQLite instances:

- Successful callback: artifact exists, scan completion persisted, job `completed`.
- Duplicate callback (identical JSON): acknowledged with no duplicate findings/issues.
- Changed callback: rejected; original evidence remains unchanged.
- Concurrent callbacks: only one runs ingestion.
- Failed worker execution: status `failed`, failure evidence persisted.
- Fault injection during artifact insert or scan recorder completion: status `ingestion_failed`, no false success.
- Restart during ingestion: job remains nonterminal and expires or receives explicit operator reconciliation.
- Unclaimed, canceled, or expired job: result submission refused.
- Authenticated Gitea issue/PR handling: confirm independent deduplication; never enable automatic third-party issue publishing without trust gates.

## Verification in this patch environment

Source formatting, parsing, patch replay, and ZIP integrity can be checked locally. **Full Go tests, race testing, actual database execution, worker callbacks, and live staging integration are NOT VERIFIED here**, because the available Go 1.23.2 toolchain cannot obtain the required Go 1.26.11 toolchain. Do not merge as production-verified on static checks alone.
