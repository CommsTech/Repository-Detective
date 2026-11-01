# First tester package manifest

Date: 2026-06-02  
Build commit: `46cf4bf`  
Build command: `make clean-beta-release && make beta-release`

## Generated files

```text
dist/repository-detective-beta/
  repository-detective          # ELF ~20 MB
  checksums.txt
  config.example.yaml           # from config/private-beta.example.yaml
  docker-compose.beta.yml
  README_BETA.md
  RELEASE_NOTES.md
  .env.example
```

## Checksums

File: `dist/repository-detective-beta/checksums.txt`

```
cb4a7ddba524ea2dcc48e9fee7003a699f34123aac4d2fde919bf67902b1c89e  repository-detective
```

Verify: `cd dist/repository-detective-beta && sha256sum -c checksums.txt`

## SBOM status

| Item | Status |
|------|--------|
| `sbom-go.cdx.json` | **Not generated** |
| Reason | `cyclonedx-gomod` not installed on build host |
| Policy | Optional for first tester cohort; document in release notes |

## Secrets check

```bash
./scripts/check-beta-package-secrets.sh
grep -RInE '(REPOSITORY_DETECTIVE_GITEA_TOKEN|REPOSITORY_DETECTIVE_API_KEY|AKIA|BEGIN RSA|BEGIN OPENSSH)' \
  dist/repository-detective-beta
```

| Check | Result |
|-------|--------|
| `check-beta-package-secrets.sh` | **PASS** |
| Real credentials in bundle | **None** |
| Placeholder references only | `.env.example`, compose env substitution |

## Build environment

| Item | Value |
|------|--------|
| Builder | `golang:1.23-bookworm` container (go not on host PATH) |
| Artifact owner | `commstech` (non-root) |
| `dist/` committed | **No** (gitignored) |

## Referenced docs (source repo, not in bundle)

Testers receive bundle + operator pointers to:

- [PRIVATE_BETA_TESTER_GUIDE.md](PRIVATE_BETA_TESTER_GUIDE.md)
- [PRIVATE_BETA_RELEASE_NOTES.md](PRIVATE_BETA_RELEASE_NOTES.md)
- [PRIVATE_BETA_FEEDBACK_TEMPLATE.md](PRIVATE_BETA_FEEDBACK_TEMPLATE.md)
- [FIRST_TESTER_ROLLOUT_PLAN.md](FIRST_TESTER_ROLLOUT_PLAN.md)

## Distribution checklist

- [ ] Verify checksum with tester
- [ ] Send via secure channel (not public git)
- [ ] Include announcement draft customized with contact/path
- [ ] Confirm report-only first scan in onboarding
- [ ] Do not include operator `.env` or `data/repository-detective.db`
