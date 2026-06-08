# Issue and finding reconciliation

Repository Detective tracks **findings** (scan evidence) separately from **forge issues** (Gitea/GitHub tickets).

## Key concepts

| Term | Meaning |
|------|---------|
| **Finding** | A persisted security/quality item with fingerprint, severity, and scan evidence |
| **Active-present** | Finding still appears in the latest completed scan |
| **Forge issue** | A Gitea/GitHub issue created (or linked) from a finding |
| **external_issues** | Database mapping between a finding and a forge issue number |

## Findings without forge issues

This is normal when:

- **Report-only / dry-run** scan — findings stored, no issues filed
- **Backlog control** or **max issues per scan** capped filing
- **Severity/confidence gates** filtered the finding out of filing
- **Pre-install audit** — always report-only
- Finding is **informational** or suppressed by learning/calibration

## Findings mapped to a specific issue

When filing is enabled, new issues include a **Repository Detective fingerprint** in the body. The reconciliation panel shows:

- Active-present findings
- Open forge issues tracked in `external_issues`
- Drift when Gitea issues were closed manually but DB rows stayed open

Use **Reconcile issues** on the repo detail page (or `POST /api/v1/repos/:id/reconcile-issues`) to sync state.

![Reconciliation panel](../assets/screenshots/reconciliation.png)

## Stale `external_issues` rows

If issues were closed in Gitea outside Repository Detective, the DB may still show them as open. A product rescan plus reconciliation (or `scripts/product-repo-resync.py`) repairs mappings without suppressing real findings.

## Related

- [REPORT_ONLY_MODE](REPORT_ONLY_MODE.md)
- [ENABLE_ISSUE_FILING](ENABLE_ISSUE_FILING.md)
- `docs/ISSUE_RECONCILIATION.md`
