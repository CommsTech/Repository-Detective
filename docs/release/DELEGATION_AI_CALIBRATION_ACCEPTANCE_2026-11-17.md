# Repository Detective — runner, AI and issue-fidelity acceptance

Date: 2026-11-17. Working-tree changes against user archive nine. **Staging only.**

## Product outcome

A scanner earns trust by showing the exact commit, the checks that ran, warnings and errors, evidence for each finding, the external issue lifecycle, and the verification of a proposed fix. Neither a quiet dashboard nor a low false-positive rate is proof of safe code. Verified true negatives require a curated, versioned test corpus with known-injected defects and clean controls.

## Implemented in this patch

| Area | Previous risk | Change | Verification status |
| --- | --- | --- | --- |
| Runner job claim | Any worker claimed the oldest native job regardless of job type | Worker declares capabilities, heartbeat is checked by the core and SQLite filters job types **before** claiming | Regression test written; integration pending |
| Claim concurrency | Race could overwrite a claimed row | New filtered claim checks the conditional UPDATE affected exactly one row | Regression test written; live multiworker test pending |
| Runner registration | First claim raced with asynchronous initial heartbeat | Initial synchronous HMAC ping before claiming | Regression test written; runner test pending |
| Calibration | G104/G204/G304 globally treated as informational | Rule-wide calibration now prohibited; individually reviewed findings stay possible | Existing tests updated; execution pending |
| AI malformed JSON | Auditor could return empty success; code review could create fake 100%-confidence finding | Invalid, missing or NULL schema returns an explicit incomplete-analysis error | Mock transport tests written; execution pending |
| AI attack-surface extraction | Snake-case model fields were not mapped to Go structures; model output could appear empty | Added explicit JSON field tags and mandatory array shape checks | Regression tests written; execution pending |
| Forge selection | GitHub request without GitHub client silently fell back to Gitea | Missing target forge fails visibly | Unit regression written; execution pending |
| Filing confidence | Confidence of `0` bypassed the minimum threshold | Zero/unknown is not eligible; AI-origin claims require >=0.90 and file, line, description and evidence | Unit regression written; execution pending |
| Duplicate issue prevention | Remote lookup failure or durable claim store failure allowed new issue creation | Fail closed and retry after forge/store recovery | Integration test pending |
| Noise UX | Undifferentiated, sprawling calibration tables | Three-stage measure / review / verify workflow; clearer limitations and narrower language | HTML/CSS static checks; browser acceptance pending |
| Container toolchain | Go 1.26.10, Alpine 3.20, unpinned `gosec@latest` | Go **1.26.11** patch, Alpine **3.24**, gosec **v2.30.0** pinned | Source inspection only; full Docker build/scanner compatibility pending |

Container version sources: https://go.dev/doc/devel/release ; https://www.alpinelinux.org/releases/ ; https://pkg.go.dev/github.com/securego/gosec/v2 . These version edits do not claim that every transitive dependency or scanner is current. Use SBOM, Trivy/Grype and govulncheck from the *newly built* image and inspect pinned image digests before promotion.

## Mandatory acceptance matrix before publishing

1. **Compile and test**: `go version` shows 1.26.11; `go test ./...`, `go test -race ./runner ./store ./issues ./learning ./ai` on supported runner, `go vet ./...`; all return 0. Confirm no newly introduced test flakes.
2. **Native-worker feature matrix**: register a graph-only worker and a remediation-verify worker; enqueue both jobs; graph-only worker must not claim verify. Verify HMAC failure, stale heartbeat, wrong capabilities, empty allowlist, no compatible jobs and expiry. Prove `queued -> running -> completed/failed` with exact commit SHA.
3. **Sandbox**: use disposable VM workers for malicious repositories. Test workspace containment, network egress denial for scanner tasks, no leaked forge/admin credentials, cancellation/timeout, partial scanner failures and result size caps. Conventional Docker alone is not the required isolation boundary.
4. **AI**: mock 200-valid, 200-non-JSON, `null`, missing `findings`, 429, 500, timeout and malformed code diff. Confirm token budgets, retry limits and no phantom findings. Verify patch proposals against a clean checkout and compare original/new commits, scanner results and test results before any PR.
5. **Calibration**: inject seeded true positives for G104/G204/G304, secret patterns, vulnerable dependencies and custom rules. Confirm calibration never hides them. Measure false-positive *and false-negative* rates by rule, repository and scanner version; reject rule adjustments on insufficient or stale evidence.
6. **Issue publication**: use disposable Gitea and GitHub/Forgejo repos. Test missing credentials, forge outages, duplicate fingerprints, repeated webhooks, concurrent creators, low confidence, missing AI evidence, malicious Markdown, and redaction. No external preinstall issue submission without owner permission and independently corroborated evidence.
7. **UI/UX**: audit desktop/light/dark/mobile/keyboard, focus order, screen-reader labels, loading/error/empty states, back-navigation and direct deep-links. Exercise real data at 1, 100 and 1000 repositories and record p50/p95 page-load latency, memory and peak query times.
8. **Supply chain**: `docker compose config`, build all Docker targets, run `scripts/acceptance-scanner-image.sh` inside the resulting scanner image plus executable scanner fixture tests, produce image SBOM, scan for CVEs and sign/check release artifacts. Don't promote a new base image solely on version strings.
9. **Preinstall disclosure**: draft-first; operator review, permission to disclose, sanitized evidence, exact commit pin and repo owner verification before sending externally. Promotional link must be informative, unobtrusive and optional.

## Release blockers unchanged

- Full Gitea/GitHub Actions dispatch -> status -> artifact receipt -> independent verification is NOT implemented by this patch. The native-worker capability scheduler is a component of runner delegation, not a production-safe execution sandbox.
- AI-generated patches, PR creation, tests and same-commit rescanning are NOT proven end to end. Unattended fix application is therefore not enabled as a new capability.
- The available editing environment cannot fetch the required Go toolchain. New regression tests are written but are NOT run here. CI and disposable staging must execute this matrix before merging.
- Formal UX usability testing with new and veteran maintainers remains pending; this pass is a focused interaction/presentation refinement, not a completed global redesign.

## Rollback

Before deployment, snapshot the database and record current container digests. The runner claim protocol now requires a matching heartbeat and capability payload, so deploy the updated core and worker **together**. Roll back both if either side cannot claim jobs. Leave `runner_delegation_enabled` off while versions are mixed. No database migration is introduced by this patch.

### Scanner version backlog (intentionally not bulk-upgraded in this patch)

The install script currently pins Trivy 0.74.0, Grype 0.84.0, Syft 1.18.1, Gitleaks 8.21.2 and Semgrep 1.76.0. Multiple versions are materially behind November 2026 upstream releases. Staging must update and test them **as a separate pinned matrix**: confirm upstream signatures/checksums, downloader filenames, Alpine 3.24 compatibility, CLI output/JSON schema, vulnerability database refresh, offline behavior and CPU/memory regression. A version bump that silently disables a scanner is a failed release. The newly provided presence check is just a first gate; all tools must pass known-positive fixtures and known-negative controls.
