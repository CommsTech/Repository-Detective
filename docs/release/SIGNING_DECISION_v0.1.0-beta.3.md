# Signing investigation — v0.1.0-beta.3

## Decision for beta.3

**CHECKSUM_ONLY** (OCI digest + SBOM SHA-256).  
Image/SBOM cryptographic signing: **SIGNING_NOT_IMPLEMENTED**.

Do not describe checksum publication as a cryptographic authenticity signature.

## Options evaluated

| Option | Verdict |
|--------|---------|
| Long-lived cosign key in repo / casual file | **Rejected** — unsafe custody |
| Sigstore keyless via Gitea Actions OIDC | **Not demonstrated** for this release; needs independently verifiable issuer binding to the exact digest |
| GitHub Actions cosign of GHCR mirror | **Deferred** — must not sign a different rebuild than the canonical Gitea digest; if used later, must re-sign/attest the **same** digest already published |
| Cosign attach (legacy) | Prefer attestations over obsolete attachment patterns when signing is implemented |

## Provenance

No SLSA level, hermetic build, or reproducible-build claim is made for beta.3.

Simple documented identity (not provenance attestation):

- Source commit in image: `e130bfb`
- Digest: `sha256:6a615548c8a1fc2494140e73f1c3bd3f78f0ed54a7b15eaa7a1025e83e308727`
- Acceptance: `docs/release/ACCEPTANCE_v0.1.0-beta.3.md`

## Next release gate

Future tagged releases should fail or warn clearly if expected container SBOM cannot be produced (`scripts/generate-release-sbom.sh` via `publish-docker-image.sh`). Signing remains optional until a trusted OIDC identity is proven end-to-end.
