# FAQ

Common operator questions for **Repository Detective**.

## Auth

**How do agents and scripts authenticate?**  
Use `X-Repository-Detective-API-Key: <key>` or `Authorization: Bearer <key>`. Legacy `X-Bugbot-API-Key` is rejected with **401**.

**Where is the OpenAPI spec?**  
`GET /api/v1/openapi.yaml` (authenticated) or [docs/openapi.yaml](https://git.commsnet.org/commstech/repository-detective/src/branch/main/docs/openapi.yaml) in the main repo.

## Scans

**How do I list scans?**  
There is no fleet-wide `GET /api/v1/scans`. Use `GET /api/v1/repos/:id/scans` or the UI at `/ui/scans`.

**Why does System Health say 12/12 tools available but an old scan shows tools missing?**  
Tool availability is live. Scan summaries are historical. Re-run a scan after installing binaries (for example Syft / cyclonedx-gomod) to refresh SBOM and scanner results.

**What is report-only mode?**  
Scans can run without filing forge issues. See [Report-only scans](REPORT_ONLY_SCANS).

## Findings & learning

**How does false-positive learning work?**  
Use `/ui/learning` or the `/api/v1/calibration/*` endpoints. Acceptances are **repo-scoped**; security/secret categories stay Reject-only. See [Learning and calibration](LEARNING_AND_CALIBRATION).

**Is there a `/api/v1/learning/*` API?**  
No. Learning is UI plus calibration API.

## Health & SBOM

**Where do I check scanner binaries?**  
`/ui/health` and `GET /api/v1/status` / `/health` (`tools_summary`). See [Scanner health](SCANNER_HEALTH) and [SBOM](SBOM).

## More help

- [Troubleshooting](TROUBLESHOOTING)
- [Operator runbook](OPERATOR_RUNBOOK)
- [Configuration](CONFIGURATION)

---

See also [Home](Home).
