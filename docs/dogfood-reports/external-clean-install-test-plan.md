# External clean install — test plan

**Status:** `not_run` (beta package partial proof exists)  
**Marketing blocker:** Yes — full VM proof required before public marketing  
**Private beta blocker:** No — homelab + beta package sufficient for invited cohort

## Prior partial proof (2026-06-11)

| Step | Result |
|------|--------|
| Fresh tree copy (exclude data/.git/dist) | PASS |
| `cp .env.example .env` | PASS |
| `make beta-release` | PASS → `dist/repository-detective-beta/` |
| Package contents | PASS |
| `docker build --target all-in-one` on clean copy | not completed on shared host |
| Alternate port + Gitea connect + scan | deferred |

## Full test plan (when clean VM available)

### Environment

- **Host:** Clean VM or container (document OS, arch, Docker version)
- **Network:** Can reach Gitea forge; no existing RD install
- **Secrets:** Local `.env` only; never commit

### Steps

| # | Step | Pass criteria |
|---|------|---------------|
| 1 | Clone `https://git.commsnet.org/commstech/Repository-Detective.git` at `6d011cf+` | Clone succeeds |
| 2 | `cp .env.example .env` — set API key, Gitea URL/token | No secrets in git |
| 3 | `docker build --target all-in-one -t repository-detective:all-in-one .` | Image builds |
| 4 | `docker compose -f docker-compose.beta.yml up -d` | Container healthy |
| 5 | `curl http://127.0.0.1:8081/health` | `status=healthy` |
| 6 | Login / API key unlock UI | Dashboard loads |
| 7 | Connect Gitea + add **one owned test repo** | Repo connected |
| 8 | Report-only scan | Scan completes; 0 issues filed |
| 9 | View findings page | HTTP 200 |
| 10 | View SBOM page | Empty or artifact; honest state |
| 11 | View pre-install page | Loads |
| 12 | Docs links from UI/README | Resolve |
| 13 | `./scripts/operator-smoke-test.sh` | Pass |
| 14 | Collect logs | No panics; no token leaks |
| 15 | Stop container + cleanup | Clean uninstall documented |

### Artifacts to produce

Update [external-clean-install-report.md](external-clean-install-report.md) with:

- VM spec
- Timings (clone, build, first scan)
- Scan ID
- Screenshot optional (no secrets)
- Log summary

## If not run

Mark marketing readiness **NOT READY**. Private beta may proceed with operator-prepared homelab or hand-delivered beta package.

## References

- `docs/SETUP.md`
- `README_BETA.md` in beta package
- [PRIVATE_BETA_OPERATOR_RUNBOOK_RC.md](../beta/PRIVATE_BETA_OPERATOR_RUNBOOK_RC.md)
