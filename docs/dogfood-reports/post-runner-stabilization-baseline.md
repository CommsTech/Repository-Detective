# Post-runner stabilization baseline

Recorded: 2026-06-09  
Latest commit: `9055547` (fix(runner): satisfy staticcheck on execute input conversion)

## Repository state

| Check | Status |
|-------|--------|
| `.env` staged | no |
| Runner secret staged | no |
| Local binary staged | no |
| `dist/` staged | no |
| Working tree | clean at baseline |

## Live deployment

| Setting | Value |
|---------|-------|
| Container | `repository-detective` healthy |
| Live revision | `abb3eaa` (container not yet rebuilt to `9055547`) |
| Runner delegation | **disabled** (`runner_delegation_enabled: false`) |
| Remediation PR | **disabled** |
| Gitea Actions backend | disabled |
| All-repo scanning | off |

## Product repo (Repository-Detective)

| Metric | Value |
|--------|-------|
| Open Gitea issues | **1** (#48 operator task) |
| Active-present (scan `f6102e4fed8e2b37`) | **89** |
| High/critical | **0** |
| Latest scan ID (live DB) | `2463e276e8a2b979` (0 files — invalid; use rescan) |
| Last good scan | `f6102e4fed8e2b37` (890 files, 89 findings, graph available) |

## Runner worker docs

- `docs/RUNNER_WORKER_QUICKSTART.md`
- `docker-compose.runner.example.yml`

## Test window plan

1. **Phase 1** — Classify all 89 active-present findings (`remaining-active-present-classification.md`).
2. **Phase 2** — Bounded health/static calibration batch (skip docs, tests, rule self-match; allow best-effort ignored errors).
3. **Phase 3** — Single product rescan with graph enabled; verify active-present decrease.
4. **Phase 4** — Native runner soak (graph → SBOM → remediation_verify); disable delegation after test.
5. **Phase 5** — Remediation PR controlled gate (`not_approved` default).
6. **Phase 6** — Gitea wiki HTTP 500 diagnostics update (non-blocking).
7. **Phase 7** — Full verification, beta-release, push.

## Remediation PR state

- Global remediation PR: **disabled**
- Controlled test PR: **not approved** (gate doc in Phase 5)

## Remaining blockers

- Gitea wiki HTTP 500 (server-side)
- Operator issue #48 (homelab AI/Qdrant connectivity)
- 89 active-present findings pending calibration/rescan
- Gitea act_runner token rotation if earlier token was real
