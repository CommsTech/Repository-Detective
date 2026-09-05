# Why findings are issues (not PR comment spam)

Repository Detective deliberately keeps **each finding in one canonical lifecycle record** (typically a Gitea issue, plus the local finding row).

That design enables:

- **Deduplication** by fingerprint across pushes and PRs
- **History** and recurrence when the same issue returns
- **Evidence** and scanner provenance
- **False-positive / suppression** disposition
- **Remediation planning** and closure/reopen state
- **Reconcile** against forge issues without deleting audit trails

Pull requests receive a **compact policy summary** (one upserted comment marked `repository-detective-policy-summary`), not one bot comment per finding.

Absence of inline review-comment spam is therefore an intentional product choice — not a missing feature.

Related: [POLICY.md](POLICY.md) · [ISSUE_RECONCILIATION.md](ISSUE_RECONCILIATION.md) · [FINDING_RESOLUTION_SEMANTICS.md](FINDING_RESOLUTION_SEMANTICS.md) · [PR_SUMMARY_IDEMPOTENCY.md](PR_SUMMARY_IDEMPOTENCY.md).
