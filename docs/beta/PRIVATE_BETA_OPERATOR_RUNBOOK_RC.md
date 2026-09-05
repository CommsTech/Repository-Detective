# Private beta operator runbook (RC)

**Release:** `6d011cf` / live `rc-e3e19ec`  
**Audience:** Operators onboarding invited testers.

## Pre-flight checklist

- [ ] `main` at `6d011cf` or newer
- [ ] `make beta-release` succeeded
- [ ] `./scripts/check-beta-package-secrets.sh` pass
- [ ] `./scripts/operator-smoke-test.sh` pass
- [ ] Product dogfood: active-present **0** (scan `926a5f56a26f03c9` or latest)
- [ ] Remediation PR: **disabled**
- [ ] Runner delegation: **disabled**
- [ ] AI Recommendations: **disabled**
- [ ] Container scanning: **opt-in off**

## Onboard one tester

1. Send [PRIVATE_BETA_RC_RELEASE_NOTES.md](PRIVATE_BETA_RC_RELEASE_NOTES.md) + [PRIVATE_BETA_TEST_SCOPE.md](PRIVATE_BETA_TEST_SCOPE.md)
2. Provide beta package or homelab URL + API key (secure channel)
3. Assign **one repo** — tester must own it or have org approval
4. Set repo to **report-only** (see below)
5. Record: tester name, repo, date, expected scan ID slot for feedback

## Add one repo

**UI:** Configure → connect Gitea → add repository  
**API:** Connect webhook / register repo per `docs/SETUP.md`

Verify repo settings:

- Issue filing off or dry-run profile for first scan
- Scan profile: `beta_standard` or `standard_deterministic` recommended
- No runner policy until delegation explicitly enabled

## Force report-only

Options (pick one):

1. **Per-scan API:** `"report_only_dry_run": true` in analyze request
2. **Global:** `auto_create_issues: false` in config
3. **Repo profile:** disable filing in repo settings UI

Verify after scan:

```bash
# Reconciliation: forge_open_issues should be 0 for report-only
curl -H "X-Repository-Detective-API-Key: $KEY" \
  http://127.0.0.1:8081/api/v1/repos/{id}/reconciliation
```

## Run a scan

```bash
curl -X POST http://127.0.0.1:8081/api/v1/analyze \
  -H "X-Repository-Detective-API-Key: $KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "owner":"TESTER_OWNER",
    "repository":"TESTER_REPO",
    "ref":"main",
    "trigger_type":"manual",
    "analysis_depth":2,
    "report_only_dry_run":true,
    "enable_code_graph":true
  }'
```

Wait for: scan complete → persistence → graph → issue sync → reconciliation.

## Scanner / SBOM expectations

Explain to testers before first scan:

- **binary_missing** on trivy/grype/gitleaks/semgrep means that engine did not run — not a clean bill of health.
- **sbom_tool_missing** with `requirements.txt` present means Syft is absent; recommend full image or install `syft`.
- First-scan volume can be high on graph-heavy repos; scan detail shows **actionable** vs **grouped informational** counts.

## Review findings

- UI: `/ui/repos/{id}` → latest scan → findings list
- Finding detail: actionable sections, calibration hints
- Reconciliation panel: active-present vs forge issues explained

## Export a report

- Executive report from UI if enabled
- Or API/dashboard summary for operator review

## Collect feedback

Preferred path (structured, fixable issues):

1. Run **report-only** scan and capture **scan ID**
2. Review findings in UI (detail pages include fingerprint + template links)
3. For each item, open the matching Gitea template on `commstech/Repository-Detective`:
   - General feedback → `beta_feedback`
   - False positive → `scanner_false_positive`
   - Missed detection → `missed_detection`
   - UI/docs/scanner bugs → matching template under `.gitea/ISSUE_TEMPLATE/`
4. **Never paste secrets** — redact tokens, `.env`, PHI/PII
5. Include finding URL/fingerprint and scan ID in every report

Docs fallback: [PRIVATE_BETA_FEEDBACK_TEMPLATE.md](PRIVATE_BETA_FEEDBACK_TEMPLATE.md) · triage policy [../triage/ISSUE_TRIAGE_POLICY.md](../triage/ISSUE_TRIAGE_POLICY.md)

Store feedback record: `{tester, scan_id, date, category, disposition}`.

## Avoid issue filing (default)

| Feature | Ensure |
|---------|--------|
| `auto_create_issues` | false |
| First scan | `report_only_dry_run: true` |
| Pre-install | report-only by design |
| Container scan | `create_issues: false` |
| GitHub provider | not release-proven — keep off |
| GitLab | not implemented |

## Roll back tester access

1. Disable webhook or remove repo from scheduler
2. Revoke API key or rotate `REPOSITORY_DETECTIVE_API_KEY`
3. Optional: delete tester repo row after export if needed
4. Confirm no open forge issues were created (should be 0 for report-only)

## Check logs

```bash
./scripts/operator-log-health-check.sh
docker logs repository-detective --since 1h 2>&1 | tail -50
```

Look for: panics, token leaks, unexpected issue creation.

## Check active-present

```sql
-- Latest scan for repo_id N
SELECT COUNT(DISTINCT f.id) FROM findings f
JOIN finding_instances fi ON fi.finding_id = f.id AND fi.scan_id = '<scan_id>'
WHERE f.repository_id = N AND f.status = 'open';
```

Or use reconciliation API / UI panel.

## Verify no secrets committed

- Never commit tester `.env`, `data/repository-detective.db`, or logs with tokens
- Run `./scripts/check-beta-package-secrets.sh` before distributing bundle

## Disable tester access

- Remove repo connection
- Rotate API key
- Document in feedback log: `access_revoked`

## Risky features — explicit disabled state

| Feature | RC default | Enable only when |
|---------|------------|------------------|
| Remediation PR | off | Never in private beta |
| Runner delegation | off | Controlled container demo window |
| AI Recommendations | off | Operator + tester agree; token budget set |
| Container scanning | opt-in | Owned image, non-prod host |
| Issue filing | off / report-only | After Gitea scratch proof |
| Gitea Actions backend | off | Not in beta |

## Every beta scan should have

- **Report ID / scan ID** recorded
- **Feedback template** filed or N/A with reason
- **0 forge issues** if report-only

## References

- [PRIVATE_BETA_TEST_SCOPE.md](PRIVATE_BETA_TEST_SCOPE.md)
- [ISSUE_FILING_POLICY.md](ISSUE_FILING_POLICY.md)
- [gitea-filing-controlled-test-plan.md](../dogfood-reports/gitea-filing-controlled-test-plan.md)
