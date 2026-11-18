# Runner operations — current implementation

> **2026-11-17 correction:** Earlier versions of this guide described the Gitea Actions backend as complete and an `auto` fallback to core. Neither assertion is safe or supported by the current code. This page supersedes that guidance; check release evidence before production deployment.

Repository Detective has a **native worker** implemented with an authenticated polling/job/result protocol. A native worker may offload scans, graph, SBOM and validation workloads under operator-controlled policies. Gitea Actions workflow files exist, but fully verified dispatch → status → artifacts → evidence ingestion is **not implemented**. GitHub Actions integration is similarly not an operational RD execution backend.

## Architecture

The core holds policies, credentials, job queue, repository records, findings, notifications and issue/PR state. The native worker polls with an HMAC-shared secret, receives a bounded specification, checks out the target commit, executes supported deterministic tasks, and returns normalized findings and status. The core validates the callback before ingesting. **Never transfer administrative Gitea credentials or AI credentials into a repository-controlled job specification.**

For new deployments:

1. Keep delegation disabled during initial baseline setup; run Doctor and verify scanner availability.
2. Configure a dedicated native worker using [the worker quickstart](RUNNER_WORKER_QUICKSTART.md) and the hardened-but-not-fully-isolated [Compose overlay](../docker-compose.runner.example.yml).
3. Generate a strong independent `REPOSITORY_DETECTIVE_RUNNER_SHARED_SECRET` and set the same secret securely on core and worker.
4. Enable `runner_delegation_enabled: true`, select `runner_mode: native`, and choose `runner_policy: native` (or `auto`) for the repository you intend to delegate. Manual and scheduled full scans are currently the supported automatic delegation triggers.
5. Confirm worker heartbeat and `queued → running → completed`, exact commit, scanner statuses, and failed-job handling using an **isolated, non-sensitive test repository**.
6. Treat failures as failures. Delegation errors do **not** silently fall back to core. Previously queued `gitea_actions` jobs are not claimed by native workers.

**Important:** An always-on Docker worker is not a disposable sandbox. To handle malicious repositories, run the worker itself in an isolated, disposable VM or equivalent environment with network policy and ephemeral credentials. Do not expose the host Docker socket. See [Threat model and evidence](SECURITY_SANDBOX_AND_TRUST.md).

Gitea Actions example workflows may be executed separately by Gitea as part of normal forge CI, but the app must not report them as verified RD job results without real traceable ingestion. This is an **open engineering milestone**.

## Scanner image release gate

Build the worker image, then run `sh /path/to/scripts/acceptance-scanner-image.sh` inside it and follow with scanner-specific known-positive/negative fixtures. The install script contains older pinned scanner versions. Missing tools, stale databases, unsupported output formats or untrusted image provenance must fail promotion; mere presence of a CLI is insufficient.
