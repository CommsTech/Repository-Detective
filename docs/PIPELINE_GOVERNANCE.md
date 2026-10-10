# Pipeline and runner governance (Gitea #46)

Repository Detective supports **reviewing pipeline-as-code** and **runner delegation** — not a full Gitea Actions marketplace audit.

## Static checks on workflow files

When `.gitea/workflows/*` or `.github/workflows/*` are in scan scope, static rules flag:

| Rule | Risk |
|------|------|
| `GOV-ACTION-FLOATING-REF` | Third-party action uses `@v4` / `@main` instead of immutable SHA |
| `GOV-PIPELINE-SECRET-ECHO` | `echo` / `printf` may print secrets in logs |

Enable security static analysis (`enable_security` / standard profile).

## Runner delegation

See [RUNNERS.md](RUNNERS.md):

- Ephemeral runners recommended
- `runner_shared_secret` for callbacks
- Isolate production runners from dev

## This repository CI

[`.gitea/workflows/ci.yml`](../.gitea/workflows/ci.yml) runs lint, test, govulncheck, and Docker build smoke. Pin third-party actions to commit SHAs when your forge policy requires it.

## Operator actions

1. Require PR review for workflow changes
2. Store CI secrets in forge secret store, not YAML
3. Mask secrets in job logs
4. Audit `uses:` lines periodically
