# Remediation review and PR execution gate — 2026-11-17

## Scope

Baseline: `repository-detective-main(20261118-021014).zip` (user-provided). This change is a **source candidate**, not a deployed release.

The patch protects the review lifecycle, binds UI actions to the requested finding, refuses to silently skip unsupported validation commands, and surfaces patch-attempt failures to operators. It does not implement a remote sandbox, repair arbitrary dependency vulnerabilities, or make AI-generated patches trustworthy.

## Behavioral changes

- **Review:** Only `proposed → approved` or `proposed → rejected`. SQLite enforces the state predicate in the UPDATE; repeated, conflicting and stale review requests fail. Existing rejected/superseded plans cannot be revived.
- **UI binding:** A plan ID posted from a different finding page is rejected before approval, rejection, or PR creation. Merge-evidence actions reject attempt IDs unrelated to the current finding.
- **PR validation:** Every command must be allowlisted; a supported command cannot mask a later invalid command. Terminal plan states are ineligible even if `require_approval` is disabled.
- **Runner boundary:** With `remediation_pr_use_runner_verification=true` (the existing default), PR execution is blocked because the PR executor has no runner-backed validation handoff. Explicitly setting it to `false` permits the **pre-existing unisolated local execution path** and should be considered unsafe for arbitrary repositories. Keeping `remediation_pr_enabled=false` remains recommended for public/untrusted repositories.
- **Remote effects:** Already recorded open/merged PR attempts are checked before another PR is created. If attempt persistence fails, the operator receives an error explicitly warning that a PR may already exist. This is **not** a multi-instance atomic claim, and it cannot discover an unrecorded forge-side PR.
- **UX:** Finding details now show attempt IDs, failure explanations, validation results, patch summaries and PR links in a compact disclosure. `pr_opened` and `pr_merged` status are **not** presented as verified closure.

## Local evidence

| Check | Result | Evidence/limit |
|---|---|---|
| Go syntax | PASS | Go 1.23 standard-library parser parsed 704 Go files; not type checking |
| Go formatting | PASS | `gofmt` on changed files |
| Go template syntax | PASS | All 37 templates parsed with declared helper names; not a real handler render |
| CSS parse | PASS | `tinycss2` parsed shared theme without syntax errors |
| JavaScript parse | PASS | `node --check scripts/ui-remediation-evidence-check.js` |
| Synthetic layout | PASS | Chromium at 390/768/1280 px in dark/light; no overflow with deliberately long error and branch names |
| Full Go tests | NOT RUN | Go 1.26.11 needed; download of toolchain unavailable in this environment |
| Live authenticated UI | NOT RUN | Requires staging instance and prepared finding fixture |
| Gitea / runner end-to-end | NOT RUN | No external forge, runner sandbox, or approved execution tested |

## Required release gates

1. On a Go 1.26.11 runner: run `gofmt -l` over changed Go files, `go test ./...`, `go vet ./...`, and your scanner/deep-scan suite. Test a representative finding with proposed, approved, rejected and superseded plans.
2. Confirm attempting to approve/reject someone else's finding ID produces an error and never calls the review service. Confirm a stale browser tab cannot reverse an accepted decision.
3. Under `remediation_pr_use_runner_verification=true`, ensure no clone, test process, branch or PR is created. Document the behavior in configuration UI.
4. In a disposable owned repository only, and only with **explicitly approved local validation** (`remediation_pr_use_runner_verification=false`), test allowed/unsupported commands, failed tests, successful PR, duplicate click, persistence failure, and re-scan. No merging should be automatic.
5. Run `RD_FINDING_URL=https://staging.example/ui/findings/123 REPOSITORY_DETECTIVE_API_KEY=... node scripts/ui-remediation-evidence-check.js` using a non-production fixture. Check browser keyboard navigation, 200/400/500 responses, and light/dark presentation. Avoid including real credentials in logs or shell history.
6. **Before claiming sandboxed AI remediation:** implement a real isolated runner handoff and durable multi-process PR attempt claim; test malicious-repository execution, commit pinning, remote artifact verification, timeout/cancel, and post-change scanning.

## Documentation truth

Root `README.md` is intentionally unchanged: it remains a brief product landing page. Detailed execution boundaries are in `docs/REMEDIATION_PR.md`, `docs/REMEDIATION_PRS.md`, and `docs/RD-008B_CLASS_B_EXECUTION.md`; release history is in `CHANGELOG.md`.
