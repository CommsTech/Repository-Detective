# Qdrant local learning verification

Date: 2026-06-08

| Item | State |
|------|-------|
| Required for scans | **No** |
| Default enabled | `false` |
| Local collection | `cah_findings` |
| Vector size | 1024 (Cosine) |
| Payload | `CAHFindingPayload` — redacted |
| Scan on Qdrant failure | Continues (disabled store no-ops) |

## Tests

- `memory/qdrant/client_test.go` — `TestDefaultConfigUsesCAHFindings`
- `memory/qdrant/redaction_test.go` — secret stripping

## Homelab

Live `.env` may override collection; scans do not depend on Qdrant availability.

See [QDRANT_LOCAL_TESTING.md](QDRANT_LOCAL_TESTING.md).
