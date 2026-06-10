# Docker image scanning architecture

Repository Detective scans **container images** via **native runners**, not by mounting the Docker socket into the core application.

## Supported scan targets

| Target type | Description |
|---|---|
| `registry_image` | Remote registry reference (`ghcr.io/org/app:tag`, digest-pinned preferred) |
| `local_docker_image` | Image already present on a runner host (`docker images`) |
| `compose_file` | Images referenced in `docker-compose.yml` / `compose.yaml` |
| `kubernetes_manifest` | Images in Kubernetes/Helm YAML manifests |
| `runner_host_inventory` | Explicit opt-in inventory of local daemon images on a registered runner |

## Safe execution model

```text
Repository Detective core
  → stores targets and discovered references
  → enqueues container_image_scan jobs
  → receives normalized results
  → correlates findings with repos/services
  → files issues only when policy allows

Repository Detective native runner (label: container-scan)
  → optional Docker socket or registry credentials
  → pull/inspect image per policy
  → runs Syft / Trivy / Grype
  → uploads SBOM + vulnerability findings
  → redacts secrets; enforces timeout/size limits
```

## Image sources

1. **Registry images** — `ghcr.io`, `docker.io`, private registries (credentials via env/mount only).
2. **Local Docker daemon** — images on runner host; requires explicit opt-in and runner with socket.
3. **Compose / Stack / Kubernetes** — deterministic parsers extract references from repo files; scan jobs created per image.

## Security requirements

- Docker socket access **only on runner**, never required on core.
- `container_scanning_enabled=false` by default.
- `container_scan_require_runner=true` by default.
- `container_scan_allow_core_docker_socket=false` by default.
- Registry credentials via `REGISTRY_AUTH_FILE` / `DOCKER_CONFIG` env on runner only — **never in job payload or logs**.
- Pull policy configurable (`never`, `if_missing`, `always`).
- Scan timeout and max image size enforced.
- Remote hosts require explicit runner registration with labels (e.g. `container-scan`, `prod-a`).
- Scans scoped by registry/image allowlists.

## Finding rules

| Rule | Meaning |
|---|---|
| `CONTAINER-IMAGE-REFERENCE` | Discovered image reference |
| `CONTAINER-MUTABLE-TAG` | Tag is mutable (`latest`, no digest) |
| `CONTAINER-NO-DIGEST` | No digest pin in deployment manifest |
| `CONTAINER-UNSCANNED-IMAGE` | Reference discovered but not yet scanned |
| `CONTAINER-VULNERABLE-IMAGE` | CVE from Trivy/Grype on scanned image |
| `CONTAINER-SBOM-GENERATED` | SBOM artifact stored |

## Issue filing

Default: **report-only**, no issues created. Enable `container_scan_create_issues` only after report quality is proven.

See also: [beta operator guide](beta/DOCKER_IMAGE_SCANNING_BETA.md).
