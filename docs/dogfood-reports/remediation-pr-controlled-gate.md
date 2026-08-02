# Remediation PR controlled gate

**Decision state:** `not_approved`

Operator must change state to `approved_for_one_test_pr` before any PR is created.

---

## Candidate test repo

| Field | Value |
|-------|-------|
| Owner | `commstech` |
| Repository | `Repository-Detective` (product — use only with explicit approval) |
| Alternative | dedicated owned test repo (preferred for first live PR) |

## Candidate finding (low severity)

| Field | Value |
|-------|-------|
| Rule | `HEALTH-IGNORED-ERROR` |
| Severity | low |
| Example path | `main.go` (type assertion discard) |
| Fingerprint | from latest scan `4f8617f80f1ef1e8` |

## Proposed remediation

Handle or explicitly document the ignored error at the call site (minimal diff, single file).

## Expected diff size

≤ 15 lines (1 file)

## Test command

```bash
go test ./...
go vet ./...
GOFLAGS=-buildvcs=false staticcheck ./...
./scripts/operator-smoke-test.sh
```

## Runner / native verification

```bash
# Optional: remediation_verify delegated job after patch branch exists
curl -X POST -H "X-Repository-Detective-API-Key: $API_KEY" \
  -H "Content-Type: application/json" \
  -d '{"repository_id":1,"job_type":"remediation_verify"}' \
  http://127.0.0.1:8081/api/v1/runner/jobs/enqueue-delegated
```

## PR safety checks

- [ ] Remediation PR globally still gated (`remediation_pr_enabled: false` until test window)
- [ ] Single PR only; no bulk filing
- [ ] Dry-run remediation plan reviewed
- [ ] Branch name and target ref confirmed
- [ ] Rollback: close PR + revert commit if verify fails

## Rollback plan

1. Close Gitea PR without merge.
2. Delete remote branch.
3. Set decision state back to `not_approved`.
4. Rescan product repo to confirm finding state.

## Config needed (test window only)

```env
REPOSITORY_DETECTIVE_REMEDIATION_PR_ENABLED=true
# plus existing Gitea token with repo write + PR scope
```

## Approval checkbox

- [ ] **I approve exactly one controlled Remediation PR test** (operator initial: ______ date: ______)

## Decision states

| State | Meaning |
|-------|---------|
| `not_approved` | **current** — no PR |
| `approved_for_one_test_pr` | operator approved; agent may create one PR |
| `completed` | test PR merged or closed with report |
| `blocked` | dependency or safety blocker |
