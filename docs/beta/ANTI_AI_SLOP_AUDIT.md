# Anti-AI-slop audit

Deterministic evidence from `go test`, `go vet`, module checks, and code review — no mandatory LLM judge.

## Dead code / duplicate helpers

| Finding | Action |
|---------|--------|
| Duplicate `nullInt64` in `projects_sqlite.go` | **Fixed** — reuse store helper |
| Unused imports after beta UI split | **Fixed** during compile |

## Hallucinated dependencies

| Check | Result |
|-------|--------|
| `go list -m all` | All modules resolve via module proxy in Docker build |
| `go mod tidy -diff` | Clean after sprint changes |
| npm/Python refs in docs | Chart.js vendored in `ui/static/`; scanner pins in Dockerfile |

## Untested feature stubs

| Area | Status |
|------|--------|
| SBOM generate/check | Unit tests in `sbom/sbom_test.go` |
| Project groups | CRUD test in `store/projects_test.go` |
| Configure / pre-install / projects UI | Smoke routes in `ui/handler_test.go` |
| Runner delegation | Disabled by default; capability page documents config keys |
| Remediation PR | Disabled by default; eligibility API exists |

## Public API / docs mismatches

| Item | Status |
|------|--------|
| `/configure` route | **Fixed** — was missing; nav pointed at `/repos` |
| `/preinstall` when disabled | **Fixed** — 200 with disabled banner, not 404 |
| SBOM doc claimed Syft roadmap only | **Updated** — runtime SBOM via `sbom` package |

## Duplicate structural patterns

- Capability status builders remain centralized in `ui/capability_status.go` (no duplicate nav logic).

## Remaining risks

1. Ruff finding volume on Python repos needs profile gating (homelab_infra partial mitigation).
2. LLM reflection stages remain optional and token-capped — never sole closure gate.
3. Some beta docs describe roadmap items explicitly as not beta-supported.

## Commands

```bash
docker run --rm -v "$PWD":/src -w /src golang:1.23-bookworm go test ./...
docker run --rm -v "$PWD":/src -w /src golang:1.23-bookworm go vet ./...
```
