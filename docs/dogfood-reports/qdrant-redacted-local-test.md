# Qdrant redacted local test

**Date:** 2026-06-04 (UTC)  
**Scope:** Small-sample write/read/similarity test only — no backfill, no RuView, no external push  
**Scan source:** `<scan-id>` (operator verification rescan on a private dogfood repository)  
**Runner:** `scripts/qdrant_redacted_local_test.py`

> **Sanitized report.** Dogfood used operator-local Gitea, Qdrant, and embedding credentials via `.env` / untracked config. No API keys, tokens, or internal hostnames are recorded here.

---

## Executive summary

| Check | Result |
|-------|--------|
| Qdrant reachable | **Yes** (operator-local instance; URL set via `REPOSITORY_DETECTIVE_QDRANT_URL` / `QDRANT_TEST_URL`) |
| Collection compatible (1024 / Cosine) | **Yes** — `cah_findings`, `on_disk_payload: true` |
| Sample write (8 points) | **Success** |
| Payload redaction verified | **Yes** |
| Similarity @ 0.7 | **Works** (self + cross-sample matches; no scan suppression applied) |
| Qdrant left enabled in app config | **No** — `qdrant_enabled: false` unchanged |

**Tested against a local Qdrant instance.** Collection: `cah_findings`. Vector size: 1024. Distance: Cosine. Payload mode: `cah_compat` (code: `CAHFindingPayload`). Redaction verified.

---

## 1. Qdrant reachability

```bash
curl -s http://localhost:6333/collections | jq .
# or http://127.0.0.1:6333 — operator-dependent
```

| Check | Result |
|-------|--------|
| Generic default in committed `config.yaml` | `http://127.0.0.1:6333` |
| Operator override | `.env` / `QDRANT_TEST_URL` for dogfood runs |

---

## 2. Collection status

**Collection:** `cah_findings`

| Property | Expected | Observed |
|----------|----------|----------|
| Vector size | 1024 | 1024 |
| Distance | Cosine | Cosine |
| `on_disk_payload` | true | true |
| Points before test | — | 77 (shared collection; operator environment) |
| Points after test | — | 85 (+8 prefixed test points) |

**Mismatch guard:** Vector size and distance matched config — writes were allowed. No recreation or destructive changes.

---

## 3. Configuration during test

App config was **not** switched to live Qdrant ingestion (avoids accidental backfill). Operational test used REST + embeddings directly.

Committed defaults (unchanged):

```yaml
qdrant_enabled: false
qdrant_url: "http://127.0.0.1:6333"
qdrant_collection: "cah_findings"
qdrant_vector_size: 1024
qdrant_similarity_threshold: 0.7
```

`qdrant_payload_mode: cah_compat` is implemented in code (`memory/qdrant/redaction.go` → `CAHFindingPayload`), not a separate YAML key.

---

## 4. Sample scope

- **Count:** 8 findings (target 5–10)
- **Run ID in payload:** `qdrant-redacted-local-test`
- **Payload `finding_id` prefix:** `rd-test-` (isolates test rows from production CAH payloads)
- **Qdrant point IDs:** UUID v5 derived deterministically from `rd-test-{fingerprint}` (server requires UUID/integer point IDs)

| Rule / type | Count | Notes |
|-------------|-------|-------|
| `CKV_SECRET_6` | 1 | Known false-positive class; placeholder description only |
| `DL3018` | 5 | Hadolint container findings |
| `GRAPH-SUSPICIOUS-ISLAND` | 2 | Graph noise samples |

No raw code, secrets, tokens, full issue bodies, or exploit payloads were stored.

---

## 5. Payload fields observed (read-back)

Keys on first read-back point:

`bug_class`, `confidence`, `description`, `exploit_primitive`, `file_path`, `finding_id`, `gitea_issue`, `id`, `line`, `repository`, `rule_id`, `run_id`, `severity`, `source`, `subsystem`, `target`, `timestamp`, `verdict`

**Absent (verified):** raw secrets, code fences, API tokens, full Gitea issue bodies, private disclosure text.

