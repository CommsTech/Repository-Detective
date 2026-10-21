# Public release mirror (GitHub tags & provenance)

Canonical releases live on **Gitea** (git tags + container packages).  
GitHub is the **public discovery mirror** — keep `main` history-preserving when possible.

## Rules

1. **Never force-push** GitHub `v*` release tags once published.
2. Prefer `./scripts/sync-gitea-to-github.sh --github` for `main` (full history). Do **not** use `--github-snapshot` for routine releases.
3. A GitHub `vX.Y.Z` tag must represent the **same source tree** that produced the released container (or an explicitly documented limitation).
4. Release notes must include the **immutable OCI digest**, plus SBOM / provenance when available.

## Signed / attested beta release checklist

For the next public beta (and any commercial beta):

1. **Tag** on Gitea: `vX.Y.Z` at a known commit SHA  
2. **Immutable GitHub Release** for that tag (do not allow asset replacement)  
3. **Container digest** published in release notes  
4. **SBOM** attached (Syft/CycloneDX or SPDX)  
5. **Build provenance / artifact attestation** linking image ↔ commit ↔ workflow (GitHub artifact attestations or equivalent)

That supply-chain story matches a product that sells trust in software inspection.

## Creating a public GitHub release for an existing Gitea tag

```bash
# Prefer history-preserving main first:
./scripts/sync-gitea-to-github.sh --github

# Then publish the release tag (see publish-github-release-snapshot.sh only if
# a sanitized orphan tag is still required by push protection):
./scripts/publish-github-release-snapshot.sh --tag v0.1.0-beta.3 --source-commit e130bfb
```

## Validation

```bash
./scripts/validate-github-release-tags.sh
```

## beta.3 note

Gitea tag `v0.1.0-beta.3` → commit `e130bfb` (image source).  
Later `main` tips may include product UX work beyond that image tag — release notes must bind digest ↔ source commit explicitly.
