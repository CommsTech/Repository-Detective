# Development Issues Log

## Review Findings (2026-05-30)

### Fixed in this session

| Priority | Issue | Resolution |
|----------|-------|------------|
| CRITICAL | Project did not compile (`ID::` syntax error, broken imports, circular deps) | Fixed syntax, aligned module path to `git.commsnet.org/commstech/bugbot`, extracted shared types to `models/` |
| CRITICAL | Webhook auth in `main.go` used plain string compare; rate limiting unused | Consolidated on `handlers.WebhookHandler` with HMAC-safe secret check and per-IP rate limiting |
| HIGH | `go.mod` import path mismatch (`github.com/yourusername/...` vs `yourusername/...`) | Unified on Gitea module path |
| HIGH | Missing `go.sum` | Generated via `go mod tidy` |
| HIGH | Gitea file content returned base64-encoded but never decoded | Added base64 decode in `GetFileContent` |
| MEDIUM | `Repository.DefaultBranch` missing — `ListAllFiles` could fail | Added field to struct |
| MEDIUM | API endpoints allowed requests when `api_key` empty | Reject with 503 when unset |
| MEDIUM | Raw string backticks in `createAnalysisPrompt` broke Go parser | Replaced markdown fences with plain delimiters |
| LOW | Duplicate webhook/analysis logic between `main.go` and `handlers/` | `AnalysisProcessor` interface wires engine into secure handler |

### Open / follow-up

| Priority | Issue | Notes |
|----------|-------|-------|
| MEDIUM | `handlers/webhook.go` rate limiter map grows unbounded | Add periodic cleanup or LRU |
| LOW | `WebhookHandler` still allows empty secret (logs warning only) | Consider failing closed in production mode |
| LOW | Integration tests with mocked Gitea/OpenWebUI | Unit tests added for core logic |
| LOW | Stashed local change: `docker-compose.minimal.yml` version `2.4` → `3.8` | Run `git stash pop` if still wanted |

### Completed improvements (2026-05-30)

| Item | Resolution |
|------|------------|
| Push scans entire repo | `AnalyzeChangedFiles` + `CollectChangedFiles` |
| PR scans entire branch | Uses `GetChangedFiles` with scoped CAH pipeline |
| `max_concurrent_analyses` unused | `limiter` package semaphore in `runAnalysis` |
| OpenWebUI model hardcoded | `openwebui_model` config + client field |
| No unit tests | Tests in `handlers/`, `analyzers/`, `limiter/` |

## Common Challenges to Watch For

- Gitea plugin API compatibility
- OpenWebUI API integration complexity
- Multi-language code analysis accuracy
- Performance optimization for large repositories
- Error handling and logging