**CKV sample description (redacted):**  
`Base64 High Entropy String. placeholder [credential-like value redacted] in config.env.template (redaction verification sample)`

---

## 6. Similarity query

- **Threshold:** 0.7 (`qdrant_similarity_threshold`)
- **Filter:** `repository` / `target` = operator dogfood repo (sanitized in stored payloads)
- **Results:** Top hit score **1.000** (self-match); additional test samples **≥ 0.7**
- **Auto-suppression:** None — manual write test only; no scan pipeline or issue lifecycle changes

---

## 7. Embedding / vector notes

| Item | Detail |
|------|--------|
| Embedding API | Operator OpenAI-compatible `/v1/embeddings` (URL/key in `.env` only) |
| Native vector length observed | **768** (gateway-dependent) |
| Collection requirement | **1024** |
| Test adjustment | Zero-pad 768 → 1024 for upsert/search **in this proof only** |

**Risk:** Production enablement needs a **native 1024-dim** embedding endpoint (or `qdrant_vector_size` aligned to actual model output and collection recreated). Padding is not suitable for production dedup quality.

---

## 8. Qdrant enabled after test?

**No.** `qdrant_enabled` remains `false` in committed `config/config.yaml`.

Optional cleanup: delete test points listed in `qdrant-redacted-test-result.json` (UUIDs only; no secrets).

---

## 9. Risks and limitations

1. **Embedding dimension mismatch** (768 vs 1024 in this dogfood run) blocks trustworthy production dedup until resolved.
2. **Go `Upsert` point ID format:** `memory/qdrant/client.go` passes fingerprint string as Qdrant point `id`; server requires UUID/integer — live writes may fail until client maps fingerprint → UUID (test script already does).
3. **Test artifacts** may remain in a shared `cah_findings` collection (`rd-test-` payload prefix).
4. **Shared collection** may include legacy points; filter by `repository`/`target` mitigates cross-repo noise.

---

## 10. SQLite / operator scripts

A `database is locked` error during direct SQLite schema probes while the app container is running is **expected** under concurrent writes. Prefer:

- API reads first
- Read-only SQLite with `busy_timeout` when needed
- Avoid schema probes during active scans
- DB backup before offline inspection

Consider a future helper API/command for schema/status instead of ad-hoc `docker exec` SQLite access.

---

## 11. Recommendation for future enablement

1. Fix embedding pipeline to emit **1024** dimensions (or align collection size to the model).
2. Fix Qdrant point ID mapping in Go (UUID v5 from fingerprint) before `qdrant_enabled: true`.
3. Set `REPOSITORY_DETECTIVE_QDRANT_URL` in operator `.env` (not committed).
4. Re-run `scripts/qdrant_redacted_local_test.py` after embedding fix **without** zero-padding.
5. Keep `qdrant_enabled: false` until the above are done; then enable for **new findings only** (no historical backfill).

**Next step (per plan):** RuView pre-install audit at `standard` depth — third-party/manual report only (no upstream issues, email, or auto-submit).

---

## Operator-specific credentials

Repository Detective may be tested with a local operator’s Gitea, Qdrant, and API credentials during dogfooding. These credentials are never required by the product and must not be committed.

Use environment variables, Docker secrets, or local untracked config for:

- `REPOSITORY_DETECTIVE_API_KEY`
- `REPOSITORY_DETECTIVE_GITEA_TOKEN`
- `REPOSITORY_DETECTIVE_WEBHOOK_SECRET`
- `REPOSITORY_DETECTIVE_QDRANT_URL`
- runner shared secrets
- notification tokens/webhooks

Reports and examples must use placeholders or sanitized values.

---

## Machine-readable result

`docs/dogfood-reports/qdrant-redacted-test-result.json` (sanitized; no internal URLs)

---

## Output checklist

| Output | Value |
|--------|-------|
| Qdrant reachable | **yes** (operator-local) |
| Collection compatible | **yes** |
| Sample write | **success** (8 points) |
| Redaction verified | **yes** |
| Qdrant left enabled | **no** |
| Report path | `docs/dogfood-reports/qdrant-redacted-local-test.md` |
