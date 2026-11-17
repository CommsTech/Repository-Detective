# Gitea Actions backend — implementation status

**NOT OPERATIONAL as an end-to-end Repository Detective execution backend (2026-11-17).**

This repository contains a Gitea Actions workflow template and a request-builder structure, but RD does not yet provide the necessary workflow dispatch, run-status polling, artifact/result ingestion, commit/policy binding, and final independent verification to treat an Actions job as completed remediation evidence. Configuration options named `gitea_actions_*` are **not** evidence that that backend is live.

Gitea `act_runner` itself is a separate CI service and can execute repository workflows independent of Repository Detective. CI on a shared or privileged runner can execute untrusted project code and must not be used as a security sandbox without separate host-level isolation and strict network, secret and privilege controls.

Use the [native worker](RUNNERS.md) for supported delegated RD jobs, or manually run the checked-in `.gitea/workflows/repository-detective-verify.yml` in a properly isolated Gitea Actions environment as **external CI**. Do not automatically accept external CI as verified closure without checking the target commit, workflow identity, results and artifact integrity.

## Work required before activation

Implement and test: authenticated dispatch, job correlation and expiry, least-privilege tokens, exact commit pinning, repository-scope constraints, Gitea API error/retry handling, safe workflow permissions, status monitoring, signed/validated artifact ingestion, failure and cancellation semantics, per-job resource isolation, and evidence-backed closure. Include adversarial tests for forged results, mutable branch refs, malicious workflow options, missing artifacts and runner compromise.

See [Security, sandboxing and trust](SECURITY_SANDBOX_AND_TRUST.md) and [Release validation](release/RELEASE_VALIDATION_2026-11-17.md).
