# Verify a Repository Detective release

This guide shows how to verify **`v0.1.0-beta.3`** (accepted public-beta baseline).

Signing status for this release: **CHECKSUM_ONLY** (OCI digest + SBOM checksums).  
Cryptographic image/SBOM signing is **not** implemented for beta.3 — see decision notes below.

## 1. Identity of the release

| Field | Value |
|-------|-------|
| Version | `v0.1.0-beta.3` |
| Image digest | `sha256:6a615548c8a1fc2494140e73f1c3bd3f78f0ed54a7b15eaa7a1025e83e308727` |
| Source commit (in image) | `e130bfb` |
| E2E forge | Gitea **1.22.3** |
| Acceptance | [ACCEPTANCE_v0.1.0-beta.3.md](release/ACCEPTANCE_v0.1.0-beta.3.md) |

## 2. Pull by immutable digest

```bash
# Canonical registry
docker pull git.commsnet.org/commstech/repository-detective@sha256:6a615548c8a1fc2494140e73f1c3bd3f78f0ed54a7b15eaa7a1025e83e308727

# Public GHCR mirror (same digest)
docker pull ghcr.io/commstech/repository-detective@sha256:6a615548c8a1fc2494140e73f1c3bd3f78f0ed54a7b15eaa7a1025e83e308727
```

## 3. Confirm digest equivalence

```bash
docker image inspect \
  git.commsnet.org/commstech/repository-detective@sha256:6a615548c8a1fc2494140e73f1c3bd3f78f0ed54a7b15eaa7a1025e83e308727 \
  --format '{{.Id}}'

docker image inspect \
  ghcr.io/commstech/repository-detective@sha256:6a615548c8a1fc2494140e73f1c3bd3f78f0ed54a7b15eaa7a1025e83e308727 \
  --format '{{.Id}}'
```

Both should print the **same** image ID. Tag equality alone is not enough.

## 4. Confirm version metadata inside the container

```bash
docker run --rm --entrypoint '' \
  git.commsnet.org/commstech/repository-detective@sha256:6a615548c8a1fc2494140e73f1c3bd3f78f0ed54a7b15eaa7a1025e83e308727 \
  sh -c 'strings /app/repository-detective | grep -E "v0.1.0-beta.3|e130bfb" | head'
```

Or start briefly and query `/health` / `/api/v1/about` / Doctor for `version`, `commit`, `build_date`.

## 5. Verify SBOM checksums

```bash
cd docs/release/sbom
sha256sum -c SHA256SUMS
```

Expected files (container SBOM for the digest above):

- `repository-detective-v0.1.0-beta.3.spdx.json`
- `repository-detective-v0.1.0-beta.3.cdx.json`
- `SHA256SUMS`
- `GENERATION.md` (tool, version, timestamp)

These are **container** SBOMs (Syft against the OCI image), not a source-tree-only inventory described as the container SBOM.

## 6. Signature / provenance (beta.3)

| Mechanism | Status |
|-----------|--------|
| OCI digest documentation | **Provided** |
| SBOM SHA-256 checksums | **Provided** |
| Cosign / Sigstore image signature | **SIGNING_NOT_IMPLEMENTED** |
| SBOM attestation | **SIGNING_NOT_IMPLEMENTED** |
| SLSA / hermetic / reproducible build claim | **Not claimed** |

**Release signing decision for beta.3: `CHECKSUM_ONLY`.**

Do not treat checksum publication as a cryptographic authenticity signature.

Investigation notes: [release/SIGNING_DECISION_v0.1.0-beta.3.md](release/SIGNING_DECISION_v0.1.0-beta.3.md).

Future releases may add Sigstore keyless signing **only** when a verifiable CI OIDC identity can bind to the **exact** distributed digest without long-lived keys in the repo. Until then, verify digests + SBOM checksums.

## 7. Acceptance evidence

Read [ACCEPTANCE_v0.1.0-beta.3.md](release/ACCEPTANCE_v0.1.0-beta.3.md) for:

- clean-install proof on the published digest  
- published-image Gitea 1.22.3 E2E proof  
- scanner inventory  
- known PARTIAL / NOT_PROVEN items  

## 8. Quick checklist

1. Correct tag `v0.1.0-beta.3` (optional convenience)  
2. Immutable digest matches the table above  
3. Gitea registry digest == GHCR digest  
4. SBOM `sha256sum -c SHA256SUMS` passes  
5. Signature step: N/A for beta.3  
6. Provenance step: N/A (not claimed)  
7. Acceptance document reviewed  
