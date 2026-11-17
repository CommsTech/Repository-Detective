# Beta release build (CI-independent)

## Quick start

```bash
cd /home/commstech/Repository-Detective
make beta-release
```

Output directory:

```text
dist/repository-detective-beta/
  repository-detective
  checksums.txt
  sbom-go.cdx.json          # when cyclonedx-gomod available
  config.example.yaml
  docker-compose.beta.yml
  README_BETA.md
  RELEASE_NOTES.md
```

## Requirements

- Go 1.23+ with CGO (sqlite)
- Optional: `cyclonedx-gomod` on PATH for release SBOM

## What is excluded

- `.env`, tokens, local DBs
- Built binary is **not** committed to git (`dist/` gitignored)

## When Gitea Actions is flaky

Operators can ship internal/private beta using this script alone. CI remains useful but is not the only release path.

## Verify

```bash
cd dist/repository-detective-beta
sha256sum -c checksums.txt
./repository-detective --help  # or run via compose
```
