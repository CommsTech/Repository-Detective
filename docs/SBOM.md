# Software bill of materials (SBOM)

Repository Detective pins dependencies in source control. This document is the operator-facing SBOM summary; the authoritative lockfiles are `go.sum`, `Dockerfile`, and (when used) `vendor/modules.txt`.

## Application (Go)

| Component | Version | Lock source |
|-----------|---------|-------------|
| Go toolchain (module) | 1.21 | `go.mod` |
| Go toolchain (Docker build) | 1.22 | `Dockerfile` builder stage |
| gin-gonic/gin | v1.9.1 | `go.mod` / `go.sum` |
| google/uuid | v1.6.0 | `go.mod` / `go.sum` |
| robfig/cron/v3 | v3.0.1 | `go.mod` / `go.sum` |
| sirupsen/logrus | v1.9.3 | `go.mod` / `go.sum` |
| spf13/viper | v1.17.0 | `go.mod` / `go.sum` |
| golang.org/x/time | v0.5.0 | `go.mod` / `go.sum` |
| modernc.org/sqlite | v1.34.4 | `go.mod` / `go.sum` |

Indirect Go modules are fully pinned in `go.sum`. Regenerate after dependency changes:

```bash
go mod tidy
go test ./...
```

Optional offline builds: `./scripts/vendor-deps.sh` then build with `-mod=vendor`.

## Container base

| Component | Version | Notes |
|-----------|---------|--------|
| golang (builder) | 1.22-alpine | `Dockerfile` |
| alpine (runtime) | 3.20 | `Dockerfile` |

## Scanner binaries (Docker, `INSTALL_EXTERNAL_TOOLS=true`)

| Tool | Pinned version | Install location |
|------|----------------|------------------|
| Trivy | 0.57.1 | `/usr/local/bin` |
| Grype | 0.84.0 | `/usr/local/bin` |
| Gitleaks | 8.21.2 | `/usr/local/bin` |
| Semgrep | 1.76.0 | pip (`semgrep==1.76.0`) |
| golangci-lint | v1.55.2 | `/usr/local/bin` |
| Ruff | 0.8.4 | `/usr/local/bin` |

## Go-installed scanners (always in image)

| Tool | Pinned version |
|------|----------------|
| govulncheck | v1.1.3 |
| gosec | v2.21.4 |
| staticcheck | v0.5.1 |

## Web UI (static assets)

| Asset | Version | Source |
|-------|---------|--------|
| Chart.js | 4.4.1 | `ui/static/chart.umd.min.js` (npm chart.js@4.4.1) |

## Machine-readable export

For SPDX or CycloneDX in CI, generate from the lockfiles:

```bash
# Example: CycloneDX for Go modules (requires cyclonedx-gomod on PATH)
cyclonedx-gomod mod -json -output sbom-go.cdx.json

# Example: SPDX from go.sum (requires go-spdx on PATH)
go-spdx -o sbom-go.spdx
```

Commit updated `go.sum` / `Dockerfile` pins when bumping tools; update this table in the same change.

## Related docs

- [SCANNERS.md](SCANNERS.md) — scanner configuration and enable flags
- [OPERATOR_READINESS.md](OPERATOR_READINESS.md) — verify binaries before production
- [DEPLOYMENT_ISSUES.md](DEPLOYMENT_ISSUES.md) — known deploy/version mismatches
