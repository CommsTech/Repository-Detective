# Non-product repo dry-run plan

Generated: 2026-06-07  
Status: **PLAN ONLY** — do not execute without operator approval.

## Objective

Validate Repository Detective against 1–2 non-product repos in **report-only** mode before any fleet issue filing.

## Candidate selection criteria

| Criterion | Requirement |
|-----------|-------------|
| Size | 1 small (<50 files), 1 medium (50–500 files) |
| Ownership | Same Gitea org (`commstech`), non-Repository-Detective |
| Risk | No production secrets; no customer data |
| Familiarity | Operator knows expected layout (Go, shell, or docs-only) |
| Exclusions | No fleet repos, no archived/mirror repos |

**Suggested candidates (operator to confirm):**

1. Small: a docs-only or single-service homelab repo
2. Medium: a Go utility repo with `go.mod` and CI

## Dry-run configuration

| Setting | Value |
|---------|-------|
| Mode | analyze + persist findings **only** |
| Issue filing | **disabled** (`issue_policy=report_only` or forge filing off) |
| Backlog-control | **enabled** (remain on) |
| Max issue creation | **0** |
| AI / LLM auditors | **disabled** |
| Auto-remediation | **disabled** |
| Scan profile | `standard_deterministic` |
| Workspace mode | `archive` (default) |

## Runtime limits

| Limit | Value |
|-------|-------|
| Max concurrent scans | 1 |
| Per-repo timeout | 10 minutes |
| Max findings persisted | 2000 per repo |
| Max runtime (total dry-run) | 30 minutes |

## Scanner baseline

Record `scanner_results` per scan. Treat timeouts (gosec, staticcheck, hadolint) as variance — not product regressions.

## Verification checklist (per repo)

- [ ] Scan completes; persistence `complete`
- [ ] `issue_sync` reaches `complete` or `skipped` (not stuck `pending`)
- [ ] **0** Gitea issues created
- [ ] **0** duplicate issue burst
- [ ] DB size delta acceptable (`du -h data/repository-detective.db`)
- [ ] Findings match repo structure (no false mass criticals)
- [ ] Export report for operator review

## Rollback / stop conditions

Stop dry-run immediately if:

- Any Gitea issue is created
- Open issue count increases on non-target repo
- DB growth > 500 MB in one session
- Scanner crash loops or OOM
- Duplicate fingerprint filing detected

**Rollback:** disable scheduler; delete dry-run scan rows if needed; do **not** mass-close issues.

## Operator approval gate

Before running:

1. Confirm candidate repo list (1 small + 1 medium)
2. Confirm issue filing disabled in config
3. Confirm backlog-control enabled
4. Record approval timestamp in operator log

## Execution command (when approved)

```bash
# Example — replace owner/repo; verify filing disabled first
curl -X POST http://localhost:8081/api/v1/analyze \
  -H "X-Repository-Detective-API-Key: $REPOSITORY_DETECTIVE_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{"owner":"commstech","repository":"<CANDIDATE>","ref":"main"}'
```

## Post dry-run

- Compare findings to manual expectations
- Document scanner variance
- Update `all-gitea-repos-scan-readiness.md`
- **Do not** enable fleet filing without second operator sign-off
