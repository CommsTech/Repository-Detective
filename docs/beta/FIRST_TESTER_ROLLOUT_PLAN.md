# First tester rollout plan

Private beta — trusted testers only. Report-only first; no forge side effects without operator approval.

## Tester scope

| Constraint | Value |
|------------|-------|
| Testers | **1–3 trusted** individuals only |
| Repos per tester | **1** initially |
| First scan mode | **Report-only** (`report_only_dry_run: true`) |
| Issue filing | **Off** — do not enable |
| Remediation PRs | **Off** |
| LLM sanity gate | **Off** |
| All-repo scan | **Not allowed** |
| Global calibration accept | **Blocked** |

## Tester prerequisites

- Access to a self-hosted Gitea (or GitHub) instance
- API token with **read** repository scope minimum; write scope **not required** for report-only
- Docker 24+ **or** Linux amd64 for binary install
- One **non-production** test repository they own or operate
- Agreement to use [PRIVATE_BETA_FEEDBACK_TEMPLATE.md](PRIVATE_BETA_FEEDBACK_TEMPLATE.md)
- Agreement **not** to share `.env`, tokens, or database files

## Distribution

1. Operator sends `dist/repository-detective-beta/` via secure channel (not public git)
2. Include links to:
   - [PRIVATE_BETA_TESTER_GUIDE.md](PRIVATE_BETA_TESTER_GUIDE.md)
   - [PRIVATE_BETA_RELEASE_NOTES.md](PRIVATE_BETA_RELEASE_NOTES.md)
   - [PRIVATE_BETA_FEEDBACK_TEMPLATE.md](PRIVATE_BETA_FEEDBACK_TEMPLATE.md)
3. Provide operator contact for support and escalation

## First scan flow

1. Unpack beta bundle; verify `sha256sum -c checksums.txt`
2. Copy `config.example.yaml` → `config/config.yaml`
3. Create `.env` from `.env.example`; set API key and forge token locally
4. Start: `docker compose -f docker-compose.beta.yml up -d --build` (or binary quickstart)
5. Confirm `GET /health` → healthy
6. Unlock UI at `/ui` with API key
7. Connect **one** test repository (webhook or manual analyze API)
8. Run scan with `"report_only_dry_run": true`
9. Review executive report, findings, scanner transparency
10. Visit `/ui/learning` for calibration recommendations (read-only accept for repo scope only if operator approves)
11. Submit feedback template
12. **Do not** enable issue filing or remediation PRs

## Safety gates (operator stop conditions)

| Gate | Threshold / action |
|------|-------------------|
| Max repos per tester | 1 — pause if second repo added without approval |
| Max issues created | **0** — stop rollout immediately if any issue appears |
| Max PRs created | **0** — stop immediately |
| Finding volume storm | >5000 new findings in one scan on small repo — investigate before continuing |
| Scanner failure storm | >50% scanners failing — pause; check image and timeouts |
| DB locking / readonly errors | Stop; fix permissions before next tester |
| Secrets in logs/reports | Stop; rotate credentials; redact before resume |
| UI auth leaks raw API key | Stop; file defect; do not add testers |
| Tester enables issue filing | Revoke until operator review |

## Feedback collection

Collect via [PRIVATE_BETA_FEEDBACK_TEMPLATE.md](PRIVATE_BETA_FEEDBACK_TEMPLATE.md):

- False positives (finding ID, file, reason)
- Missed findings (expected but absent)
- Scanner failures (name, timeout vs missing)
- Confusing UI (page, flow)
- Report clarity (executive report usefulness)
- Install friction (Docker vs binary, time to healthy)
- Performance (scan duration, repo size)
- Repo map / graph usefulness

## Operator review cadence

| When | Action |
|------|--------|
| After each tester install | Confirm `/health`, report-only scan, 0 issues |
| Weekly during cohort | Review feedback; triage defects |
| Before tester #2 | Confirm tester #1 feedback incorporated or documented |
| Before any issue filing | Separate gate review (not part of this cohort) |

## Escalation

1. Tester hits safety gate → operator pauses that tester
2. Product repo active-present > 0 → halt all tester onboarding
3. Suspected credential leak → rotate keys, rebuild bundle if needed

## Success criteria (cohort complete)

- [ ] 1–3 testers installed successfully
- [ ] All first scans report-only with 0 issues created
- [ ] Feedback templates received from each tester
- [ ] No safety gate violations
- [ ] Operator documents top 3 friction items for next batch

## References

- [PRIVATE_BETA_OPERATOR_RUNBOOK.md](PRIVATE_BETA_OPERATOR_RUNBOOK.md)
- [FIRST_TESTER_PACKAGE_MANIFEST.md](FIRST_TESTER_PACKAGE_MANIFEST.md)
- [LIVE_UI_ROUTE_VERIFICATION.md](LIVE_UI_ROUTE_VERIFICATION.md)
