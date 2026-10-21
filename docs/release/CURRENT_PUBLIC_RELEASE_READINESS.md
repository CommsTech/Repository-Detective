# Current Public Release Readiness

**Status:** `v0.1.0-beta.6` is the current public prerelease.
**Checked:** 2026-10-21
**Source:** merge of pull request #482 onto main (CI format gate, webhook duplicate claims, AI TLS preflight).

## Ready

- Public shipped defaults no longer select the operator's Gitea host, organization, repository, network, filesystem paths, or credentials.
- Existing environment-driven configuration remains available for arbitrary Gitea installations.
- Docker Compose supports a neutral local source-build path; published images can be selected with `RD_IMAGE`.
- `gofmt`, `go vet`, and staticcheck `v0.8.2` pass on tracked packages. Package tests for the webhook, store, AI, scanner, and preflight changes pass. A full clean-room `go test ./...` on the tag remains the CI job.
- AI is optional and deterministic scanning remains the default.

## Still open after this prerelease

1. Record the immutable digest of the `all-in-one` image built from the tagged commit.
2. Run a clean-install test using the published image and a disposable Gitea instance.
3. Run a report-only scan against a disposable repository and verify scanner evidence, issue deduplication, webhook behavior, and the API auth matrix.
4. Re-run the shipped-surface portability and secret scan after publication artifacts are generated.

Shipped examples stay placeholders. Operator forge URLs, tokens, organizations, and databases stay outside the git tree.

## Known scope limits

- Gitea 1.22.3 is the proven forge baseline; Forgejo remains unproven.
- Multi-user/RBAC is not production-ready.
- Remediation PR execution remains disabled by default and its sandbox is NOT_PROVEN.
- Full scanner throughput and upgrade E2E remain NOT_PROVEN for this candidate.

## Deployment separation

The ClusterManager dogfood deployment may keep its private `.env`, image override, internal Gitea URL, organization, and AI endpoint outside this repository. Those values must never be copied into public examples, release assets, or the public GitHub/Gitea snapshots.
