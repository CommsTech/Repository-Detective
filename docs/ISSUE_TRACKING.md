# Issue and feature tracking

Repository Detective uses **Gitea** for product backlog (`commstech/Repository-Detective`). Scan findings use separate `repository-detective/*` labels via the issue manager.

## Templates

Issue forms live in [.gitea/ISSUE_TEMPLATE/](../.gitea/ISSUE_TEMPLATE/):

| Template | Use |
|----------|-----|
| `bug_report.yaml` | Product bugs |
| `feature_request.yaml` | Enhancements |
| `compliance_privacy.yaml` | Privacy/compliance-readiness gaps |
| `accessibility.yaml` | WCAG-aligned UI barriers |
| `scanner_false_positive.yaml` | Scanner noise tuning |
| `security_triage.yaml` | Triage of scan findings |

## Recommended Gitea labels

Product backlog labels (create on the **repository-detective** product repo, not finding issues):

- `type/bug`, `type/feature`, `type/docs`, `type/compliance`, `type/privacy`, `type/accessibility`, `type/security`, `type/scanner`, `type/false-positive`, `type/ui`, `type/api`, `type/reporting`
- `severity/critical` … `severity/low`
- `status/needs-triage`, `status/accepted`, `status/in-progress`, `status/blocked`, `status/ready-for-test`, `status/done`
- `priority/p0` … `priority/p3`

Finding issues continue to use `repository-detective`, `repository-detective/security`, `severity/*`, lifecycle labels — see [issues/labels.go](../issues/labels.go).

## Milestones (closeout sprints)

| Milestone | Scope |
|---------|--------|
| Sprint 1 - Issue and Feature Backlog | Templates, labels, backlog |
| Sprint 2 - Accessibility | WCAG-aligned UI |
| Sprint 3 - Privacy and Compliance Readiness | Data protection docs + safer handling |
| Sprint 4 - Repository Detective Self-Scan | Dogfood |
| Sprint 5 - Bug and Feature Closeout | P0/P1 fixes |
| Sprint 6 - Release Readiness | Evidence + release notes |

## Prepared backlog (not auto-created)

Markdown issue specs: [issues/](issues/README.md). Create on Gitea with:

```bash
./scripts/gitea-backlog-setup.sh --labels-only   # labels + milestones (requires GITEA_TOKEN in .env)
./scripts/gitea-backlog-setup.sh --issues        # documents manual create only (no bulk API)
```

**Closeout evidence:** When `GITEA_TOKEN` is present in `.env`, `gitea-backlog-setup.sh --labels-only` created `type/*`, `severity/*`, `status/*`, `priority/*` labels and Sprint 1–6 milestones on `commstech/Repository-Detective` (verified via Gitea API list). Individual backlog **issues** were not bulk-created.

## Scan finding issues vs product issues

| | Product (Gitea repo) | Finding (per scanned repo) |
|--|----------------------|----------------------------|
| Created by | Humans / closeout script | `issues.Manager` after scans |
| Labels | `type/*`, `status/*` | `repository-detective/*` |
| Dedup | Manual | Fingerprint + SQLite forge mappings |

See [ISSUE_BACKLOG.md](ISSUE_BACKLOG.md) for historical Gitea issue numbers.
