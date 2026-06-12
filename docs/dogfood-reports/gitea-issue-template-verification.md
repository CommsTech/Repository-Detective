# Gitea issue template verification

**Date:** 2026-06-02  
**Repo:** `commstech/Bugbot`  
**Commit under test:** (pending push — templates in `.gitea/ISSUE_TEMPLATE/`)

## Summary

| Check | Result |
|-------|--------|
| Templates in repo | **yes** — 14 YAML form templates + `config.yml` |
| `config.yml` present | **yes** — `blank_issues_enabled: false` |
| Templates visible in Gitea UI | **pending** — verify after push to `main` |
| Blank issues allowed | **expected: no** (if Gitea version supports `blank_issues_enabled`) |
| Test issue created | **no** — controlled verification only |
| Test issue closed | **n/a** |

## Templates detected (repository)

| Template file | Purpose |
|---------------|---------|
| `bug_report.yaml` | General bug report |
| `beta_feedback.yaml` | Beta feedback |
| `scanner_false_positive.yaml` | False positive |
| `missed_detection.yaml` | Missed detection |
| `scanner_parser_bug.yaml` | Scanner/parser bug |
| `ui_ux_issue.yaml` | UI/UX issue |
| `docs_gap.yaml` | Documentation gap |
| `security_triage.yaml` | Security finding review |
| `container_scan_issue.yaml` | Container scan |
| `sbom_issue.yaml` | SBOM issue |
| `preinstall_audit_issue.yaml` | Pre-install audit |
| `feature_request.yaml` | Feature request |
| `operator_task.yaml` | Operator task |
| `compliance_privacy.yaml` | Compliance/privacy |
| `accessibility.yaml` | Accessibility |
| `config.yml` | Blank issue gate + contact links |

## config.yml behavior

```yaml
blank_issues_enabled: false
```

If the live Gitea instance ignores this key (older versions), Markdown/YAML templates still work; blank issues may remain available — document limitation in operator runbook.

## Manual UI verification steps (post-push)

1. Open https://git.commsnet.org/commstech/Bugbot/issues/new
2. Confirm template picker lists beta/scanner templates
3. Confirm blank issue is discouraged or disabled
4. Open `scanner_false_positive` and `beta_feedback` — verify fields render
5. Do **not** submit test issues unless operator approves controlled test

## Gitea version limitation

Wiki HTTP 500 is a separate blocker; issue templates use repository files and may work independently. Confirm on live instance after deploy.

## Evidence

- Repository path: `.gitea/ISSUE_TEMPLATE/`
- UI links added: finding detail (false positive), scan detail (beta feedback), learning page
- Triage docs: `docs/triage/ISSUE_TRIAGE_POLICY.md`, `docs/triage/LABEL_TAXONOMY.md`
