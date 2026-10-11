# Docker image scanning — private beta

## Status

**Opt-in preview.** Disabled by default. Requires a native runner with the `container-scan` label.

## Enable (operator)

1. Install a native runner on each Docker host that should scan images.
2. Label the runner: `container-scan` (plus environment labels e.g. `prod-a`).
3. Mount Docker socket **only into the runner** if local daemon images are needed.
4. Provide registry credentials via env (`REGISTRY_AUTH_FILE` or `DOCKER_CONFIG`) on the runner — never in core config.
5. Set in core config:

```yaml
container_scanning_enabled: true
container_scan_require_runner: true
container_scan_allow_core_docker_socket: false
container_scan_create_issues: false
container_scan_default_policy: report_only
```

6. Add allowlists:

```yaml
container_scan_allowed_registries:
  - ghcr.io
  - docker.io
container_scan_allowed_runner_labels:
  - container-scan
```

## Workflow

1. Scan a repository (or open **Container Images** in UI).
2. Click **Discover images** — parses Dockerfile, compose, K8s manifests.
3. Select one or more images.
4. Click **Scan selected image**.
5. Review vulnerabilities, SBOM, digest, scanner coverage.
6. Enable issue filing only after report quality is validated.

## Multi-server

```text
Server A runner: docker,container-scan,prod-a
Server B runner: docker,container-scan,prod-b
Core: schedules by label, stores results centrally, no Docker access
```

## Limitations (beta)

- Core never mounts Docker socket.
- Missing Trivy/Grype/Syft on runner → degraded coverage warning, not a crash.
- Not all registries supported without credentials on runner.
- No automatic scan of every image on every server.
