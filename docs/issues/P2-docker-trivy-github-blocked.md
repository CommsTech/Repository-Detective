# Docker build: Trivy download fails when GitHub/CDN is blocked

**Priority:** P2  
**Type:** deployment  
**Component:** Dockerfile, networking

## Summary

`INSTALL_EXTERNAL_TOOLS=true` installs Trivy from GitHub releases. On DNS-filtered hosts (storage.googleapis.com / github.com sinkholed), `go mod download` and Trivy curl return 404 or connection errors.

## Workaround (shipped)

1. Run `./scripts/vendor-deps.sh` so `vendor/` is in the build context (see `.dockerignore`).
2. Stage a local binary: `cp ~/.local/bin/trivy deploy/bin/trivy` before build (see `deploy/bin/README.md`).
3. Or hotfix a running container: `docker cp ~/.local/bin/trivy repository-detective:/usr/local/bin/trivy`

## Acceptance

- Image build succeeds on commstech build host without manual copy when network allows GitHub releases.
