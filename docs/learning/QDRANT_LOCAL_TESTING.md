# Qdrant local testing (`cah_findings`)

Qdrant is **optional** — not required for private beta or production scans.

## Local learning collection

| Setting | Local test value |
|---------|------------------|
| Collection | `cah_findings` |
| Vector size | 1024 (Cosine) |
| Enabled | `qdrant_enabled: false` by default |

Enable only for local/private semantic similarity experiments:

```yaml
qdrant_enabled: true
qdrant_url: "http://127.0.0.1:6333"
qdrant_collection: "cah_findings"
```

Or via env:

```bash
REPOSITORY_DETECTIVE_QDRANT_COLLECTION=cah_findings
```

## Payload schema

Uses `CAHFindingPayload` (`memory/qdrant/redaction.go`):

- Redacted description (no raw secrets)
- Severity, file path, rule id, repository, fingerprint metadata
- No raw code blocks unless redacted

## Redaction policy

- `RedactSummary()` strips credentials, code fences, long snippets
- Secrets never stored in Qdrant payload
- Scan pipeline continues if Qdrant is disabled or unreachable

## Tests

```bash
go test ./memory/qdrant/...
python3 scripts/qdrant_redacted_local_test.py  # optional local proof
```

## Not required for beta

Beta package ships with `qdrant_enabled: false`. Operators may enable locally without affecting scan correctness.

See also [QDRANT.md](../QDRANT.md).
