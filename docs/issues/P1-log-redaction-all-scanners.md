# Extend scanner log redaction to all scanners

**Labels:** `type/privacy`, `type/scanner`, `priority/p1`, `status/needs-triage`  
**Milestone:** Sprint 3 - Privacy and Compliance Readiness

## Summary

`trivy` uses `logResultInfo` with `internal/security.RedactLogField`. Other scanners still log `detail=%q` with raw stderr in some paths.

## Acceptance criteria

- [ ] All scanner `Infof`/`Warnf` paths use `scanners/logResultInfo` or `security.RedactLogField`
- [ ] Unit test in `internal/security/redact_test.go` remains green
- [ ] No token-like strings in sample log output from integration tests

## Evidence

- `scanners/log_redact.go` added in closeout sprint
- `scanners/semgrep.go`, `gosec.go`, etc. still log raw `result.Detail` in places
