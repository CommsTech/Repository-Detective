# Private beta tester scope

**Release:** RC private beta (`6d011cf`, live `rc-e3e19ec`)  
**Principle:** Start narrow, report-only, owned repos only.

## Allowed (first cohort)

| Action | Notes |
|--------|-------|
| Connect **one repo** per tester initially | Operator adds repo or approves self-connect |
| **Report-only scan first** | `report_only_dry_run: true` or filing disabled in repo settings |
| Pre-install audit of **public HTTPS repos** | Report-only; no forge side effects |
| Review findings pages | Including detail, severity, confidence, calibration hints |
| Review SBOM pages | Empty state or artifact when available |
| Review repository map / graph | When graph enabled on scan |
| Submit feedback | Using [Gitea issue templates](https://git.commsnet.org/commstech/repository-detective/issues/new) (`.gitea/ISSUE_TEMPLATE/`) or docs in `docs/beta/` |
| View configure / health pages | Read-only config review |
| Export executive report | Browser print/PDF when available |

## Not allowed without operator approval

| Action | Why |
|--------|-----|
| All-repo / bulk scanning | Out of beta scope; noise and trust risk |
| Issue filing | Live Gitea proof pending; start report-only |
| Remediation PR | Disabled globally |
| AI Recommendations | Disabled by default; advisory only |
| Runner delegation | Disabled; opens container scan surface |
| Container scanning against **production Docker hosts** | Opt-in; controlled windows only |
| Third-party disclosure submission | Never automatic |
| Scanning repos tester does not own | Legal/trust boundary |
| Enabling GitHub issue filing | Not release-proven |
| GitLab issue filing | Not implemented |

## Ideal beta repos

| Criterion | Guidance |
|-----------|----------|
| Size | Small to medium (< ~2k files ideal) |
| Sensitivity | Non-sensitive; **no PHI/PII**, no customer secrets |
| Ownership | **Owned by tester** or explicit org approval |
| Manifests | Prefer Dockerfile, `go.mod`, `package.json`, or Python manifest |
| Forge | **Gitea preferred** for first cohort |
| Languages | Go, shell, YAML, markdown docs OK; monorepos OK with patience |
| Avoid | Production creds in tree, `.env` committed, regulated data |

## Scan progression (recommended)

```text
1. Operator onboards tester → one repo connected
2. Tester runs report-only scan → verify 0 issues filed
3. Tester reviews findings + SBOM + graph
4. Tester submits feedback template
5. Operator triages → calibration / config / bug
6. Optional: second report-only scan after operator adjustment
7. Only with operator approval: issue filing trial on scratch repo
```

## Scanner coverage (slim image)

The default beta Docker image includes **deterministic** scanners (static, health, graph, linters) but may **not** bundle every optional binary.

| Status | Meaning for testers |
|--------|---------------------|
| **found / clean** | Tool ran on this scan |
| **binary_missing** | Tool was **not available** in this runtime — repo was **not** scanned by that engine. This does **not** mean the repo is clean. |
| **sbom_tool_missing** | Dependency manifest detected (e.g. `requirements.txt`) but **Syft** is unavailable — no SBOM was generated. Install Syft or use the full scanner image. |

See Configure → Health for installed vs configured tools. First scans on small homelab repos may show **many low/info findings** (graph + debug heuristics) — use severity filters and grouped informational summary on the scan page.

## Feedback expectations

Every beta session should produce:

- **Scan ID** (from UI or API)
- **Feedback record** (template filled)
- **Confirmation:** no secrets in attachment

False positives must use [PRIVATE_BETA_FALSE_POSITIVE_TEMPLATE.md](PRIVATE_BETA_FALSE_POSITIVE_TEMPLATE.md).

## Out of scope for testers

- Public marketing or social posts
- Comparing to commercial tools in public forums
- Sharing unredacted scan output externally
- Running against repos they do not control

## Operator escalation

If tester needs filing, container scan, runner, or AI Recommendations → operator follows [PRIVATE_BETA_OPERATOR_RUNBOOK_RC.md](PRIVATE_BETA_OPERATOR_RUNBOOK_RC.md) and explicit approval checklist.
