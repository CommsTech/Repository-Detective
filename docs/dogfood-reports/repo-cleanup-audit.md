# Repository cleanup audit

**Date:** 2026-06-05  
**Scope:** `commstech/Repository-Detective` working tree — classify operator leftovers vs product artifacts.

---

## git status

Clean after `ab97c40` push (pre this audit’s doc-only changes).

---

## Sensitive / local-only findings

| Path | Classification | Action |
|------|----------------|--------|
| `.env` | local_only | Already in `.gitignore` — never commit |
| `config/config.yaml` | local_only | Already gitignored |
| `data/repository-detective.db` | local_only | Already gitignored (`data/`, `*.db`) |
| `deployment-backups/` | local_only | Already gitignored |
| `restore-drill-test/` | local_only | Already gitignored |
| `deployment-backups/*/repository-detective.db` | local_only | Gitignored via parent |
| `restore-drill-test/data/repository-detective.db` | local_only | Gitignored |
| `redact/secrets.go` | keep | Source — not a secret file |

---

## Keep (product)

```text
application source (Go packages)
ui/static/logo.svg          # product logo (no logo.png in repo today)
docs/                       # operator + design docs
scripts/                    # operator scripts (vendor-deps, smoke tests, deploy)
docker/deploy assets        # Dockerfile, compose files
tests                       # *_test.go across packages
config/config.yaml.example
.env.example
```

---

## Commit (sanitized dogfood reports)

Already committable via `.gitignore` exceptions:

| Report | Purpose |
|--------|---------|
| `issue-resolution-sprint-plan.md` | Batch strategy |
| `current-security-blocker-verification.md` | P0 re-verification (this sprint) |
| `repo-cleanup-audit.md` | This file |
| `issue-resolution-batch-plan-current.md` | Current batch plan |

Other `docs/dogfood-reports/*` remain **local_only** unless explicitly allowlisted.

---

## Delete (operator action — not automated)

Do not delete from this task (may contain drill data the operator wants):

```text
deployment-backups/         # old DB snapshots — prune when no longer needed
restore-drill-test/         # drill artifact
data/repository-detective.db              # production DB on host — backup before delete
```

---

## gitignore updates (this sprint)

Added allowlist entries for:

- `current-security-blocker-verification.md`
- `repo-cleanup-audit.md`
- `issue-resolution-batch-plan-current.md`

Already ignored: `.env`, `data/`, `*.db`, `deployment-backups/`, `vendor/`, private triage scripts.

---

## needs_review

| Item | Notes |
|------|-------|
| `vendor/` | Generated offline — not committed; correct for supply-chain policy |
| `docs/DOGFOOD_REPORT_FIRST_39_REPOS.md` | Gitignored — may contain fleet-specific detail |
| `ui/static/logo.png` | Referenced in README template — **not present**; use `logo.svg` until PNG exported |
| Root binaries `/repository-detective` | Gitignored build outputs |

---

## Recommended operator cleanup (manual)

```bash
# After backup:
# rm -rf deployment-backups/ restore-drill-test/
# docker image prune -f   # if disk constrained (see TROUBLESHOOTING)
```

Do **not** remove application source, tests, sanitized docs, or config examples.
