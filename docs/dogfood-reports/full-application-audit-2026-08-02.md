# Full application audit — 2026-08-02

**Product:** Repository Detective  
**Live:** `rc-full-audit7` on `repository-detective:rc-sbom-tools`  
**Wiki:** https://git.commsnet.org/commstech/repository-detective/wiki  

## Verdict

**Conditional GO** for private beta. Accuracy metrics, operator docs, and wiki now match live behavior. Remaining gap: rebuild the all-in-one image with Go **1.25** so cyclonedx-gomod / staticcheck / golangci-lint match `go.mod`.

## Fixes shipped

| Area | Fix |
|------|-----|
| Dashboard trust | `scanner_tools_missing_count` uses **live** tool probes (was historical `10` while health was 12/12) |
| ShellCheck | Flat JSON 0.10 parser → **found** |
| Trivy | `--output` + per-scan `--cache-dir` → **found** |
| SBOM | Syft fallback after cyclonedx-gomod → **`sbom_generated` / 58 packages** |
| Grype | Corrupt DB lived under container `XDG_CACHE_HOME=/app/data/cache` (not `$HOME/.cache`); rebuilt there → **found** |
| Subprocess env | Ensure HOME / cache vars for scanner DBs |
| Docs / wiki | Full wiki pages; API publisher; OpenAPI corrections; **24 pages** live |

## Verification highlights

| Check | Result |
|-------|--------|
| `/health` | 12/12 tools, healthy |
| Dashboard tools missing | **0** |
| Legacy Bugbot API header | **401** |
| UI routes | `/ui`, health, findings, learning, configure, reports, scans, repos → **200** |
| Wiki | https://git.commsnet.org/commstech/repository-detective/wiki |
| Product scan (pre-timeout stress) | shellcheck/trivy/SBOM OK; after cache fix **grype found** |

## Residual

1. **Image Go 1.23 vs go.mod 1.25** — rebuild `repository-detective:all-in-one` from current Dockerfile.  
2. **golangci-lint / staticcheck** noise until that rebuild.  
3. **14d parse_failed counter** ages out historically.  
4. Fleet finding backlog — triage via learning/calibration (existing).

## Evidence scans

- `dc5054dba074ae6e` — trivy found; SBOM generated (58 pkgs)  
- `162855c88c51017f` — same + tools_missing 0  
- `ea52b8d69e63c7f2` — **grype found** after `/app/data/cache` DB rebuild (scan also hit analysis timeouts under load)
