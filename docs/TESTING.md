# Testing

How to verify Repository Detective locally before deploying.

## Unit tests

From the repository root:

```bash
go test ./...
```

On Linux CI, race detection is also run:

```bash
go test -race ./...
```

Windows/macOS without CGO: use `go test ./...` only (`-race` requires CGO).

### Package coverage

| Package | What is tested |
|---------|----------------|
| `handlers/` | Webhook HMAC, push routing, repo filters |
| `analyzers/` | Static rules, scope, dedup, LLM target selection |
| `scanners/` | Workspace creation, JSON parsing, optional live Trivy/linter runs |
| `gitea/` | API response decoding |
| `ai/` | Provider config and transports |

Optional integration tests (skipped when tools are not installed):

```bash
go test -v ./scanners/ -run Integration
```

## Static analysis and build

Matches Gitea Actions CI (`.gitea/workflows/ci.yml`):

```bash
gofmt -s -l .                    # should print nothing
go vet ./...
staticcheck ./...
go build -ldflags "-s -w" -o bin/repository-detective .
```

## Docker smoke test

Build and confirm the container starts and `/health` responds:

```bash
docker build -t repository-detective:test .
docker run -d --rm --name repository-detective-test -p 18080:8080 \
  -e REPOSITORY_DETECTIVE_SKIP_STARTUP_CHECKS=true \
  -e REPOSITORY_DETECTIVE_GITEA_URL=http://example.com \
  -e REPOSITORY_DETECTIVE_GITEA_TOKEN=test \
  -e REPOSITORY_DETECTIVE_AI_PROVIDER=ollama \
  -e REPOSITORY_DETECTIVE_AI_BASE_URL=http://127.0.0.1:11434/v1 \
  repository-detective:test
sleep 5
curl -sf http://127.0.0.1:18080/health
docker stop repository-detective-test
```

## Verify scanner binaries in the image

After building the Docker image:

```bash
docker run --rm repository-detective:test sh -c \
  'trivy --version && grype version && golangci-lint version && ruff --version && shellcheck --version'
```

All five commands should print version info.

## End-to-end scan test

1. Start Repository Detective with real Gitea and AI credentials (or `REPOSITORY_DETECTIVE_ENABLE_LLM_AUDITORS=false` for deterministic-only).
2. Trigger a manual scan:

```bash
curl -X POST http://127.0.0.1:8081/api/v1/analyze \
  -H "X-Repository-Detective-API-Key: $REPOSITORY_DETECTIVE_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{"owner":"org","repo":"repo","ref":"main"}'
```

3. Watch logs for scanner output:

```bash
docker logs repository-detective --tail 100 | grep -E 'SCANNER|CAH:SCAN'
```

Expected log lines when scanners run:

```
[SCANNER:trivy] found N issue(s)
[SCANNER:grype] found N issue(s)
[SCANNER:linters] found N issue(s)
[CAH:SCAN] External scanners found N candidate(s)
```

## Deterministic-only test

Disable LLM auditors to confirm zero AI calls during scan:

```bash
REPOSITORY_DETECTIVE_ENABLE_LLM_AUDITORS=false
```

Push a commit with a known issue (e.g. hardcoded secret in a `.go` file) and confirm a Gitea issue is created without `[CAH:SCAN] Running LLM auditors` in logs.

## Webhook test

In Gitea → repo → Settings → Webhooks → Test delivery. Expect **HTTP 200**.

Push a commit to a watched repo; confirm `docker logs` shows webhook processing and scan stages.

## CI on Gitea

Pushes to `main` trigger `.gitea/workflows/ci.yml`:

- Format, vet, staticcheck, golangci-lint
- Unit tests with race detector
- Docker build + health smoke test

Tag `v*` triggers release binaries in `.gitea/workflows/release.yml`.

## Related docs

- [SETUP.md](SETUP.md) — deployment walkthrough
- [SCANNERS.md](SCANNERS.md) — Trivy, Grype, linter configuration
- [TROUBLESHOOTING.md](TROUBLESHOOTING.md) — common failures
