# Public beta readiness — verification status

**Source baseline reviewed:** `repository-detective-main(20261118-193222).zip` (2026-11-18).

**Current release verdict:** **NOT CERTIFIED FOR UNRESTRICTED PUBLIC BETA**. This is a source review, not evidence of a deployed release. The latest live image/tag, runner state, and production behavior were not independently verified in this pass.

Repository Detective can be useful for controlled testing today, but the presence of a configuration switch, a CI workflow, or a successful static check is not proof that its complete security workflow operates as intended.

## What the source currently supports

- Self-hosted application with a product-focused README, local login, finding investigation, custom declarative rules, scanner integrations, and evidence-backed reports.
- Operator-initiated report-only scans, optional AI advice, reviewable remediation plans, and Gitea-oriented forge integration.
- Commit-pinned CI self-scan report generation for **every source push** to the configured Gitea/GitHub CI branches. When unconfigured, the job must generate `EVIDENCE_INCOMPLETE` rather than disappearing. The latest working-tree patch adds a release-time evidence gate; real runner/publication behavior has not been verified.
- In-app, review-first feedback drafting, without automatic public-log uploads or anonymous server-side issue filing.

## Required acceptance gates before wide distribution

| Priority | Area | Pass evidence required |
|---|---|---|
| P0 | Source, tests and release reproducibility | Full Go 1.26.11 suite (`go test -race ./...`), vet, format, reproducible image builds, dependency/SBOM scans, source SHA to image digest provenance |
| P0 | Self-scan and release integrity | Both CI platforms generate commit-specific reports; tag-triggered image/release workflows must fail closed on incomplete evidence, attach checksummed reports, and never publish on upload/gate failures. See [release evidence gate](RELEASE_EVIDENCE_GATE_2026-11-18.md). |
| P0 | Runner isolation | Real worker dispatch, capability matching, clean ephemeral workspaces, denied secrets/network access, evidence persistence and interrupted-job recovery verified in a disposable environment |
| P0 | Authentication and authorization | Local user/role authorization matrix including attempts to cross repository/user boundaries; external Authentik/Gitea login remains unverified, not production-enabled by assumption |
| P0 | Remediation correctness | AI plan → isolated patch → validation → rescan → reviewable PR; verify failure handling and no false closure or hidden execution on the main host |
| P0 | Scanner precision and completeness | Known-positive/known-negative fixtures, missing scanner behavior, baseline versus candidate comparison, calibration regression tests including false negatives |
| P1 | User experience | Authenticated desktop/mobile, light/dark, keyboard, forms, filters, both exports, diagnostics, and issue-draft privacy tests with screenshots/traces |
| P1 | Performance | p50/p95 page and scan duration, CPU/RAM/disk benchmarks, and high-volume finding queries on representative repositories |
| P1 | Public onboarding and support | Fresh install with placeholder-only config, successful first scan, support issue draft review, README links and clean release notes |

## Important distinctions

A `REVIEW_FINDINGS` or `REVIEW_COVERAGE` self-scan decision is **not** a security certification. The report records a point-in-time observation from configured scanners. Likewise, a passed unit suite does not verify untrusted-code containment, live forge side effects, or end-to-end AI remediation.

For detailed execution instructions, use [External-agent full acceptance](EXTERNAL_AGENT_FULL_ACCEPTANCE_2026-11-18.md) and [CI self-scan acceptance](CI_SELFSCAN_IDENTITY_ACCEPTANCE_2026-11-18.md). For the user-facing installation path, return to the [README](../../README.md).

## Historical note

A previous version of this document was a **2026-10-21 snapshot** describing `v0.1.0-beta.6` and an October GHCR digest. That snapshot should not be treated as the current release or used to recommend an image tag. Verify current release assets and image digests against the actual forge and registry before publication.
