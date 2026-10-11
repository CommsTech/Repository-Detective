# Container scan runner preparation

Recorded: 2026-06-10

## Test window config (temporary)

Applied via `config/config.yaml` overlay (reverted after demo):

| Setting | Value |
|---|---|
| `runner_delegation_enabled` | true |
| `container_scanning_enabled` | true |
| `container_scan_create_issues` | false |
| `container_scan_require_runner` | true |
| `container_scan_allow_core_docker_socket` | false |
| `container_scan_allowed_registries` | `docker.io` |
| `container_scan_allowed_runner_labels` | `container-scan` |

Core Docker socket: **not mounted**.

## Runner

| Item | Value |
|---|---|
| Runner ID | `rd-runner-container-scan` |
| Label (documented) | `container-scan` |
| Job types | `container_image_scan` only |
| Core URL | `http://127.0.0.1:8081` |
| Docker socket on runner host | available (`/usr/bin/docker`) — not used for registry pull of `alpine:3.20` |
| Registry credentials | none |

## Scanner tools (runner host)

| Tool | Path | Version |
|---|---|---|
| trivy | `~/.local/bin/trivy` | 0.68.2 |
| grype | `~/.local/bin/grype` | 0.114.0 |
| syft | `~/.local/bin/syft` | 1.45.1 |

## Rollback steps

1. Stop native runner process (`pkill -f repository-detective-runner` or kill PID).
2. Revert `config/config.yaml` test block; set `runner_delegation_enabled: false`.
3. Restart `repository-detective` container (no test overlay).
4. Confirm `/api/v1/containers/config` shows `enabled: false`.
