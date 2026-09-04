# Docker images and deployment profiles

Repository Detective ships as **three image targets** from one multi-stage `Dockerfile`. Choose based on deployment shape — not on product features (suppression, remediation, and policy are the same in all variants).

## Image targets

| Image tag (example) | Target | Use case | Approx. size |
|---------------------|--------|----------|--------------|
| `repository-detective:core` | `core` | Control plane only; scanners on separate runners | Smallest (~50–80 MB + your base) |
| `repository-detective:runner` | `runner` | Gitea Actions / delegated scan workers | Large (~1–2 GB with Python scanners) |
| `repository-detective:all-in-one` | `all-in-one` | Homelab single container | Largest (core + all tools) |

### core

Includes:

- `repository-detective` binary (web, API, UI, scheduler, DB migrations, issue manager, policy)
- **git** (repository clone/checkout in-process)
- SQLite path: `/app/data` (default DB file `repository-detective.db`)
- Config mount: `/app/config/config.yaml`

Does **not** include: trivy, grype, gitleaks, semgrep, govulncheck, gosec, staticcheck, hadolint, checkov.

Use with [RUNNERS.md](RUNNERS.md) delegation or mount scanner binaries via a custom image layer.

### runner

Includes:

- `repository-detective-runner` binary
- Same scanner toolchain as all-in-one (when `INSTALL_EXTERNAL_TOOLS=true`)
- Non-root user `repositorydetective` (UID 1001)

Does **not** run the web server. Workers call back to core over HMAC-authenticated APIs.

### all-in-one

Includes everything in **core** plus **runner** binary and the full scanner toolchain. Default for `docker-compose.yml` and `./deploy.sh`.

## Build commands

### Prefer published images (operators)

All-in-one builds take **~30–60 minutes** and produce a **~4 GB** image. Releases publish to **Gitea Package Registry** (canonical); **GHCR** is an optional public mirror.

```bash
docker login git.commsnet.org   # Gitea username + token (package read)
docker pull git.commsnet.org/commstech/repository-detective:all-in-one
# or pin: git.commsnet.org/commstech/repository-detective:v0.1.0-beta.3

export RD_IMAGE=git.commsnet.org/commstech/repository-detective:all-in-one
docker compose pull
docker compose up -d
```

Public mirror (after sync):

```bash
docker pull ghcr.io/commstech/repository-detective:all-in-one
```

Publish from CI: tag `v*` on Gitea (`.gitea/workflows/docker-publish.yml` / `release.yml`).  
Publish a local sanitized build: `./scripts/publish-docker-image.sh --tag v0.1.0 --mirror-ghcr`.

| Registry | Role | Example tag |
|----------|------|-------------|
| `git.commsnet.org/commstech/repository-detective` | **Canonical** | `:all-in-one`, `:vX.Y.Z` |
| `ghcr.io/commstech/repository-detective` | Public mirror | same tags |

Set Gitea package visibility as needed for your beta testers. GHCR mirror packages should be Public for discovery.

**Sanitize before any manual publish:** if a local image was built with a live `config/config.yaml` bind-copied in, strip it first (parent layers can still contain secrets after a simple `rm`):

```bash
docker build -f Dockerfile.sanitize-publish -t repository-detective:all-in-one-publish .
# Flatten so deleted layers are not pushed:
cid=$(docker create repository-detective:all-in-one-publish)
docker export "$cid" | docker import \
  --change 'ENV PATH=/usr/local/go/bin:/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin' \
  --change 'WORKDIR /app' --change 'USER repositorydetective' \
  --change 'ENTRYPOINT ["/usr/local/bin/docker-entrypoint.sh"]' \
  --change 'CMD ["/app/repository-detective"]' \
  - repository-detective:ghcr-publish
docker rm "$cid"
./scripts/publish-docker-image.sh --source repository-detective:ghcr-publish --tag v0.1.0
```

`.dockerignore` excludes `config/config.yaml` and backups; the main `Dockerfile` only copies example configs into images.

### Build from source (developers)

```bash
# From repository root
export RD_VERSION=0.1.0 RD_COMMIT=$(git rev-parse --short HEAD) RD_BUILD_DATE=$(date -u +%Y-%m-%dT%H:%M:%SZ)

docker build --target core -t repository-detective:core \
  --build-arg VERSION="$RD_VERSION" --build-arg COMMIT="$RD_COMMIT" --build-arg BUILD_DATE="$RD_BUILD_DATE" .

docker build --target runner -t repository-detective:runner \
  --build-arg INSTALL_EXTERNAL_TOOLS=true \
  --build-arg VERSION="$RD_VERSION" --build-arg COMMIT="$RD_COMMIT" --build-arg BUILD_DATE="$RD_BUILD_DATE" .

docker build --target all-in-one -t repository-detective:all-in-one \
  --build-arg INSTALL_EXTERNAL_TOOLS=true \
  --build-arg VERSION="$RD_VERSION" --build-arg COMMIT="$RD_COMMIT" --build-arg BUILD_DATE="$RD_BUILD_DATE" .
```

Offline / DNS-filtered networks:

