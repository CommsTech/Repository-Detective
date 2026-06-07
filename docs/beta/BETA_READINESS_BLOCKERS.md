# Beta readiness blockers

Generated: 2026-06-07  
Product: Repository Detective — Inspect. Analyze. Improve.

## Product baseline

| Check | Status |
|-------|--------|
| Open product issues | 1 (#48 operator task) |
| Active-present findings | 0 |
| Backlog-control | active |
| Report-only dry-run | available |
| Limited issue filing | **NOT approved** |
| All-repo scan | **NOT started** |
| Latest commit (start) | `c344f6b` |

## Blocker inventory

| Blocker | Owner area | Status | Test required | Release impact |
|---------|------------|--------|---------------|----------------|
| CI/release flakiness | CI / release | open | `docker-build-verify.sh`, Gitea Actions | High — blocks automated release |
| Prepackaged beta artifacts | release | in progress | `make beta-release` | High — operator manual path |
| Public repo safety audit | security / docs | in progress | gitleaks, grep, doc review | **Blocker** for public beta |
| AI-code/slop audit | engineering | in progress | `go test`, staticcheck, manual review | Medium |
| Pre-install audit page 404 | UI | **fixed** | GET `/ui/preinstall` 200 | High |
| Configure page not displaying | UI | **fixed** | GET `/ui/configure` 200 | High |
| Setup diagnostics visible after setup | UI | **fixed** | nav hides wizard link post-setup | Medium |
| Feature flags disabled/untested | platform | **documented** | FEATURE_FLAG_TEST_MATRIX | High |
| Runner delegation disabled/untested | runner | open | smoke with secret + callback | Low for internal beta |
| Notifications disabled/untested | notify | open | webhook smoke | Low for internal beta |
| Pre-install audit disabled/untested | preinstall | open | enable + smoke audit | Medium |
| Remediation PR disabled/untested | remediation | open | controlled test only | Low — disabled by default |
| SBOM generation/checking | scanners | **implemented** | unit + scan integration | High for supply-chain story |
| Risk map category segmentation | UI | **done** | dashboard stacked chart | Medium |
| Favicon missing | UI | **done** | layout head link | Low |
| Project grouping / multi-repo | store / UI | **beta** | CRUD + report smoke | Medium |
| Docker full rebuild reliability | docker | open | `docker-build-verify.sh` | High |
| Scanner gaps (grype DB, shellcheck) | docker | partial | scanner status on health | Medium |
| Per-repo continuous calibration | store | **documented** | calibration scope tests | Medium |
| Cursor Bugbot comparison | docs | **done** | benchmark plan doc | Medium — positioning |
| Ruff finding gating (post-install) | profile | open | homelab dry-run | Medium |

## Gate decisions (unchanged until this sprint completes)

- Limited issue filing: **blocked**
- All-repo scan: **blocked**
- Non-product issue filing: **blocked**

## Sprint exit criteria

1. No 404 on pre-install or configure pages.
2. Public safety audit passes with no committed secrets.
3. `make beta-release` produces checksums + SBOM without committing binaries.
4. Feature flag matrix documents every flag with pass/fail or explicit beta default.
5. Evidence-based Cursor Bugbot comparison published (no unverified superiority claims).
