# Live pre-install sandbox audit report

Recorded: 2026-06-08

## Prerequisites

| Item | Value |
|------|-------|
| Image revision | `9ed0898` + hot-swap binary `0a06a82` (double-.git URL fix) |
| `git` in container | yes — 2.45.4 |
| Pre-install enabled | yes |

## Private IP blocking

| URL | Result |
|-----|--------|
| `https://192.168.1.1/admin/repo.git` | blocked — private/local network |

## Safe public repo audit

| Field | Value |
|-------|-------|
| Audit ID | `061578b7-2df7-4d13-b79f-69e0265928f7` |
| Repo | `https://github.com/octocat/Hello-World.git` |
| Status | **completed** |
| Commit SHA | `7fd1a60b01f91b314f59955a4e4d4e80d8edf11d` |
| Recommendation | safe |
| Issues created | 0 |
| PRs created | 0 |

## Sandbox metadata (from summary_json)

| Field | Value |
|-------|-------|
| Sandbox ID | `18b74007bbb6` |
| Enabled | true |
| Clone mode | shallow-single-branch-no-tags |
| Submodules disabled | true |
| Read-only workspace | true |
| Private IP blocked | true |
| Workspace files / bytes | 29 / 24269 |
| Workspace path | `/tmp/rd-preinstall-18b74007bbb6-…/repo` |

## Workspace cleanup

Temp workspace directory **not present** after audit completion (defer cleanup verified).

## Report — Sandbox and Safety section

Install risk summary report includes:

- Sandbox enabled, clone mode, limits, timeout
- Issue/PR creation: 0
- Disclosure: operator approval required
- “Repository Detective did not execute this repository's code…”
- Sandbox ID recorded

## Bug fixed during verification

URLs ending in `.git` previously produced `*.git.git` clone URLs, causing clone failure even with `git` installed. Fixed in `0a06a82`.

## Code execution

No install/build/test commands run; scanners only (several binaries missing in image — pre-existing).
