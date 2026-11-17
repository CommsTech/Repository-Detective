# Runner delegation — operator quick reference

**Native worker:** supported, opt-in, requires shared HMAC secret and a reachable authenticated core endpoint. Manual and scheduled scans may be delegated under `native` or `auto` policy. If no worker is available, the queued job remains pending until completion/expiry; if enqueuing fails, the operation fails. It **does not** silently execute on the core.

**Gitea Actions / GitHub Actions:** do not enable as Repository Detective job-dispatch backends; the complete dispatch/result/verification pipeline has not been implemented. Forge CI workflows may still run independently, with their own trust and isolation requirements.

## Compose deployment example

Run the core plus the native-worker overlay from the repository root:

```bash
cp .env.example .env
# Configure forge credentials and a strong runner-only HMAC secret in .env.
# Set runner_delegation_enabled=true and runner_mode=native on core.
docker compose -f docker-compose.yml -f docker-compose.runner.example.yml config
# Review effective config, mounts and credentials before starting services.
docker compose -f docker-compose.yml -f docker-compose.runner.example.yml up -d
```

The overlay runs as UID/GID 1001, with read-only root, no Docker socket, CPU/memory/pid limits and temporary storage. **This is not a security boundary for actively hostile source code.** The worker can still reach the core and forge over its bridge network. Place it on an isolated host or disposable VM for third-party/repository supply-chain investigations. Private repo clone authentication requires explicit design; do not embed access tokens in clone URLs.

Verify worker heartbeats, job status, exact repository revision, scanner completeness, result correlation and timeout behavior. Record failures as failures. See [Runner operations](RUNNERS.md), [Sandbox and trust](SECURITY_SANDBOX_AND_TRUST.md), and [Quickstart](RUNNER_WORKER_QUICKSTART.md).
