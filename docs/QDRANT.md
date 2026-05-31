# Qdrant Semantic Dedup

Bugbot can connect to your **existing Qdrant server** to avoid duplicate Gitea issues when findings are semantically similar to ones already filed in the same repository.

This is the connective tissue layer — Bugbot does not bundle Qdrant; point it at the instance you already run (for example alongside OpenClaw at `~/.openclaw/memory/qdrant/` or a LAN host).

## How it works

```
New finding generated
  ├─ Embed title + description + category (OpenAI-compatible /v1/embeddings)
  ├─ Search Qdrant collection filtered by repository
  ├─ Score >= threshold → comment on existing Gitea issue (no duplicate)
  └─ Otherwise → create issue, upsert vector + payload into Qdrant
```

## Configuration

`config/config.yaml` or environment variables:

```yaml
qdrant_enabled: true
qdrant_url: "http://127.0.0.1:6333"
qdrant_api_key: ""                    # optional
qdrant_collection: "bugbot-findings"
qdrant_vector_size: 1536
qdrant_similarity_threshold: 0.85
embedding_model: "text-embedding-3-small"
embedding_base_url: ""                # defaults to BUGBOT_AI_BASE_URL
embedding_api_key: ""                 # defaults to BUGBOT_AI_API_KEY
min_issue_confidence: 0.5             # epistemic gate — discard below this
```

Environment equivalents:

```bash
BUGBOT_QDRANT_ENABLED=true
BUGBOT_QDRANT_URL=http://192.168.255.11:6333
BUGBOT_QDRANT_COLLECTION=bugbot-findings
BUGBOT_QDRANT_SIMILARITY_THRESHOLD=0.85
BUGBOT_EMBEDDING_MODEL=text-embedding-3-small
BUGBOT_MIN_ISSUE_CONFIDENCE=0.5
```

## Epistemic confidence gates

| Confidence | Behavior |
|------------|----------|
| `< 0.5` | Discarded (no issue) |
| `0.5 – 0.7` | Issue created with `low-confidence` label |
| `0.7 – 0.9` | Issue created normally |
| `≥ 0.9` | Issue created with `high-confidence` label |

Severity and category names are also applied as Gitea labels when they exist on the repo.

## Collection setup

Bugbot auto-creates the collection on startup when `qdrant_enabled=true`. Vector size must match your embedding model dimensions.

If you already use Qdrant for OpenClaw, use a **separate collection** (default `bugbot-findings`) to avoid mixing payloads.

## Verify connectivity

```bash
curl http://127.0.0.1:6333/collections
docker logs gitea-bugbot 2>&1 | grep -i qdrant
```

Expected log when enabled:

```
Qdrant semantic dedup enabled (collection=bugbot-findings, threshold=0.85)
```

## Related

- [SCANNERS.md](SCANNERS.md) — deterministic Trivy/Grype/linter layer
- [CAH_PIPELINE.md](CAH_PIPELINE.md) — full pipeline stages including dedup clusters (`cluster-000`, …)
