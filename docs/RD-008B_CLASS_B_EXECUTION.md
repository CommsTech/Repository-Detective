# RD-008B — Class-B Remediation Execution Decision

**Date:** 2026-09-04  
**Status:** DECIDED — **Option C (Explicit unsafe/local execution mode)** for Community beta  
**Gate:** Must remain satisfied before expanding remediation UX under RD-015.

## Answers (current reachable behavior)

| # | Question | Finding |
|---|----------|---------|
| 1 | Can repository-controlled commands execute on the RD control-plane host/container? | **YES, when remediation PR validation runs.** `patcher.Executor.Run` clones the repo workspace and executes **allowlisted** validation commands via `RunAllowedCommand` (`go test`/`go vet`, `staticcheck`, `hadolint` only). Repository-supplied arbitrary shell is rejected by the allowlist parser — but allowlisted tools still run **on the control plane** against untrusted tree content. |
| 2 | What credentials/environment are visible? | Validation uses `security.MinimalSubprocessEnv()`-style constraints on some paths; historical risk was full env inheritance. Operator secrets must not be passed; treat residual leakage as **PARTIAL**. Forge tokens used for clone/push are handled by git helpers with redaction on errors. |
| 3 | What filesystem is visible? | The ephemeral clone workspace under RD data/tmp (plus tool caches under `$HOME`/XDG). Not a full host mount by design, but **not** a proven isolated FS namespace. |
| 4 | Is network access available? | **Yes by default** for the control-plane process network namespace. `go test` may download modules; scanners may fetch DBs. No product-enforced network deny for Class-B validation today. |
| 5 | Can tests/builds spawn arbitrary child processes? | Allowlisted commands can spawn children of those tools (e.g. `go test` compiling packages). Not arbitrary `bash -c`, but **not** a full sandbox. |
| 6 | Does remediation prefer/require a runner? | Config flag `remediation_pr_use_runner_verification` exists and is exposed in Configure UI, but **`Executor.Run` does not currently route validation to the runner**. Job type `remediation_verify` exists on runners (**CODE_PRESENT**) but is **not wired** as the mandatory path. |
| 7 | Can an operator enable Class-B without realizing isolation level? | **Yes.** Enabling Remediation PR without reading SECURITY_MODEL can look like “safe automation.” Doctor now reports Class-B as **NOT_PROVEN** and warns when PR remediation is enabled. |
| 8 | Smallest safe Community boundary? | Keep Class-B **disabled by default** (`remediation_pr_enabled=false`). Require explicit opt-in. Strong warnings. Prefer dedicated runner. **Do not claim sandboxing.** |

## Decision: Option C

**Chosen:** Explicit unsafe/local execution mode.

### Rules for Community beta

1. Remediation PR creation remains **off by default**.
2. When enabled without a proven isolated runner path, UI/docs/doctor must state **Class-B isolation: NOT_PROVEN** and that validation may execute on the control plane.
3. Never characterize allowlisting as a sandbox.
4. Onboarding must **not** silently enable remediation.
5. Prefer recommending a dedicated runner host; Option A (required isolated runner) is the follow-up hardening target once runner verification is actually wired end-to-end.
6. Option B (embedded ephemeral sandbox) is **NOT_IMPLEMENTED** and must not be advertised.

### Proof levels

| Capability | Level |
|------------|-------|
| Allowlist parser | UNIT_TESTED |
| Local validation on control plane | WIRED |
| Runner `remediation_verify` job type | CODE_PRESENT (not mandatory) |
| Product-enforced Class-B sandbox | NOT_IMPLEMENTED / NOT_PROVEN |
| Doctor Class-B disclosure | WIRED + UNIT_TESTED |

### Phase 5 implication

RD-015 must not expand remediation execution UX until either:

- Option A is implemented (runner required + isolation criteria), or
- Option C warnings remain prominent and defaults stay off.

See also: [SECURITY_MODEL.md](SECURITY_MODEL.md).
