# Deep-scan findings reconciliation and report accuracy — 2026-11-17

**Source:** Maintainer-supplied `repository-detective-main(8).zip` (`c5745929`), `findings-20261117T194227Z.csv`, and the 19-page repository audit PDF generated at 2026-11-17 19:40:57 UTC. The CSV is a **persisted findings export**, not a fresh list of issues independently verified at the current source revision. The PDF contains an actual latest-scan outcome table. This document is a scoped development change, **not** a penetration test, live deployment receipt, or release certification.

## What the supplied evidence proves

| Quantity | Observed | Meaning |
|---|---:|---|
| CSV findings, entire monitored fleet | 14,467 | 36 distinct repositories in this CSV |
| CSV records for `commstech/Repository-Detective` | 617 | Includes historical resolved records |
| Self-repository open / resolved_verified | 409 / 208 | **Persisted lifecycle state**, not necessarily today's source issues |
| Self-repository open `LINT-GO-errcheck` | 233 | 91 records refer to `_test.go` files |
| Self-repository open `LINT-GO-staticcheck` | 67 | Historical persisted rule records; not equivalent to a current staticcheck run count |
| Self-repository open G304 / G204 | 6 / 3 | Seven/eight cited elsewhere may refer to another scan scope or point in time |
| Scanner outcome records in audit PDF | 16 | 15 outcomes `found`/`clean`, one `binary_missing` (golangci-lint) |

The deep scan headline shared separately by the maintainer (`54 high · 82 medium · 18 low · 46 info`) is **not** directly comparable to 409 persisted open records because the two data sets have different scopes and lifecycles. Do not represent those as a trend until repository, commit, scanners, filters, and time windows match.

## Defects corrected in this patch

1. **Report coverage misclassification:** `clean`, `found`, `passed` and `ok` were incorrectly counted as scanner failures in the executive summary and release recommendation. Scanner outcomes now use the same successful-status vocabulary as the existing quality-metrics SQL, and explicit errors still fail conservatively. The `binary_missing` outcome remains a coverage limitation.
2. **Executive risk selection:** The newest 20 open findings were fetched *before* in-memory severity sorting, allowing low-severity noise to bury older high-severity findings. The executive selection and technical appendix now request severity-first ordering in SQL **before** LIMIT and OFFSET.
3. **Internal-only false assurance:** Clean internal graph/static/health results do not by themselves authorize a positive release recommendation in the absence of evidence from any independent scanner.
4. **Test cleanup:** A bounded set of test-store `Close()` defers now checks errors. Further unchecked errors require site-by-site semantic analysis; an ignored read, write or cleanup failure must not be blanketed with a suppression.

Regression tests added for mixed successful/incomplete scanner results, internal-only evidence, and an older high-severity finding hidden behind newer low-severity results. **Execution of these regression tests remains pending** in this workspace because the code requires Go 1.26.10 and the available local toolchain is Go 1.23.2.

## G304 / G204 review decisions — NO blanket suppressions

The source archive and CSV show the following **open** sites (line numbers may drift after merges). Each case needs documented trust boundaries and a positive review before any narrow `#nosec` exemption is accepted:

| Rule | Source location (CSV) | Security interpretation / required evidence | Decision |
|---|---|---|---|
| G304 | `runner/executor.go` workspace read | Walked repository paths are untrusted; confirm symlink resolution, canonical containment and disposable runner isolation under attacker-controlled trees | REVIEW |
| G304 | `scanners/go_workspace.go` package reader | File comes from analysis workspace. Confirm containment and symlink behavior in both API and native-worker modes | REVIEW |
| G304 | `sbom/persist.go` artifact copy | Trace origin of source/destination paths, ensure they cannot escape the durable artifact root, and enforce symlink/atomic-write protections | REVIEW |
| G304 | `main.go` CA-cert file loading and nearby historical lines | Operator-specified path is not necessarily a repository path. Validate file type/size and trusted configuration authority; re-anchor stale CSV lines to the actual AST | REVIEW |
| G204 | `runner/clone.go` `git rev-parse` | Argument-vector invocation avoids shell interpolation, but clone transport, target directory, git configuration and worker network/credential isolation still matter | REVIEW |
| G204 | `operator/scanners.go` scanner-version probe | Verify executable names and flags are internal constants, path lookup is trusted and the probe cannot inherit secrets | REVIEW |
| G204 | `patcher/exec.go` allowlisted validator | The argv parser restricts commands, but `go test`/build tools can execute code from the repository. Require a disposable, constrained, no-secret worker before unattended execution | **BLOCK auto-remediation on core** |

**A scanner needs to read source and execute tools, but that fact is not itself proof a flagged site is safe.** Exemptions require a precise site, code-path evidence, security reviewer, documented residual risks, unit tests, and a time-bound review. Never suppress all G304/G204 findings in the linter configuration. Unknown, stale or moved source positions remain `NOT_VERIFIED` until reconciled by fingerprint/AST and a fresh scan.

## Suggested acceptance sequence

1. Merge only into a staging branch. Run `gofmt -l`, `go test -race -count=1 ./...`, `go vet ./...`, the configured linters, and browser tests in the actual Go 1.26.10 CI image.
2. Render a repository report with fixture outcomes `{found, clean, binary_missing}`: it must report **one** missing/incomplete outcome, not three; verify the recommendation is conservative.
3. Add an older high finding and more than 20 newer low findings: the high finding must be in **Top risks**, and the technical appendix must page by global severity rank.
4. Fix test-file error checks in bounded packages. Compare **fresh, same-commit** scans with the previous snapshot, track new vs resolved fingerprints, and keep non-actionable alerts visible without mislabeling them false positives.
5. For G304/G204, run hostile repository fixtures in a disposable runner with no credentials, no host Docker socket, constrained filesystem/egress/CPU/memory, enforced timeouts, and verified destruction after completion. Document failures before considering site-specific exclusions.
6. Reconcile the current CVE finding and the SBOM at the same commit; an SBOM timestamp from an earlier revision is not proof of the current image's dependency state.

**Release assessment:** This patch corrects evidence interpretation and some test hygiene. It does not implement an isolated remediation engine, prove scanner completeness, perform production integration, or authorize any suppression or release.
