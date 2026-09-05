# Batch 4b baseline

Generated: 2026-06-02 (Sprint 3 closeout)

## Repository state

| Item | Value |
|------|-------|
| Latest commit | `600d0a7` — docs(dogfood): verify batch 4a active fixes |
| Branch | `main` |
| `.env` staged | no |
| `repository-detective` ELF staged | no |
| Backlog-control | active (`dogfood_backlog_control_enabled: true`) |
| Latest scan | `db2d7061eaac8eb0` (1093 instances) |

## Issue counts (pre–batch 4b)

| Metric | Count |
|--------|------:|
| Gitea open | 57 |
| Real active (present in latest scan) | 11 |
| Resolved-absent (open, absent from scan) | 14 |
| Needs human review | 2 |
| Out of scope (summaries) | 30 |

## CI

| Item | Value |
|------|-------|
| Run | #116 (id 1868) |
| SHA | `600d0a7` |
| Status | completed **failure** |
| Likely failing step | Build Docker image (`docker build --target core`) — Alpine apk/user flake caused `chown: unknown user/group repositorydetective` when `adduser` did not persist across cached layers |

## Docker

| Item | Value |
|------|-------|
| Partial rebuild | working (builder, apk-retry) |
| Full rebuild | **broken** — core stage `chown` on missing `repositorydetective` user after apk flake |
| Fix planned | `scripts/docker-alpine-runtime-setup.sh` — deterministic apk + user creation with hard fail + verify |

## Active findings (11)

| Issue | Rule | Path |
|------:|------|------|
| #345 | TRIVY-MIS-DS011 | Dockerfile multi-arg COPY |
| #53 | REL-INTERNAL-INFRA-REF | Makefile:175 |
| #66 | REL-INTERNAL-INFRA-REF | deploy.ps1:52 |
| #143–#145 | REL-INTERNAL-INFRA-REF | preinstall/url.go |
| #280 | REL-INTERNAL-INFRA-REF | patcher/git.go:137 |
| #296 | REL-INTERNAL-INFRA-REF | deploy/nginx-repository-detective.conf.example |
| #321 | G201 | store/findings_batch_sqlite.go |
| #324 | G203 | ui/ui_helpers.go |
| #332 | G304 | scanners/archive_extract.go |

## Resolved-absent candidates (14)

#206, #228, #232, #259, #260, #261, #262, #312, #322, #326, #327, #328, #330, #331

## Prior batches

Batch 2, 3a, 3b, 3c, 3d, 4a — verified at sprint start.
