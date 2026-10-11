# Review Rubrics

Repository-Detective reports findings with a 0 to 5 review posture:

- `0` critical failure
- `1` major deficiencies
- `2` partial coverage
- `3` mostly compliant
- `4` strong coverage
- `5` fully compliant or no findings

Security gates can block a merge. Optimization and public-release checks are advisory unless the organization makes them mandatory.

## Security

The core security rubric covers secrets, supply chain/SBOM, structural analysis, input validation, crypto, error and resource handling, access control, audit logging, maintainability, and repository governance.

Recommended tool coverage:

- Secrets: Gitleaks, TruffleHog
- Supply chain and SBOM: OWASP Dependency-Check, Syft, Grype, Dependency-Track
- SAST and validation: Semgrep, CodeQL, SonarQube, Fortify, Checkmarx
- Resource and defect analysis: SonarQube, language linters, Coverity
- Governance: platform APIs, OpenSSF Scorecard
- Host hardening for self-hosted runners: OpenSCAP

## Pipeline Governance

Pipeline checks are treated as a separate security rubric because a compromised runner or workflow can bypass every other gate.

- Pipeline definition reviewed and branch-protected
- Secrets sourced from managed stores, masked, scoped, and rotated
- Runners isolated by environment; ephemeral runners preferred
- Third-party actions pinned to immutable commit hashes
- Pipeline tokens scoped to least privilege
- Security gates enforced with approved, logged overrides
- Pull requests cannot tamper with the workflow that evaluates them
- Artifacts signed and traceable to commit and run
- Pipeline activity logged and monitored
- Self-hosted runners patched and hardened

Repository-Detective's built-in static rules flag floating action refs and obvious secret-printing patterns. Branch protection, runner isolation, token scopes, signing, and monitoring require platform API checks or human review.

## Public Release

Public-release review focuses on information disclosure around the code, not only the current source tree.

- Full git history scrub for secrets and internal infrastructure
- Internal comments, commit messages, names, ticket IDs, and codenames removed
- PHI, PII, and intellectual-property review completed by an accountable reviewer
- Internal URLs, private IPs, VPN/proxy details, and environment-specific endpoints removed
- Test data and fixtures are synthetic
- Dependency licenses permit public redistribution
- Release authorization is documented
- Stale branches and tags are pruned or reviewed
- README, docs, and wiki content sanitized
- Issues, pull requests, and project metadata reviewed before publication

Repository-Detective's built-in static rules flag private IPs, localhost, `.local`, and `.internal` references in scanned files. Full history, branch/tag cleanup, authorization, licensing, and issue tracker review need external tooling and human sign-off.

## Optimization

Optimization checks are advisory. Profile first, then fix the measured hot paths.

- Algorithmic complexity
- Database and query efficiency
- Memory allocation and footprint
- Network and I/O efficiency
- Caching and invalidation
- Concurrency and contention
- Dead and redundant code
- Resource pooling and reuse
- Build, image, and binary size
- Serialization efficiency

Repository-Detective's built-in static rules flag simple nested-loop and per-call HTTP-client patterns. Use pprof, py-spy, JDK profilers, query analyzers, k6, JMeter, Locust, Prometheus, or Grafana for runtime evidence.
