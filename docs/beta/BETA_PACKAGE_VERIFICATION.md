# Beta package verification

Date: 2026-06-07  
Commit: `6a2cbfd` + verification re-run

## Build commands

```bash
make clean-beta-release
make beta-release
./scripts/check-beta-package-secrets.sh
```

## Package contents

```text
dist/repository-detective-beta/
  repository-detective      # ELF, ~20MB, owned by commstech
  checksums.txt
  config.example.yaml
  docker-compose.beta.yml
  README_BETA.md
  RELEASE_NOTES.md
  .env.example              # placeholders only
```

## Verification checklist

| Check | Result |
|-------|--------|
| Builds as current user | PASS |
| No root-owned dist blocking rebuild | PASS (`clean-beta-release` + staging dir) |
| checksums.txt present | PASS |
| README_BETA.md present | PASS |
| docker-compose.beta.yml present | PASS |
| config.example.yaml safe (no secrets) | PASS |
| No live `.env` in package | PASS |
| No repository-detective.db in package | PASS |
| No local binaries beyond release ELF | PASS |
| `check-beta-package-secrets.sh` | PASS |

## SBOM

| Item | Status |
|------|--------|
| sbom-go.cdx.json | Not generated — `cyclonedx-gomod` not installed at build time |
| Policy | Optional for private beta; recommended for public beta |

## Secrets grep

Manual grep for real credentials found only safe example references:

- `.env.example`: `REPOSITORY_DETECTIVE_API_KEY=change-me-to-a-secure-random-string`
- `docker-compose.beta.yml`: env var substitution comments

No `REPOSITORY_DETECTIVE_GITEA_TOKEN`, `AKIA`, or private keys in package.

## Artifacts not committed

`dist/` is gitignored — built package is local-only.
