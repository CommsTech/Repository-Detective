# Qdrant Semantic Dedup

Repository Detective can connect to your **existing Qdrant server** to avoid duplicate Gitea issues when findings are semantically similar to ones already filed in the same repository.

This is the connective tissue layer — Repository Detective does not bundle Qdrant; point it at the instance you already run.

## How it works

```text
New finding generated
  ├─ Build redacted normalized embedding text (source, rule, category, path_class, verdict, summary)
  ├─ Embed via OpenAI-compatible /v1/embeddings
  ├─ Search Qdrant collection filtered by repository/target
  ├─ Score >= threshold → comment on existing Gitea issue (no duplicate)
  └─ Otherwise → create issue, upsert redacted vector + payload into Qdrant
```

## Collection: `cah_findings`

Default collection name for CAH compatibility:

| Setting | Default |
|---------|---------|
| Collection | `cah_findings` |
| Vector size | `1024` |
| Distance | Cosine |
| Similarity threshold | `0.7` |
| On-disk payload | `true` |

Repository Detective auto-creates the collection when enabled. If an existing collection has a **different vector size**, writes are blocked until the collection is recreated or config matches.

## Redacted payload fields

Stored payloads are **redacted** — never raw secrets, code snippets, tokens, or full issue bodies.

| Field | Content |
|-------|---------|
| `id` / `finding_id` | Stable fingerprint |
| `bug_class` | Normalized category |
| `severity` | Critical/High/Medium/Low |
| `file_path` | Safe path (+ line) |
| `description` | Redacted summary only |
| `exploit_primitive` | Normalized primitive (rule/category) |
| `subsystem` | Top-level path/package |
| `source`, `rule_id` | Scanner metadata |
| `confidence`, `verdict`, `run_id` | Optional metadata |
| `gitea_issue`, `issue_url` | Link to filed issue |
| `repository`, `target` | Repo filter keys |

Example redacted description:

```text
Hardcoded credential-like value detected in config template.
```

Not:

```text
AWS_SECRET_ACCESS_KEY=actual_secret_here
```

## Configuration

`config/config.yaml` or environment variables:

```yaml
qdrant_enabled: true
qdrant_url: "http://127.0.0.1:6333"
qdrant_api_key: ""
qdrant_collection: "cah_findings"
qdrant_vector_size: 1024
qdrant_similarity_threshold: 0.7
embedding_model: "text-embedding-3-small"
embedding_base_url: ""
embedding_api_key: ""
min_issue_confidence: 0.5
```

Environment equivalents (prefer `REPOSITORY_DETECTIVE_*`; legacy `BUGBOT_*` via envcompat):

```bash
REPOSITORY_DETECTIVE_QDRANT_ENABLED=true
REPOSITORY_DETECTIVE_QDRANT_URL=http://127.0.0.1:6333
REPOSITORY_DETECTIVE_QDRANT_COLLECTION=cah_findings
REPOSITORY_DETECTIVE_QDRANT_VECTOR_SIZE=1024
REPOSITORY_DETECTIVE_QDRANT_SIMILARITY_THRESHOLD=0.7
```

## Operator-specific credentials

Repository Detective may be tested with a local operator’s Gitea, Qdrant, and API credentials during dogfooding. These credentials are never required by the product and must not be committed.

Use environment variables, Docker secrets, or local untracked config for:

- `REPOSITORY_DETECTIVE_API_KEY`
- `REPOSITORY_DETECTIVE_GITEA_TOKEN`
- `REPOSITORY_DETECTIVE_WEBHOOK_SECRET`
- `REPOSITORY_DETECTIVE_QDRANT_URL`
- runner shared secrets
- notification tokens/webhooks

Reports and examples must use placeholders or sanitized values. Committed `config/config.yaml` uses generic defaults (for example `qdrant_url: http://127.0.0.1:6333`); override per environment in `.env`.

## Verify connectivity

```bash
curl http://127.0.0.1:6333/collections
docker logs repository-detective 2>&1 | grep -i qdrant
```

Dogfood scripts may set `QDRANT_TEST_URL` for a one-off test without changing committed config.

## Operator scripts and SQLite

Prefer **API reads** over direct SQLite access. If you must inspect the DB while the app is running, use read-only mode, `busy_timeout`, and avoid schema probes during active scans. `database is locked` during concurrent writes is expected—not necessarily a product defect. Back up the DB before offline inspection.

Expected log when enabled:

```text
Qdrant semantic dedup enabled (collection=cah_findings, threshold=0.7)
```

## Related

- [PRIVACY.md](PRIVACY.md) — what is never stored in Qdrant
- [SCANNERS.md](SCANNERS.md) — deterministic scanner layer
- [CAH_PIPELINE.md](CAH_PIPELINE.md) — full pipeline stages
