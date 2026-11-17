# Pre-staged scanner binaries (optional)

On networks where GitHub release downloads are blocked during `docker build`, copy static binaries here before building **runner** or **all-in-one** images:

| File | Tool |
|------|------|
| `trivy` | Trivy |
| `grype` | Grype |
| `gitleaks` | Gitleaks |
| `hadolint` | Hadolint |

```bash
chmod +x deploy/bin/trivy
docker build --target all-in-one --build-arg INSTALL_EXTERNAL_TOOLS=true .
```

The install script (`scripts/install-scanner-tools.sh`) uses staged binaries when present; otherwise it downloads pinned versions documented in [docs/DOCKER.md](../docs/DOCKER.md).
