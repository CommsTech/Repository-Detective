# SBOM generation record — v0.1.0-beta.3

| Field | Value |
|-------|-------|
| SBOM type | **Container** (not source-tree-only) |
| Target digest | `sha256:6a615548c8a1fc2494140e73f1c3bd3f78f0ed54a7b15eaa7a1025e83e308727` |
| Source method | `crane pull` → docker-archive → Syft |
| Image ID (archive) | `sha256:cda753e90d714bc684d52cce620801853e881513706cafda7026dbc28f6cf0f1` |
| Formats | SPDX JSON, CycloneDX JSON |
| Generator | Syft |
| Syft version | `1.45.1` |
| Timestamp (UTC) | `2026-09-05T13:01:19Z` |
| Scope | OS packages + installed software as reported by Syft against the all-in-one container filesystem |

Files:

- `repository-detective-v0.1.0-beta.3.spdx.json`
- `repository-detective-v0.1.0-beta.3.cdx.json`
- `SHA256SUMS`

These are **not** cryptographic signatures. Signing status for beta.3: **CHECKSUM_ONLY** / **SIGNING_NOT_IMPLEMENTED**.
