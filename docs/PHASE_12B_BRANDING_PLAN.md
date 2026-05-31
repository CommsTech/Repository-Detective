# Phase 12B — Branding Compatibility Migration (Planning)

Repository Detective — **Inspect. Analyze. Improve.**

> **Status:** Implemented (Phase 12B).

## Goal

Migrate **user-facing** names from Bugbot to Repository Detective **without breaking** existing deployments, scans, issues, or integrations.

## Naming rules (unchanged from [NAMING.md](NAMING.md))

| Surface | Name |
|---------|------|
| Product / UI / docs / reports | **Repository Detective** |
| Legacy / internal compatibility | **Bugbot** |
| Tagline | Inspect. Analyze. Improve. |

### Migration rule

```text
Read old and new names.
Write new names by default.
Never break old scans, issues, or integrations.
```

### Explicit non-goals (Phase 12B)

- Do **not** rename Go packages (`bugbot/...`)
- Do **not** rename DB tables (`findings`, `scans`, etc.)
- Do **not** rename API paths (`/api/v1/*`) — aliases only if added later
- Do **not** rewrite historical Gitea issue bodies
- Do **not** change fingerprint hash algorithm or prefix for existing findings

---

## Compatibility matrix

| Artifact | Read (legacy) | Read (new) | Write (default after 12B) | Notes |
|----------|---------------|------------|---------------------------|-------|
| Env vars | `BUGBOT_*` | `REPOSITORY_DETECTIVE_*` | `REPOSITORY_DETECTIVE_*` in docs; both work at runtime | New wins if both set |
| Config YAML keys | `enable_trivy`, etc. | Same keys | Same keys | YAML keys are not product branding |
| Viper env prefix | `BUGBOT` | Also bind `REPOSITORY_DETECTIVE` | Dual bind | One deprecation log per key per process |
| Gitea labels (base) | `bugbot` | `repository-detective` | Both during transition, then new only | Lookup must accept both |
| Gitea labels (category) | `bugbot/security`, … | `repository-detective/security`, … | Both during transition | See label plan |
| Gitea labels (lifecycle) | `bugbot/open`, … | `repository-detective/open`, … | Both during transition | |
| Issue body marker | `Bugbot fingerprint:` | `Repository Detective fingerprint:` | New wording | Extract accepts both |
| Fingerprint value | `bugbot-<hex>` | Same algorithm | Same `bugbot-` prefix | **Do not change** — breaks dedup |
| Issue search | Labels `bugbot` | Labels `repository-detective` | Search both label sets | `FindIssueByFingerprint` |
| Status context | `bugbot/security-scan` | Configurable; default new name | New default, old accepted | |
| API header | `X-Bugbot-API-Key` | Optional alias `X-Repository-Detective-API-Key` | Accept both | Document both |
| API routes | `/api/v1/*` | Same | Same | No breaking rename |
| DB tables | `findings`, `repo_settings`, … | Same | Same | Display name only in UI |
| Docker image / binary | `bugbot` | Optional tag alias | Gradual | Out of scope for 12B code |
| Runner secret env | `BUGBOT_RUNNER_*` | `REPOSITORY_DETECTIVE_RUNNER_*` | Dual bind | Same precedence rule |

---

## Part A — Implementation plan

### A1. Env var aliases

**Scope:** `main.go` config load (viper), `.env.example`, `config/config.yaml` comments.

**Mechanism:**

1. After `viper.AutomaticEnv()` with prefix `BUGBOT`, register a second pass that maps `REPOSITORY_DETECTIVE_*` → internal config keys.
2. Precedence: if both `BUGBOT_FOO` and `REPOSITORY_DETECTIVE_FOO` are set, **new name wins**; log once at INFO:

   ```text
   Both BUGBOT_ENABLE_TRIVY and REPOSITORY_DETECTIVE_ENABLE_TRIVY set; using REPOSITORY_DETECTIVE_ENABLE_TRIVY
   ```

3. Deprecation: if only `BUGBOT_*` is set, log once at startup (not per-request):

   ```text
   BUGBOT_* env vars are supported but deprecated; prefer REPOSITORY_DETECTIVE_* (see docs/NAMING.md)
   ```

**Do not rename** viper mapstructure keys in YAML (`enable_trivy`, etc.) — they are not user-facing product names.

**High-risk env vars (extra care):**

| Var | Risk if mis-merged |
|-----|---------------------|
| `*_SHARED_SECRET`, `*_API_KEY`, `*_TOKEN` | Wrong credential used |
| `*_DATABASE_PATH` | Wrong DB |
| `RUNNER_*` | Runner auth break |

**Tests required:**

- New-only, old-only, both-set (new wins), neither-set (default)
- Single deprecation log per process (mock logger / sync.Once)
- Secret vars: both-set uses new value in integration test

---

### A2. Label compatibility

**Current state** (`issues/enrich.go`, `issues/category.go`, `issues/lifecycle.go`):

- Base: `bugbot`, `automated-review`
- Category: `bugbot/security`, `bugbot/secret`, …
- Lifecycle: `bugbot/open`, `bugbot/still-present`, …
- Lookup: `FindIssueByFingerprint` filters `Labels: []string{"bugbot"}`

