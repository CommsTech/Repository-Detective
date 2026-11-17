# Benchmarking Repository Detective against Veracode's public AppSec practices

**Date:** 2026-11-17
**Status:** Working-tree research and one narrow safety implementation. This is not a new release acceptance certificate, Veracode integration, or an assertion of feature parity.

## What to learn — and what to preserve

Veracode's published documentation describes risk-based finding prioritization (severity, exploitability, affected common modules and expected fix effort), policy constraints and required scan cadence, SCA with SBOM generation, and reviewed AI-generated dependency patches that account for direct/transitive dependencies and API changes. Its Fix for SCA documentation describes a sandboxed backend and explicitly cautions customers to review generated resolutions (limited-availability as of this assessment).

**Repository Detective's distinct position:** self-hosted, Gitea-first and deterministic-first; check code quality, tech debt, architecture and optimization alongside AppSec; make AI optional and preserve local source and credentials. Do not imply it uses Veracode's models, intelligence or proprietary fixes.

## Grounded gap assessment (based on the 2026-11-17 source archive)

| Capability | What Repository Detective currently has | What to build / prove | Priority |
|---|---|---|---|
| Findings and triage | Canonical findings, severity-first queue, opt-in CISA KEV and FIRST EPSS lookup | Explainable fix-first ranking using deployment criticality, reachability, exploitation evidence, affected shared modules, and estimated fix effort; show *unknown* rather than assume safe | P0 |
| Vulnerable dependencies | Scanner findings, optional AI **advice**, deterministic plans | Version- and ecosystem-aware direct/transitive upgrades in isolated disposable worker, lockfile consistency, breaking API migration, tests, rescan, human-reviewed Gitea PR | P0 |
| Sandboxed workers | Native worker exists; Gitea Actions/GitHub Actions end-to-end execution is incomplete | Gitea dispatch + per-job limits/isolation, token scope, immutable commit pinning, signed result artifact, fail-closed result ingestion | P0 |
| Release policy | Observe/Warn/Enforce policies and coverage concepts | Required scanner matrix and recency, new-vs-baseline policy, exception expiry, risk acceptance authority, policy decisions bound to immutable commit and scanned artifact | P1 |
| Supply-chain evidence | SBOM artifacts/scan paths and scanner inventories | Validate CycloneDX/SPDX SBOMs, tie to build artifact digest, SLSA-aligned provenance, signature verification, dependency drift and attestation of tested commit | P1 |
| Learning quality | Learning/calibration components with safeguards against unattended AI-only false-positive suppression | Evaluate on labeled ground-truth fixtures, report precision *and recall* and trend, detect silent false negatives, human approval to suppress; keep rule provenance and reversal | P1 |
| Continuous scanning | Scheduled/on-demand scan orchestration | Publish advisory-to-inventory impact matching; trigger affected repos for fresh CVEs without rescanning every unaffected repo; show stale DB and partial scan states | P1 |
| Developer experience | Findings search, detail workflow, custom rules, docs | Progressive disclosure for beginners, advanced drilldown for veterans, scanner/PR step timeline, reduced issue noise, tested accessibility and dark-mode navigation | P1 |
| Optimization | Scan profiles and worker/resource controls | Benchmark cold and warm scans on reproducible fixtures; cache vulnerability DB by version, cap I/O and parallelism, measure CPU/RSS/disk/latency and fail on material regressions | P2 |

**Important:** `P0` above identifies engineering priority, not proof of an exploitable product vulnerability. Actual defects must be established through tests and evidence.

## Implemented in this pass

1. Every recognized dependency-vulnerability remediation plan now explicitly requires human review and is ineligible for unattended PR generation, even if a scanner reports a fixed version. The patcher does not have an accepted dependency-upgrade executor.
2. A missing scanner-reported fixed version is an explicit plan blocker. Planning does **not** assume an upgrade exists.
3. Upgrade advice covers direct/transitive assessment, manifest+lockfile consistency, API compatibility, isolation, tests and rescanning the changed revision.
4. The finding-detail remediation step displays a beginner-readable checklist without hiding the underlying technical evidence. No external lookups or new privileges are introduced.
5. Added focused Go regression cases for dependency categories, absent fixed versions and preservation of deterministic low-risk Dockerfile routing.

