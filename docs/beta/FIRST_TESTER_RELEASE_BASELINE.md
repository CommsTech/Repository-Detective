# First tester release baseline

Generated: 2026-06-02

## Latest commit

`0e33258` — docs(beta): publish private beta release go-no-go

Private beta packaging sprint complete (`de6122b` … `0e33258`).

## Product repo state

| Gate | Status |
|------|--------|
| Open issues (Gitea) | 1 (#48 operator task) |
| Active-present findings | **0** |
| Non-product issue filing | Disabled by default |
| All-repo scan | Not started |
| LLM sanity gate | Disabled by default |
| Report-only dry-run | Available |

## Beta package status

| Item | Status |
|------|--------|
| Local bundle | `dist/repository-detective-beta/` (built at `0e33258` tree) |
| Secrets check | PASS (prior sprint) |
| Checksums | Present |
| SBOM | Optional — `cyclonedx-gomod` not installed at build |
| Config shipped | `config/private-beta.example.yaml` via `config.example.yaml` |

## Live homelab container (pre-deploy)

| Item | Value |
|------|-------|
| URL | `http://127.0.0.1:8081` |
| Container | `repository-detective` |
| Network | host |
| Image revision (stale) | `f64789d` (built 2026-06-06) |
| API `/health` | healthy |
| `/ui/configure` | **404** (stale binary) |
| `/ui/learning` | **404** (stale binary) |
| Static assets | Stale / missing routes |

**Action required:** rebuild and redeploy from current `main` before first testers.

## Safety defaults (unchanged)

| Control | Default |
|---------|---------|
| Issue filing | Off (`auto_create_issues: false` in beta config) |
| First scan | Report-only dry-run |
| Remediation PRs | Off |
| LLM sanity gate | Off |
| Runner delegation | Off |
| Notifications | Off |
| Evidence closure | On |
| Backlog control | On |

Config: `config/private-beta.example.yaml`, `docker-compose.beta.yml`

## Git hygiene

| Check | Status |
|-------|--------|
| `.env` staged | No |
| `dist/` staged | No (gitignored) |
| Local `repository-detective` ELF staged | No |
| Working tree | Clean |

## Remaining risks

1. Live container must be rebuilt — UI routes unavailable on current image
2. Full `docker-build-verify.sh` not re-run every sprint (~23 min)
3. SBOM optional in tester bundle
4. CI green status may lag local verification
5. First testers must stay report-only until explicit approval

## Prior verification (reference)

- [PRIVATE_BETA_SMOKE_TEST_REPORT.md](PRIVATE_BETA_SMOKE_TEST_REPORT.md)
- [PRIVATE_BETA_RELEASE_GO_NO_GO.md](PRIVATE_BETA_RELEASE_GO_NO_GO.md)
- [PRIVATE_BETA_TESTER_GUIDE.md](PRIVATE_BETA_TESTER_GUIDE.md)
