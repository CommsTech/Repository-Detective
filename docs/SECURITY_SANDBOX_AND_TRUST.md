# Execution isolation and trust evidence (2026-11-17 working tree)

## What runs where — current code

| Operation | Current execution path | What is **not** implemented or proven |
|---|---|---|
| API, UI, database, policy, issue lifecycle | Repository Detective **core** | Core is not a safe environment for executing arbitrary repository tests |
| Manual/scheduled delegated scan | Native RD worker, **only** when operator enables delegation with HMAC secret, selects native/auto policy, and a worker is connected | No assurance of hostile-code containment from ordinary Docker alone |
| Repository-owned deterministic custom rules | Core scan or native worker, evaluated as non-executable Go RE2 patterns | No custom scripts or semantic AST rule language |
| Gitea Actions verification | Example workflow and request builder exist | **No complete workflow dispatch, run polling, artifact/attestation ingestion and result verification**; not a working backend for automatic RD jobs |
| GitHub Actions | GitHub can host workflows outside the application | No end-to-end GitHub-hosted RD scan/remediation dispatcher with verified callback |
| AI remediation | Optional advice/planning and existing narrowly scoped controlled patcher logic | No generally safe unattended dependency-upgrade/repair engine; no proven AI validation-token budget |
| Webhook notifications | Opt-in signed outbound webhook notification channel and event filters exist | Live retries, delivery guarantees, SSRF boundaries, and every event-type mapping still require staging integration tests |
| Re-scan/closure | Existing recorded scan and finding lifecycle | A clean partial scan must not confer trust; verify exact revision, scanner execution, and evidence |

**Fail closed:** When a delegated job cannot be enqueued, RD must not silently scan the repository on core because `runner_policy=auto`. Legacy `gitea_actions` jobs cannot be claimed as native jobs; unsupported Gitea Actions settings are not functional dispatch switches. Operators who explicitly select `core` accept that scans execute on the main instance and must never treat that as an adversarial sandbox.

## Threat model for untrusted repositories

An attacker can add source files, lockfiles, Git history, scanner configuration, malformed data, or runtime scripts to a repository. Attackers may exploit bugs in parsers/scanners, malicious build hooks, dependency installation, Git filters, or CI workflow options. A non-root Docker process with a read-only root filesystem **reduces exposure** but is not a hardened isolation boundary. Any accessible forge token, AI provider token, Docker socket, host network, credential helper, or writable host mount remains a target.

**Recommended for actively hostile repositories:** a fresh disposable VM/microVM for every scan; no host socket; one-job credentials with least privilege; restrict network egress to explicitly approved clone endpoints; separate initial clone network from offline analysis; time/CPU/memory/pid/disk bounds; mount readonly base image, use an ephemeral writable workspace; isolate scanner-generated artifacts; destroy workspace and secrets after completion; run result validation and source-controlled approval in core. Enable sandbox snapshots and signed traceable evidence where feasible.

`docker-compose.runner.example.yml` is an **illustrative lower-risk native worker** overlay. It removes host networking and the Docker socket, drops capabilities, sets non-root user, read-only root filesystem, resource limits, and ephemeral `/tmp`. It **still has network egress** on the Docker bridge and **does not create per-job VM isolation**. This example is not a production certification or safe for actively adversarial code.

### Authenticating and pinning the scan

- A worker clones only `https://` or `ssh://` targets (not shell-capable Git `ext::` or `file://` transports), disallows embedded passwords, and rejects suspicious command-line branch arguments.
- When the core supplies a full 40- or 64-character Git commit SHA, the worker verifies that the checked-out HEAD is that exact revision. A moving branch does not count as an authorized revision.
- Do not use branch names as cryptographic provenance. Retain repository identity, forge, commit, scanner versions, source/lockfile hashes, policy snapshot, skipped tools, and completion/error status with the evidence.
- A signature or attestation says **who built/checked this artifact**, not that the code is free from defects. Verify identities, trust anchors, provenance, and actual tests; do not display a blanket “safe to run” seal.

## Suggested trust evidence per repository

1. **Identity & integrity:** immutable commit, authoritative upstream and publisher, branch protection, signed tags/commits where verified, release hash.
2. **Supply chain:** SBOM, dependency lockfiles, dependency provenance and licenses, suspicious lifecycle scripts, known exploited vulnerabilities, and VEX claims with evidence.
3. **Analysis:** complete/partial scanner status for SAST, secrets, history, IaC, dependency/container scans, custom rules, architecture/code quality; tool versions and database freshness.
4. **Testing:** unit/integration/fuzz tests, coverage gaps, sanitizers, performance/loop testing, build reproducibility, resource budgets, and exact revision used.
5. **Remediation:** human-reviewed diff, isolated validation, PR and approval trail, merge commit, new scan, independent closure evidence.

Record missing data as **not checked**, timeouts as **incomplete**, and AI uncertainty as **advisory**. Do not promote a finding to false positive based solely on an AI label.

## Future acceptance gates

For a real Gitea Actions backend, require audited dispatch endpoints, a signed job/commit/policy spec, least-privilege scoped runner identity, durable job correlation, expiry and cancellation, artifact validation, normalized findings, replay protection, no privileged `container.options`, and a disposable-host acceptance test. Test forged callbacks, malicious workflows, mutated refs, partial output, scanner crashes, OOM/disk exhaustion, leaked secrets, and abandoned jobs. Review Gitea and act_runner security advisories before enabling any untrusted fork CI job.

See [Custom rules](CUSTOM_RULES.md), [Native runner quickstart](RUNNER_WORKER_QUICKSTART.md), [Notifications](NOTIFICATIONS.md), and [Release validation](release/RELEASE_VALIDATION_2026-11-17.md).
