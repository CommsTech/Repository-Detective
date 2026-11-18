# Independent handoff — every-push executive evidence

**Baseline:** `repository-detective-main(20261118-175151).zip` plus this patch.

**Scope:** Gitea/GitHub source CI and the existing `scripts/ci-selfscan-executive.py`. No production credentials were used in the source-only evaluation.

## Local checks performed

- `python3 -m unittest discover scripts/tests -v` — **PASS, 15 cases** (simulated API + missing-config cases; not a live scan).
- `node scripts/tests/test_feedback.cjs` — **PASS**.
- `node scripts/test-ui-rule-builder.cjs` — **PASS**.
- Python and workflow syntax, source patch replay — separate final check receipts.

## Live staging acceptance — must be completed by an agent with access

- [ ] **Always-on CI:** Push a harmless commit to `main` and `develop` for both Gitea and GitHub. Confirm `executive_selfscan` is **not skipped** when `RD_SELFSCAN_ENABLED` is absent/false. Verify the exact 40-character pushed SHA in each artifact.
- [ ] **Missing credentials:** Unset `RD_SELFSCAN_URL`/`RD_SELFSCAN_API_KEY` in a disposable fork. The job must fail, but artifact upload must preserve `executive-selfscan.md`, `executive-selfscan.json`, and `SHA256SUMS`; report decision must be `EVIDENCE_INCOMPLETE`.
- [ ] **Empty required scanner policy:** Set `RD_SELFSCAN_REQUIRED` to commas/whitespace. The job must fail closed, preserve evidence, and report that the policy is empty.
- [ ] **Valid evidence:** Configure a disposable RD service and least-privilege API key, verify new scan ID, matching repo record, forge, full commit SHA, dry-run acknowledgement, required scanner outcomes, and `sha256sum -c SHA256SUMS` from the downloaded artifact.
- [ ] **Negative evidence:** Simulate wrong repository, wrong forge, wrong SHA, missing scanner, scanner error, timeout, malformed API output, and duplicate callbacks. No instance may be counted as a complete executive self-audit.
- [ ] **Real artifact upload:** Verify GitHub Actions upload works and test the specific Gitea/act_runner version with the pinned upload action. A CI job that fails before upload is still a **release blocker**, not a passing audit.
- [ ] **Release relationship:** Confirm image publishing is gated by a policy for the exact commit and digest, or record that self-scan remains informational and cannot be described as a release-publishing gate. Do not assume the current separate CI job blocks tags/releases.
- [ ] **Operator view:** Confirm commit ID, scanner status/coverage limitations, and action priorities are legible and do not imply the repository is 'safe'.

## Additional gates not addressed in this source pass

Full Go 1.26.11 test/race suite, fresh container build and SBOM review, independent runner sandbox tests, authenticated UI flows, SSO/RBAC isolation, AI remediation, scanner FP/FN corpus, issue-posting permissions, and production performance benchmarks are **NOT TESTED** in this pass. Continue with [the complete external-agent checklist](EXTERNAL_AGENT_FULL_ACCEPTANCE_2026-11-18.md).

**Return evidence:** CI run URLs/IDs, complete pushed SHA, Gitea and act_runner versions, sanitized job logs, downloaded report/JSON/manifest, `sha256sum -c` output, trace of one successful and one intentionally failing run, and any blocked steps with owner and reproduction instructions. Use PASS / FAIL / BLOCKED / NOT RUN for each gate. Do not paste secrets, full raw logs, or vulnerable source in tickets.