```bash
GOPROXY=https://proxy.golang.org,direct GOSUMDB=sum.golang.org ./scripts/vendor-deps.sh
cp ~/.local/bin/trivy deploy/bin/trivy   # optional pre-staged binaries
docker build --target all-in-one --build-arg INSTALL_EXTERNAL_TOOLS=true .
```

**Supply chain:** Default Docker build uses `GOPROXY=https://proxy.golang.org,direct`. For enterprise deployments, pass `--build-arg GOPROXY=https://your-internal-artifact-proxy,direct`. Do not use third-country public proxies as documented defaults.

Verify all targets:

```bash
./scripts/docker-build-verify.sh
```

**Disk requirement:** the verify script checks for at least **10 GB** free on the build filesystem (`VERIFY_MIN_DISK_GB` to override). All-in-one builds are large; **30+ GB free** is recommended. If the host is full, smoke tests fail with SQLite errors — see [TROUBLESHOOTING.md](TROUBLESHOOTING.md#disk-full-docker-build-or-verify-fails).

## Pinned scanner versions (all-in-one / runner)

| Tool | Version | Install method |
|------|---------|----------------|
| trivy | 0.57.1 | Release tarball or `deploy/bin/trivy` |
| grype | 0.84.0 | install.sh |
| syft | 1.18.1 | install.sh (SBOM generation) |
| cyclonedx-gomod | latest (builder) | `go install` (Go module SBOM) |
| gitleaks | 8.21.2 | Release tarball |
| semgrep | 1.76.0 | pip (`semgrep==…`) |
| govulncheck | 1.1.3 | `go install` (builder stage) |
| gosec | 2.21.4 | `go install` |
| staticcheck | 0.5.1 | `go install` |
| hadolint | 2.12.0 | Release binary |
| checkov | 3.2.254 | pip (`checkov==…`) |
| golangci-lint | 1.55.2 | install.sh (optional linters) |

Override at build time via env in `scripts/install-scanner-tools.sh` (e.g. `TRIVY_VERSION=…`).

## Compose profiles

Example file: [examples/docker-compose.yml](examples/docker-compose.yml)

| Profile | Services |
|---------|----------|
| `all-in-one` (default) | `repository-detective` |
| `core` | `repository-detective-core` |
| `runner-example` | One-shot runner image smoke (not production workflow) |

```bash
docker compose -f docs/examples/docker-compose.yml --profile all-in-one up -d --build
docker compose -f docs/examples/docker-compose.yml --profile core up -d --build
```

Root [docker-compose.yml](../docker-compose.yml) is the **recommended** all-in-one deploy: bridge networking, published port **8081**. Optional host networking: `docker compose -f docker-compose.yml -f docker-compose.host-network.yml up -d`.

## Volumes and paths

| Path | Purpose |
|------|---------|
| `/app/data` | SQLite database (`REPOSITORY_DETECTIVE_DATABASE_PATH`, default `/app/data/repository-detective.db`) |
| `/app/config` | Read-only `config.yaml` (mount from host `./config`) |
| `/app/certs` | Optional CA bundles for private AI/Gitea TLS |

**Do not** mount the Docker socket. **Do not** run privileged.

## Secrets and environment

- **Never** copy `.env` into the image (`.dockerignore` excludes it).
- Prefer `REPOSITORY_DETECTIVE_*` variables; legacy `REPOSITORY_DETECTIVE_*` still works ([envcompat](../internal/config/envcompat)).
- Provide secrets via `env_file`, Docker secrets, or orchestrator — not baked into layers.

## Health checks

- `GET /health` — liveness (no auth)
- `GET /api/v1/status` — scanner availability, DB, features (API key)

Container healthcheck runs `scripts/docker-healthcheck.sh`, honoring `REPOSITORY_DETECTIVE_PORT` / `REPOSITORY_DETECTIVE_PORT`.

## Runtime user

All targets run as **`repositorydetective` (UID 1001)**. Host `data/` should be writable by that UID or world-writable in homelab setups.

## Image size tradeoffs

| Choice | Benefit | Cost |
|--------|---------|------|
| **core** | Fast pulls, smaller attack surface | Requires runner hosts with scanners |
| **runner** | Repeatable CI workers | Large image; Python deps (semgrep, checkov) |
| **all-in-one** | Simplest ops | Slow builds/pulls; duplicates tools if you also use runners |

## Rollback plan

1. Note current image ID: `docker images repository-detective`
2. Stop container: `docker compose down`
3. Backup DB: `cp data/repository-detective.db data/repository-detective.db.bak` (see [BACKUP_RESTORE.md](BACKUP_RESTORE.md))
4. Run previous tag: `docker run … repository-detective:all-in-one@<previous-digest>`
5. Confirm `GET /health` and dashboard; re-run one manual scan

## Known limitations

- Scanner install requires network during build unless binaries are staged under `deploy/bin/`.
- **core** cannot run in-process Trivy/Semgrep/etc. without delegation or custom layers.
- musl/Alpine binaries — pre-staged `deploy/bin/*` must match architecture (amd64 assumed in install script).
- `checkov` / `semgrep` add significant image size and build time.

## Related docs

- [OPERATOR_READINESS.md](OPERATOR_READINESS.md)
- [RUNNERS.md](RUNNERS.md)
- [BACKUP_RESTORE.md](BACKUP_RESTORE.md)
- [UPGRADE.md](UPGRADE.md)
