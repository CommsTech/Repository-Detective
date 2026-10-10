# Verify a Repository Detective release

Latest tagged beta: **`v0.1.0-beta.5`** ([release notes](release/GITHUB_RELEASE_v0.1.0-beta.5.md)) — CAH trust-boundary hardening, issue-create race fix, SQLite test-mode, SBOM UX, learning/AI assist.

Signing status: **CHECKSUM_ONLY** (OCI digest + SBOM checksums when published).  
Cryptographic image/SBOM signing is **not** implemented yet — see decision notes below.

## 1. Identity of the release

| Field | Value |
|-------|-------|
| Version | `v0.1.0-beta.5` |
| Image digest | *(filled after registry push — see status.md)* |
| Source commit (in image) | *(filled after tag)* |
| E2E forge | Gitea **1.22.3** |
| Prior digest-proven baseline | `v0.1.0-beta.4` / `sha256:8d6f224b…1264ea`; `v0.1.0-beta.3` / `sha256:6a615548…308727` ([ACCEPTANCE](release/ACCEPTANCE_v0.1.0-beta.3.md)) |

## 2. Pull by tag (until digest is published)

```bash
# Canonical registry
docker pull git.commsnet.org/commstech/repository-detective:v0.1.0-beta.5

# Public GHCR mirror
docker pull ghcr.io/commstech/repository-detective:v0.1.0-beta.5
```

After digest publication, prefer `@sha256:…` pulls (same pattern as beta.4).

## 3. Confirm digest equivalence

```bash
docker image inspect \
  git.commsnet.org/commstech/repository-detective:v0.1.0-beta.5 \
  --format '{{.RepoDigests}}'

docker image inspect \
  ghcr.io/commstech/repository-detective:v0.1.0-beta.5 \
  --format '{{.RepoDigests}}'
```

Both registries should resolve to the **same** image digest. Tag equality alone is not enough.

## 4. Confirm version metadata inside the container

```bash
docker run --rm --entrypoint '' \
  git.commsnet.org/commstech/repository-detective:v0.1.0-beta.5 \
  sh -c 'strings /app/repository-detective | grep -E "v0.1.0-beta.5" | head'
```

Or start briefly and query `/health` / `/api/v1/about` / Doctor for `version`, `commit`, `build_date`.

## 5. Verify SBOM checksums

```bash
cd docs/release/sbom
sha256sum -c SHA256SUMS
```

Expected files when a container SBOM is published for this tag:

- `repository-detective-v0.1.0-beta.5.spdx.json` (or prior tag files still present)
- `repository-detective-v0.1.0-beta.5.cdx.json`
- `SHA256SUMS`
- `GENERATION.md` (tool, version, timestamp)

These are **container** SBOMs (Syft against the OCI image), not a source-tree-only inventory described as the container SBOM.

## 6. Signature / provenance

| Mechanism | Status |
|-----------|--------|
| OCI digest documentation | **Provided when published** |
| SBOM SHA-256 checksums | **Provided when published** |
| Cosign / Sigstore image signature | **SIGNING_NOT_IMPLEMENTED** |
| SBOM attestation | **SIGNING_NOT_IMPLEMENTED** |
| SLSA / hermetic / reproducible build claim | **Not claimed** |

**Release signing decision: `CHECKSUM_ONLY`.**

Do not treat checksum publication as a cryptographic authenticity signature.

Investigation notes: [release/SIGNING_DECISION_v0.1.0-beta.3.md](release/SIGNING_DECISION_v0.1.0-beta.3.md).

## 7. Acceptance evidence

Read [ACCEPTANCE_v0.1.0-beta.3.md](release/ACCEPTANCE_v0.1.0-beta.3.md) for the last digest-proven forge baseline. Beta.5 inherits that Gitea **1.22.3** forge baseline; live ClusterMGR restart/recovery remains **NOT_PROVEN**.

## 8. Quick checklist

1. Correct tag `v0.1.0-beta.5`  
2. Immutable digest matches the published table (when filled)  
3. Gitea registry digest == GHCR digest  
4. SBOM `sha256sum -c SHA256SUMS` passes when SBOM published  
5. Signature step: N/A (`CHECKSUM_ONLY`)  
6. Provenance step: N/A (not claimed)  
7. Controlled beta: issue create off by default; manual forge review  
