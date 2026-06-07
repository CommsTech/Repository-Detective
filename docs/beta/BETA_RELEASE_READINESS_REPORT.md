# Beta release readiness report

Generated: 2026-06-07  
Product: Repository Detective — Inspect. Analyze. Improve.

## Verification summary

| Check | Result |
|-------|--------|
| `go test ./...` | **PASS** (Docker golang:1.23-bookworm) |
| `go vet ./...` | **PASS** |
| staticcheck | **Skipped** — install failed (IPv6 proxy unreachable in CI container) |
| gosec | **Ran** — 35 findings (pre-existing baseline; exit 0) |
| `make beta-release` / build script | **PASS** — `dist/repository-detective-beta/` with binary + checksums |
| Docker full rebuild | **Not re-run this sprint** — prior shellcheck gap until image rebuild |
| Public repo safety | **PASS** — no secrets in git index; `.env` gitignored |
| Operator smoke | **Not run** — requires live homelab stack |

## Feature status

| Item | Status |
|------|--------|
| Pre-install audit page | Fixed — always registered; disabled shows banner |
| Configure page | Fixed — `/ui/configure` |
| Setup diagnostics in primary nav | Removed post-setup; wizard link when incomplete |
| Feature flag matrix | Documented |
| SBOM generation/checking | Implemented + unit tests |
| Risk map segmentation | Stacked category chart on dashboard |
| Favicon | `favicon.svg` in layout |
| Project grouping | DB + UI beta |
| Per-repo calibration policy | Documented; matcher already repo-scoped |
| Cursor Bugbot comparison | Published (evidence-based, no superiority claims) |

## Product repo baseline (unchanged policy)

| Gate | Status |
|------|--------|
| Open issues | 1 (#48 operator task) |
| Active-present findings | 0 |
| Limited issue filing | **NOT approved** |
| All-repo scan | **NOT started** |
| Backlog-control | active |
| Report-only dry-run | available |

## Beta package

```text
dist/repository-detective-beta/
  repository-detective   (~20MB)
  checksums.txt
  config.example.yaml
  docker-compose.beta.yml
  README_BETA.md
  RELEASE_NOTES.md
```

Note: `sbom-go.cdx.json` included only when `cyclonedx-gomod` is on PATH during build.

## Remaining blockers

1. Docker image rebuild to pick up shellcheck in scanner image
2. staticcheck in CI (network/tooling)
3. Ruff finding gating for Python homelab repos
4. Live benchmark vs Cursor Bugbot (fixture not executed)
5. Gitea Actions release path still flaky — manual beta path is fallback

## Go / no-go

**Recommendation: private beta ready (internal/homelab testers)**

- **Not** public beta ready until gitleaks CI gate + docker rebuild verified + benchmark fixture run
- **Not** no-go — core UI blockers fixed, tests green, manual release path works

## Next recommended batch

1. Full Docker rebuild verify + deploy shellcheck
2. Controlled benchmark fixture repo vs Cursor Bugbot on same PR
3. More report-only dry runs after ruff gating
4. Enable pre-install audit in staging and smoke end-to-end
