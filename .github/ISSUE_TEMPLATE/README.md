# GitHub community issue templates

**Public feedback path (RD-001):** anonymous and public-beta users should file bugs and feature requests on **GitHub Issues**.

Canonical development remains on **Gitea** (`git.commsnet.org/commstech/Repository-Detective`). Maintainers may mirror or triage GitHub issues there.

## Templates on this mirror

| Template | Use for |
|----------|---------|
| `bug_report.yml` | Incorrect product behavior |
| `feature_request.yml` | Enhancements |
| `installation_problem.yml` | Compose / ports / env / first start |
| `scanner_problem.yml` | Scanner unavailable / timeout / parse failure |
| `scanner_false_positive.yml` | Noisy or wrong finding |
| `beta_feedback.yml` | General public-beta UX feedback |

Security vulnerabilities → [SECURITY.md](../../SECURITY.md) / [private advisory](https://github.com/CommsTech/Repository-Detective/security/advisories/new) — **not** these forms.

Richer maintainer templates (ops, accessibility, SBOM, …) live under `.gitea/ISSUE_TEMPLATE/`.
