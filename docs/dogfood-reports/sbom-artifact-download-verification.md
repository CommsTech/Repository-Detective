# SBOM artifact download verification

**Date:** 2026-06-11  
**Live revision:** `rc-e3e19ec`

## Status: PASS (controlled proof)

| Check | Result |
|-------|--------|
| UI `/ui/scans/:id/sbom` | **PASS** — HTTP 200 |
| UI `/ui/repos/:id/sbom` | **PASS** |
| Download route exists | **PASS** |
| Download with real artifact | **PASS** |
| Missing artifact honest 404 | **PASS** |

## Artifact

| Field | Value |
|-------|-------|
| Scan ID | `926a5f56a26f03c9` |
| Repository | commstech/Repository-Detective (id=1) |
| Format | CycloneDX JSON |
| Generator | Syft (anchore/syft container, controlled proof) |
| Components | 895 |
| File size | 859,309 bytes |
| Container path | `/app/data/sbom-proofs/926a5f56a26f03c9/sbom.syft.cdx.json` |
| Source ref | Product scan `926a5f56a26f03c9` on `main` |

## Verification commands (redacted)

```bash
curl -H "X-Repository-Detective-API-Key: $API_KEY" \
  http://localhost:8081/ui/scans/926a5f56a26f03c9/sbom/download \
  -o /tmp/sbom.cdx.json
# HTTP 200, bomFormat=CycloneDX, 895 components

curl -H "X-Repository-Detective-API-Key: $API_KEY" \
  http://localhost:8081/ui/scans/0000000000000000/sbom/download
# HTTP 404 "sbom not available"
```

## Notes

- All-in-one core image does not bundle syft/cyclonedx-gomod; repo scans record `sbom_tool_missing`.
- Controlled proof uses external Syft against workspace, stores artifact on mounted `/app/data`, and registers `sbom_artifacts` row — validates download path end-to-end.
- Production path: install syft in image or delegate SBOM to runner (alpine:3.20 demo had syft `ok` on runner).

## Acceptance

SBOM UI and download **proven** with real CycloneDX artifact.
