# G201: SQL query constructed with fmt.Sprintf in learning_sqlite.go

## Summary

gosec `G201` detected in `learning_sqlite.go`.

**Severity:** Medium  
**Detection confidence:** 90%  
**First seen:** Sep 8, 2026  
**Last seen:** Sep 8, 2026

## Why this matters

Formatting values into a SQL string (for example with `fmt.Sprintf`) can permit SQL injection if any formatted fragment can contain untrusted syntax.

In many Go codebases the formatted fragment is only a list of `?` placeholders while runtime values stay in `ExecContext`/`QueryContext` args. Verify how the formatted fragment is generated before treating this as exploitable. If it is only trusted placeholders, classify as a false positive and feed that into Repository Detective calibration.

## Location

`store/learning_sqlite.go:473`

[Open in forge](https://git.commsnet.org/commstech/Repository-Detective/src/commit/4aa38593deadbeef/store/learning_sqlite.go#L473)

Commit: `4aa38593deadbeef`

## Evidence

```go
query := fmt.Sprintf(`DELETE FROM learning_events WHERE id IN (%s)`, placeholders)
```

## Recommended action

Determine whether the formatted SQL fragment can contain anything other than internally generated placeholders.

* If untrusted content can reach that fragment, replace the construction with a safe parameterized approach.
* If it consists only of trusted placeholder tokens and all values remain bound through query args, classify this scanner result as a false positive and feed that outcome into Repository Detective calibration.

## Verification

Run the relevant project tests and re-scan with the same scanner.

Repository Detective will consider this resolved only when the fingerprint no longer appears, or when it has been explicitly triaged as false positive or accepted risk.

<details>
<summary>Repository Detective details</summary>

- Scanner: gosec
- Rule: `G201`
- Fingerprint: `rd-dogfood-g201`
- Scan: `dogfood-scan-1`
- Finding ID: `1201`
- Fix complexity: medium
- Safe for auto PR: false

</details>
