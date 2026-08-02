# Security hardening (Phase 9.5 — OWASP Go-SCP baseline)

Repository Detective — **Inspect. Analyze. Improve.**

This document records the Phase 9.5 security hardening pass using [OWASP Go-SCP](https://github.com/OWASP/Go-SCP) as a secure-coding baseline. No new product features were added.

## Reference

- [OWASP Go-SCP](https://github.com/OWASP/Go-SCP) — Golang secure coding practices
- [OWASP Developer Guide — Go SCP](https://devguide.owasp.org/en/05-implementation/01-documentation/02-go-scp/)

## Scope reviewed

| Area | Files | Go-SCP topics |
|------|-------|----------------|
| Pre-install audit | `preinstall/url.go`, `clone.go`, `runner.go`, `checks.go` | Input validation, SSRF, file handling, command execution, resource limits |
| HTTP server | `main.go`, `internal/security/*` | Auth, headers, body limits, error handling |
| API | `api/*` | Auth, JSON responses, no secret leakage |
| UI | `ui/*`, `ui/templates/*` | Output encoding (html/template), CSRF, API key in query string |
| Scanners | `scanners/exec.go`, `workspace.go`, `archive_extract.go` | Command execution, timeouts, path validation, output caps |
| Storage | `store/*` | Parameterized SQL, redacted evidence |
| Reports | `issues/*`, `preinstall/reports.go` | Secret redaction, disclosure disclaimers |

## Findings and fixes

### Pre-install URL / SSRF

| Finding | Fix |
|---------|-----|
| DNS rebinding between validation and clone | `RevalidateHost()` called again immediately before `git clone` |
| IP literals bypassing hostname blocklist | `net.ParseIP` check on host before/alongside DNS |
| Private IPv6 ULA (`fc00::/7`) | Explicit ULA check in `isBlockedIP` |
| `.local` / `.internal` hosts | Suffix blocklist |
| Overlong URLs | `maxRepoURLLength` (2048) |
| Weak owner/name path segments | `validRepoSegment()` rejects `..` and empty parts |
| Git arg injection | Fixed argv with `--` separator before clone URL |
| Git redirect following to private targets | Re-validation at clone time; generic clone errors (no raw git output to clients) |
| Operator secrets in git env | `internal/security.MinimalSubprocessEnv()` — no inherited secrets |

### Command execution (scanners + git)

| Finding | Fix |
|---------|-----|
| Scanner subprocesses inherited full process env | `cmd.Env = security.MinimalSubprocessEnv()` in `scanners/exec.go` |
| Remediation PR validation/git commands | Fixed argv only in `patcher/validate.go`; tokenized remotes sanitized in `patcher/git.go` |
| Unbounded scanner stdout/stderr | 4 MiB cap per invocation (`cappedBuffer`) |
| Git output in error messages | Sanitized generic errors; output truncated to 256 KiB internally |

### HTTP / API / UI

| Finding | Fix |
|---------|-----|
| Missing security headers | `internal/security.MiddlewareHeaders()` on all routes |
| Unbounded JSON POST bodies | `MiddlewareMaxBody` (1 MiB default) |
| UI POST CSRF (`api_key_only`) | HMAC CSRF token derived from API key on all UI POST forms |
| UI POST CSRF (`auth_mode=local`) | Session-bound HMAC CSRF; API JSON clients exempt |
| Local admin sessions | Signed HttpOnly cookie, SameSite=Lax, Secure when HTTPS; bcrypt passwords; migration 17 tables |
| API key in query string | Documented as **homelab-only** risk; prefer `X-Repository-Detective-API-Key` header (legacy `X-Repository-Detective-API-Key` still accepted) |
| Runner callback auth | Separate HMAC (`X-Runner-*` headers) on `/api/v1/runner/*`; no operator API key; nonce replay table |

### Archive / workspace

| Finding | Fix |
|---------|-----|
| Zip extraction without per-file read cap | `io.LimitReader` on zip entry copy |
| Existing zip-slip / symlink protections | Verified; unchanged |

### Storage / reporting

| Finding | Status |
|---------|--------|
| SQL injection | Mitigated — parameterized queries throughout SQLite store |
| Raw secrets in disclosure drafts | Redaction via `issues.SanitizeSecretEvidence`; no public draft for secrets |
| Human review disclaimer | Present on all generated reports |

## Security headers (applied)

```text
X-Content-Type-Options: nosniff
X-Frame-Options: DENY
Referrer-Policy: no-referrer
Content-Security-Policy: default-src 'self'; frame-ancestors 'none'
Cache-Control: no-store  (for /ui and /api/v1 paths)
```

## Tests added

- `internal/security/middleware_test.go` — headers, body limit, CSRF
- `preinstall/url_security_test.go` — SSRF, private IPs, DNS failure, git argv
- `scanners/exec_security_test.go` — subprocess env excludes secrets
- `api/security_test.go` — API auth + headers

Run:

```bash
go test ./...
go vet ./...
```

Optional (not required in CI):

```bash
gosec ./...        # not installed in dev environment (May 2026)
staticcheck ./...  # clean after Phase 9.5 cleanup (May 2026)
```

**Previously fixed staticcheck warnings (Phase 9.5 cleanup):**

- Removed unused `(*Engine).llmEnabled`, `labelsForIssue`, `issueSeverities`
- Replaced unnecessary `fmt.Sprintf` in `issues/lifecycle.go`

## Go module proxy (supply chain)

| Environment | Recommended `GOPROXY` |
|-------------|----------------------|
| Default / CI | `https://proxy.golang.org,direct` |
| Enterprise | `https://your-internal-artifact-proxy,direct` |
| Offline / air-gapped | `off` (after `go mod vendor`) |

Always prefer `GOSUMDB=sum.golang.org` unless your policy uses an internal checksum DB.

`goproxy.cn` is **not** documented or supported. `goproxy.io` may appear in `scripts/vendor-deps.sh` only as a **temporary local workaround** when `proxy.golang.org` is unreachable — not for DoD, FedRAMP, or defense-contractor deployments.

Dockerfile default build arg: `GOPROXY=https://proxy.golang.org,direct`.

---

## Accepted risks (homelab / Phase 9.5)

| Risk | Rationale / mitigation |
|------|------------------------|
| API key in UI query string | Convenience for browser UI; documented; use header auth in production |
| DNS rebinding during long git clone | Re-check at clone start; full TOCTOU elimination needs connect-time pinning (backlog) |
| Git HTTP redirects | Git may follow redirects; mitigated by re-validation + HTTPS-only clone URL normalization |
| No CSRF for API JSON clients | API writes require `X-Repository-Detective-API-Key` (preferred), legacy `X-Repository-Detective-API-Key`, or Bearer token; session CSRF applies to browser forms only |
| Scanner binaries are trusted | External tools (trivy, semgrep, etc.) run with minimal env but full PATH |
| SQLite file permissions | Operator must protect `database_path` at OS level |
| Rate limiting on pre-install audits | Global webhook rate limit exists; dedicated audit rate limit is backlog |
| Access logs may capture `?api_key=` | **P1:** Deprecate query-string API keys in docs; add log redaction middleware; prefer `X-Repository-Detective-API-Key` header only for internet-facing deployments |

## Follow-up backlog (post Phase 9.5)

1. **Connect-time SSRF pinning** — resolve and connect to validated IP set; reject connection to other addresses.
2. **Pre-install audit rate limiting** — per-operator and global concurrency caps for clone jobs.
3. **Session-based UI auth slice 2** — per-route RBAC, admin user CRUD, scoped API tokens. Slice 1 shipped: [AUTH_LOCAL.md](AUTH_LOCAL.md).
4. **Structured audit logging** — security events without sensitive payloads. Schema in AUTH_RBAC plan §11.
5. **gosec/staticcheck in CI** — optional pipeline step when tooling is available.
6. **Dependency scanning** — `govulncheck` in release pipeline.

## Rollback

Hardening changes are additive middleware and validation. To disable:

1. Remove `router.Use(security.MiddlewareHeaders(), ...)` in `main.go` if headers cause issues.
2. Set `preinstall_audit_enabled: false` to disable third-party clone path entirely.

No database migration changes in this phase.

---

## Post-remediation safety checklist (release hardening)

After enabling the safe remediation loop (`plan → approve → patch PR → merge → rescan → verified closure`), verify:

| Control | Requirement |
|---------|-------------|
| **Git token safety** | Gitea token stored in config/env only; subprocesses use `MinimalSubprocessEnv()` — no token in scanner or validation child env |
| **Patch PR branch safety** | Branches use `remediation_pr_branch_prefix` (default `repository-detective/fix`); never push directly to default/protected branches |
| **Validation command allowlist** | Only fixed-argv commands in `patcher/validate.go` (`go test/vet`, `staticcheck`, `hadolint` path) — no shell, no install commands |
| **No protected branch push** | PR workflow creates feature branch + PR; merge is manual in Gitea |
| **No secret auto-fix** | Patcher eligibility blocks secret/credential findings; no rotation or history rewrite |
| **Notification redaction** | Webhook payloads exclude tokens, DSNs, raw diffs, and secret evidence |
| **Runner secret boundary** | Runner HMAC separate from operator API key; runners do not receive Gitea or AI credentials |
| **Closure requires scanner success** | Default `evidence_closure_require_scanner_success: true` — no verified closure when original scanner failed or did not run |
| **No third-party auto issue submission** | Pre-install audit reports only; no automatic filing to external forges |
| **Pre-install no scripts/dependency install** | Audit clones and scans only; no `npm install`, `pip install`, or repo post-clone scripts |

Operator checklist: [OPERATOR_READINESS.md](OPERATOR_READINESS.md).

Status endpoints (no secrets exposed): `GET /health`, `GET /api/v1/status`, `GET /api/v1/about`, `GET /api/v1/ai/status`.

## Phase 19 — AI cost and local learning

| Control | Detail |
|---------|--------|
| **No startup chat test** | `ai_startup_test_enabled: false` — avoids paid "Hello" completion prompts |
| **Metadata-only probe** | Default `ai_connection_test_mode: metadata_only` uses `/v1/models` when available |
| **Manual cost warning** | `POST /api/v1/ai/test-connection` warns when chat completion mode is used |
| **Local calibration only** | Rule stats and recommendations stay in SQLite — see [PRIVACY.md](PRIVACY.md) |
| **Issue reconciliation audit** | Runs persisted in `issue_reconciliation_runs` — no issue deletion |
