# Operator runbook

Day-2 operations checklist for **Repository Detective**.

## Daily / weekly

1. Open `/ui/health` — confirm tools available, no actionable failed scans spike.
2. Open `/ui` dashboard — review open findings by severity and recent scans.
3. Check `/ui/learning` for calibration recommendations (accept per-repo or reject).
4. Review `/ui/findings?status=open` focus list for critical/high triage.

## After image or tool upgrades

1. Confirm `/health` → `tools_summary.missing` is `[]`.
2. Trigger a fresh scan on a known Go repo (product self-scan).
3. Verify latest scan summary: SBOM is not `sbom_tool_missing`; scanners are not stuck on `parse_failed` from known fixed parsers.
4. Re-publish wiki if operator docs changed: see [Wiki publishing](WIKI_PUBLISHING).

## Incident patterns

| Symptom | First check |
|---------|-------------|
| Auth 401 with Bugbot header | Use `X-Repository-Detective-API-Key` or Bearer |
| Dashboard “tools missing” high but health 12/12 | Historical metric; live readiness overlay should match health after upgrade |
| SBOM missing on Go repo | Ensure `syft` / `cyclonedx-gomod` on PATH; rescan |
| ShellCheck `parse_failed` | Flat JSON from ShellCheck 0.10 — upgrade to a build with flat-array parser |
| Forge outages mislabeled as scan failures | See [Troubleshooting](TROUBLESHOOTING) |

## Reference

- [Troubleshooting](TROUBLESHOOTING) — detailed fixes
- [Scanner health](SCANNER_HEALTH)
- [Admin hardening](ADMIN_HARDENING)
- [Privacy](PRIVACY_AND_DATA_PROTECTION)
- [Release readiness](RELEASE_READINESS)

---

See also [Home](Home).