**Write behavior (transition mode — default in 12B):**

```text
New issues get BOTH:
  repository-detective + bugbot (base)
  repository-detective/<category> + bugbot/<category> (category)
  repository-detective/<lifecycle> + bugbot/<lifecycle> (lifecycle)
```

**Config knob:**

```yaml
label_compat_mode: dual   # dual | new_only | legacy_only
```

- `dual` (default for 12B): write both label sets
- `new_only`: after operator confirms Gitea labels exist
- `legacy_only`: emergency rollback

**Read behavior:**

- `FindIssueByFingerprint`: search issues with label `bugbot` **OR** `repository-detective`
- Category/lifecycle updates: attach to existing issue regardless of which label set was used

**Tests required:**

- `BuildLabels` dual mode produces expected union
- `FindIssueByFingerprint` finds issue labeled only `bugbot` or only `repository-detective`
- Template tests updated for new default labels while legacy tests still pass

---

### A3. Issue body compatibility

**Current:** `- Bugbot fingerprint: bugbot-deadbeef` (`issues/template.go`)

**New default:**

```markdown
- Repository Detective fingerprint: bugbot-deadbeef
```

**Read:** extend `ExtractFingerprintFromBody` to accept:

- `Bugbot fingerprint:`
- `Repository Detective fingerprint:`
- `- Bugbot fingerprint:` / `- Repository Detective fingerprint:`

**Do not change** fingerprint **values** (`bugbot-<hex>`). Changing the prefix would fork dedup across scans and break `findings.fingerprint` uniqueness.

**Tests required:**

- Extract from legacy and new body formats
- New issues render new marker text
- Same fingerprint value round-trips through forge search

---

### A4. Docs / UI

**Replace product-facing "Bugbot" with "Repository Detective" in:**

- UI page titles, nav, notices (already partially done)
- `docs/SETUP.md`, `docs/SCANNERS.md`, `docs/GITEA_STATUS.md`, `docs/CAH_PIPELINE.md`
- Operator-facing error messages

**Keep "Bugbot" in compatibility sections:**

- Env var migration notes
- Label/fingerprint legacy behavior
- Internal architecture docs where referring to package names

**Update [NAMING.md](NAMING.md):** link to this plan; mark 12B as scheduled phase.

---

### A5. API compatibility

**No breaking changes** to `/api/v1/*`.

**Optional (12B or later):**

- Accept `X-Repository-Detective-API-Key` as alias for `X-Bugbot-API-Key`
- Future branded path group `/api/v1/repository-detective/*` as thin alias — **not required for 12B**

**Tests required:**

- Both header names authenticate successfully
- OpenAPI/docs mention both headers during transition

---

### A6. DB compatibility

**No table renames.**

Optional display-only fields (low priority):

- Dashboard service name string
- Scan summary `product: repository-detective` in JSON — additive only

**Tests required:** migration v7 must be empty or additive-only if any column added.

---

## Implementation phases (recommended order)

| Phase | Deliverable | Risk |
|-------|-------------|------|
| **12B.1** | Env dual-bind + deprecation log + docs | Low |
| **12B.2** | Issue body read/write + fingerprint extract | Low |
| **12B.3** | Label dual-write + dual-read search | Medium — Gitea label proliferation |
| **12B.4** | API header alias | Low |
| **12B.5** | Docs/UI sweep | Low |
| **12B.6** | `label_compat_mode: new_only` default (future, operator opt-in) | Medium |

**Gate:** Phase 12 runner delegation stable ≥ 2 weeks in production with no auth/ingestion regressions.

---

## Migration risks

| Risk | Impact | Mitigation |
|------|--------|------------|
| Dual labels clutter Gitea UI | Operator confusion | Document cleanup script; `new_only` mode after transition |
| Both env vars set to different values | Wrong config | New-wins + explicit conflict log |
| Fingerprint prefix changed accidentally | Duplicate issues, broken dedup | **Never change** `bugbot-` prefix; test locked |
| Search only new labels | Miss existing issues | Dual-read in `FindIssueByFingerprint` before dropping legacy |
| Renaming status context | PR checks stop matching | Config accepts old context value |
| Mass doc rename breaks links | 404 in wikis | Keep redirects/notes in NAMING.md |

---

## Rollback plan

1. Set `label_compat_mode: legacy_only`
2. Continue using `BUGBOT_*` env vars only
3. No DB rollback needed (no schema change required)
4. Revert UI/doc wording via git if needed

---

## Tests checklist (Phase 12B acceptance)

- [ ] Env alias precedence (new wins)
- [ ] Env deprecation logged once
- [ ] Fingerprint extract (legacy + new body text)
- [ ] Fingerprint value unchanged (`bugbot-` prefix)
- [ ] Label dual-write
- [ ] Issue search dual-read
- [ ] API key header alias
- [ ] UI contains "Repository Detective" not "Bugbot" in user-facing strings
- [ ] `go test ./issues/... ./main config tests`

---

## Related documents

- [NAMING.md](NAMING.md) — current naming policy
- [RUNNERS.md](RUNNERS.md) — Phase 12 runner env vars (also need aliases in 12B)
- [SCANNER_ROADMAP.md](SCANNER_ROADMAP.md) — deterministic scanner expansion (separate track)
