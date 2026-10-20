# Current Public Release Readiness

**Status:** release candidate under review — not yet a new tagged release.
**Checked:** 2026-10-16
**Source:** `7c2bd71` (generic Gitea portability corrections)

## Ready

- Public shipped defaults no longer select the operator's Gitea host, organization, repository, network, filesystem paths, or credentials.
- Existing environment-driven configuration remains available for arbitrary Gitea installations.
- Docker Compose supports a neutral local source-build path; published images can be selected with `RD_IMAGE`.
- The portability regression tests, full Go test suite, and `go vet ./...` pass on the candidate tree.
- AI is optional and deterministic scanning remains the default.

## Must be completed before tagging a public release

1. Build and publish the full `all-in-one` image from this exact commit, then record an immutable digest.
2. Run a clean-install test using the published image and a disposable Gitea instance, not the operator's internal deployment.
3. Run a report-only scan against a disposable repository and verify scanner evidence, issue deduplication, webhook behavior, and the API auth matrix.
4. Reconcile release notes, SBOMs, checksums, version badges, and verification instructions to the final tag and digest.
5. Re-run the shipped-surface portability and secret scan after publication artifacts are generated.

## Known scope limits

- Gitea 1.22.3 is the proven forge baseline; Forgejo remains unproven.
- Multi-user/RBAC is not production-ready.
- Remediation PR execution remains disabled by default and its sandbox is NOT_PROVEN.
- Full scanner throughput and upgrade E2E remain NOT_PROVEN for this candidate.

## Deployment separation

The ClusterManager dogfood deployment may keep its private `.env`, image override, internal Gitea URL, organization, and AI endpoint outside this repository. Those values must never be copied into public examples, release assets, or the public GitHub/Gitea snapshots.