This pass **does not implement** automated dependency patches, reachability analysis, a verification score, new cloud services, or full runner sandboxing.

## Next safe build order and acceptance criteria

**Phase A — evidence-driven fix-first queue:** Define a versioned, explainable decision model; risk = severity + verified active exploitation + reachable runtime + criticality + reusable component impact. Keep *not measured*, *unknown* and *not applicable* distinct. Effort is a separate axis, not a way to hide serious findings. Acceptance: KEV high-priority fixture ranks above unexposed non-KEV fixture when inputs are otherwise equal; unavailable intelligence does not lower risk; all decisions have source timestamps; old critical findings remain in paginated results.

**Phase B — dependency fix workbench:** First support one ecosystem (Go or npm) with a disposable unprivileged worker, an explicitly pinned commit, fixed toolchain/database versions, a lockfile-aware PR, tests, SCA rescans and human approval. Expand ecosystems only after tests prove consistency. Never run arbitrary repository build scripts on the main API host. Acceptance: vulnerable/clean/major-breaking/no-fix/signed-package-tamper fixtures pass; no tokens leak to logs, AI prompts, or resulting PRs; job cancellation/timeout kills children and cleans workspaces.

**Phase C — trust evidence:** Display a *verification record*, not a blanket “safe” badge. Include commit SHA, scan tool versions, database timestamp, expected versus completed scanners, SBOM digest, artifact digest, reviewer, exception register, and test/closure evidence. Incomplete or stale evidence must visibly fail policy or be reported indeterminate, not healthy. Acceptance: every missing scanner, wrong commit, unsigned artifact, failed rescan or out-of-date database is distinguishable from a clean verified run.

**Phase D — policy baseline and CI enforcement:** Compare new against accepted findings, with grace periods, owner policy, exception expiry and periodic full scans. Human review of any change that relaxes policy. Acceptance: new high severity blocks policy, existing accepted finding is visible and time-bounded, and partial coverage never counts as success.

**Phase E — UI, performance and learning:** Test first-time setup, daily triage, forensic detail and remediation PR flows at mobile/desktop widths and both themes; run screen-reader/keyboard checks and automated browser navigation with errors, timeouts and offline scans. Benchmark scanner wall time, CPU, peak RSS, temporary disk, database update duration and AI tokens per validated fix on an immutable fixture suite. Report learning-system false negatives as carefully as false positives.

## Principles that should not change

- Scanner evidence, coverage and provenance are separate from AI opinion; never let AI-only assertions suppress ground-truth positives.
- Source or customer credentials remain private by default. A public CVE ID lookup is opt-in; shipping code to any hosted remediation service requires explicit informed consent.
- Untrusted repositories, workflow definitions and dependency install hooks are **untrusted code**, even inside a container. Isolate from the control plane and use ephemeral, least-privilege jobs with restricted outbound network.
- A passing scan proves only what that scanner tested, under that environment and policy. It is not absolute proof that the repository cannot contain vulnerabilities.

## First-party sources consulted

- Veracode, *Develop a remediation plan*: https://docs.veracode.com/r/review_remediationplan (last updated July 7, 2026)
- Veracode, *Fix for SCA*: https://docs.veracode.com/r/Veracode_Fix_for_SCA (limited availability and review caution)
- Veracode, *Security policy constraints*: https://docs.veracode.com/r/Security_policy_constraints
- Veracode, *Create and scan SBOMs*: https://docs.veracode.com/r/Supply_chain_management_and_SBOMs
- Veracode, *Veracode Pipeline Scan*: https://docs.veracode.com/r/r_pipeline_scan_commands/

Sources are used as architecture and workflow references, not as code dependencies or endorsement.
