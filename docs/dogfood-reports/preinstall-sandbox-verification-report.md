# Pre-install audit sandbox verification

Recorded: 2026-06-08

## Configuration (live)

| Setting | Value |
|---------|-------|
| `preinstall_audit_enabled` | true |
| `preinstall_sandbox_enabled` | true (default) |
| Report-only | yes — 0 issues, 0 PRs |
| Disclosure auto-submit | not enabled |

## Live tests

### Private / local network blocking

| URL | Result |
|-----|--------|
| `https://192.168.1.1/admin/repo.git` | **blocked** — "repository host resolves to a private or local network address" |
| `http://127.0.0.1:8080/malicious/repo.git` | **blocked** — non-HTTPS rejected first |

### Safe public repo clone audit

| Field | Value |
|-------|-------|
| Audit ID | `1b8b2c41-d3df-40bb-a2e3-614b2e364c21` |
| Repo | `https://github.com/octocat/Hello-World.git` |
| Status | **failed** |
| Error | `git clone failed: git operation failed` |

**Cause:** all-in-one container reports `git` missing from PATH (`/health` tools_summary). Clone isolation code path is deployed but cannot execute without `git` in the runtime image. This is a **deployment/tools gap**, not a sandbox logic regression.

## Unit test coverage (pass)

- Unique sandbox temp workspace per audit
- Path traversal / symlink escape blocked
- File count and size limits
- Clone argv: `--no-recurse-submodules`, `core.hooksPath=/dev/null`
- Scanner env does not expose operator secrets
- Retain-on-failure defer wiring

## Report sandbox section

`appendSandboxSection()` added to install-risk summary reports with:

- Sandbox enabled, clone mode, submodule policy
- Size/file/timeout limits
- Private IP blocking
- Issue/PR creation: 0
- Safety statements (no code execution, redacted secrets)

**Live report body:** not captured — audit did not complete clone on live container.

## Guarantees (by design)

- Repository Detective does not execute repository code during pre-install audit
- No package install/build/test from untrusted repos
- Workspace deleted after audit unless `preinstall_sandbox_retain_on_failure=true`
- Issues created: 0; PRs created: 0

## Recommended follow-up

Install `git` in all-in-one image (or document as required host dependency) and re-run live Hello-World audit to capture sandbox ID + report section end-to-end.
