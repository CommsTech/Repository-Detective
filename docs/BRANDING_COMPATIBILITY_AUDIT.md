# Branding compatibility audit

**Date:** 2026-06-04  
**Product:** Repository Detective — Inspect. Analyze. Improve.  
**Scope:** User-facing docs, UI, examples, and API auth documentation

---

## Rule

```text
Preferred public name:     Repository Detective
Preferred env prefix:      REPOSITORY_DETECTIVE_*
Preferred API header:      X-Repository-Detective-API-Key
Preferred labels:        repository-detective/*

Legacy compatibility:      BUGBOT_* env, X-Repository-Detective-API-Key, bugbot/* labels, bugbot-<hex> fingerprints
```

Do **not** remove legacy compatibility. Do **not** rename DB path, fingerprints, or internal module paths in this pass.

---

## References updated (preferred-facing)

| File | Change |
|------|--------|
| `README.md` | Preferred API header block; LLM env example uses `REPOSITORY_DETECTIVE_*` |
| `docs/ONBOARDING.md` | Full rewrite — Repository Detective product name, preferred header in examples |
| `docs/AI_PROVIDERS.md` | Product name, env examples, curl uses preferred header |
| `docs/TROUBLESHOOTING.md` | Wizard 401 section prefers new header/env |
| `docs/SECURITY_HARDENING.md` | Preferred header in findings table |
| `docs/AUTH_RBAC_PLAN.md` | API auth row lists preferred first |
| `docs/BACKUP_RESTORE.md` | curl examples |
| `docs/OPERATOR_READINESS.md` | curl example |
| `docs/TESTING.md` | curl example |
| `docs/NOTIFICATIONS.md` | curl example |
| `docs/DOGFOODING.md` | curl example |
| `DEPLOYMENT.md` | curl example |
| `docs/DEPLOYMENT_ISSUES.md` | API header note |
| `docs/PHASE_12B_BRANDING_PLAN.md` | Header row — preferred vs legacy |
| `docs/NAMING.md` | API header alias detail |
| `SECURITY_FIXES.md` | Middleware description |
| `ui/static/graph.js` | Fetch uses `X-Repository-Detective-API-Key` |
| `web/static/app.js` | Onboard wizard uses preferred header only |
| `main_test.go` | **New test** — both headers accepted; preferred documented in test names |

---

## References intentionally kept (compatibility / internal)

| Category | Examples | Reason |
|----------|----------|--------|
| **Env aliases** | `REPOSITORY_DETECTIVE_*` in `docker-compose*.yml`, `.gitea/workflows/ci.yml`, scripts | Legacy installs; envcompat layer |
| **API middleware** | `main.go` accepts `X-Repository-Detective-API-Key` after preferred header | Backward compatible API clients |
| **DB path** | `data/bugbot.db`, `REPOSITORY_DETECTIVE_DATABASE_PATH` default | No migration in this phase |
| **Fingerprints** | `bugbot-<hex>` values, `Bugbot fingerprint:` body marker parsing | Dedup / lifecycle compatibility |
| **Labels** | `bugbot/*` read in `dual`/`legacy_only` modes | Existing Gitea issues |
| **Gitea status context** | Default `bugbot/security-scan` | External forge integration string |
| **Go module path** | `git.commsnet.org/commstech/repository-detective` | Not user-facing; rename is separate |
| **Git remote / repo name** | `commstech/repository-detective` in clone URLs and dogfood reports | Actual forge repository name |
| **Container legacy names** | `gitea-bugbot` in some troubleshooting examples | May exist on old hosts |
| **Binary build output** | `go build -o gitea-bugbot` in TESTING.md | Dev convenience; image uses `repository-detective` |
| **CSRF salt string** | `bugbot-csrf-v1:` in `internal/security/csrf.go` | Changing breaks existing tokens |
| **Test fixtures** | `REPOSITORY_DETECTIVE_*` in envcompat/security tests | Tests legacy path |

---

## Compatibility notes (documented, not removed)

| Surface | Preferred | Legacy still works |
|---------|-----------|-------------------|
| API header | `X-Repository-Detective-API-Key` | `X-Repository-Detective-API-Key` |
| Env vars | `REPOSITORY_DETECTIVE_*` | `REPOSITORY_DETECTIVE_*` (wins: `REPOSITORY_DETECTIVE_*` if both set) |
| Issue labels (write) | `repository-detective/*` in `new_only` default | `bugbot/*` via `legacy_only` / read in `dual` |
| Issue body marker | `Repository Detective fingerprint:` | `Bugbot fingerprint:` parsed |
| Fingerprint value | — | `bugbot-<hex>` unchanged |

See [BRANDING_MIGRATION.md](BRANDING_MIGRATION.md).

---

## Tests

| Test | File | Purpose |
|------|------|---------|
| `TestRequireAPIKeyAuthAcceptsPreferredAndLegacyHeaders` | `main_test.go` | Preferred and legacy headers both authenticate |
| `TestControlPlaneRoutesRequireAPIKey` | `api/security_test.go` | Unauthenticated requests rejected (simplified router) |
| Env compat | `internal/config/envcompat/envcompat_test.go` | `REPOSITORY_DETECTIVE_*` still maps to config |

Run:

```bash
go test ./...
go vet ./...
staticcheck ./...
```

---

## Not changed (by design)

- Dogfood reports under `docs/dogfood-reports/` referencing repo `commstech/repository-detective`
- `config/config.yaml` local operator file (gitignored)
- Internal variable names (`bugbotStore`, etc.)
- License enforcement / edition gates (separate track — see [EDITIONS.md](EDITIONS.md))

---

## Related docs

- [NAMING.md](NAMING.md)
- [BRANDING_MIGRATION.md](BRANDING_MIGRATION.md)
- [EDITIONS.md](EDITIONS.md)
