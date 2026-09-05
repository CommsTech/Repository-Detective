# Private beta package verification

Date: 2026-06-02  
Commit: `de6122b` (build includes uncommitted config/docker updates until Phase 4 commit)

## Build commands

```bash
make clean-beta-release
make beta-release
./scripts/check-beta-package-secrets.sh
```

## Package contents

```text
dist/repository-detective-beta/
  repository-detective      # ELF ~20MB, commstech-owned
  checksums.txt
  config.example.yaml       # from config/private-beta.example.yaml
  docker-compose.beta.yml   # dedicated beta compose with safe env overrides
  README_BETA.md
  RELEASE_NOTES.md
  .env.example              # placeholders only
```

## Verification checklist

| Check | Result |
|-------|--------|
| Builds as current user | PASS (Docker golang:1.23-bookworm fallback; go not on host PATH) |
| checksums.txt present | PASS |
| config.example.yaml safe | PASS — empty tokens, `auto_create_issues: false` |
| docker-compose.beta.yml safe | PASS — `AUTO_CREATE_ISSUES=false`, remediation PR off |
| No live `.env` in package | PASS |
| No repository-detective.db in package | PASS |
| No local repository-detective ELF beyond release binary | PASS |
| `check-beta-package-secrets.sh` | PASS |

## SBOM

| Item | Status |
|------|--------|
| sbom-go.cdx.json / sbom.spdx.json | Not generated — `cyclonedx-gomod` not installed at build time |
| Policy | Optional for private beta; recommended before public beta |

## Secrets grep

```bash
grep -RInE '(REPOSITORY_DETECTIVE_GITEA_TOKEN|REPOSITORY_DETECTIVE_API_KEY|AKIA|BEGIN RSA|BEGIN OPENSSH|password|secret|token)' \
  dist/repository-detective-beta
```

Result: only safe placeholder references in `.env.example`, README, and empty config fields. No real credentials.

## config.example.yaml safety highlights

| Setting | Value |
|---------|-------|
| `auto_create_issues` | `false` |
| `remediation_pr_enabled` | `false` |
| `llm_sanity_gate_enabled` | `false` |
| `enable_llm_auditors` | `false` |
| `runner_delegation_enabled` | `false` |
| `notifications_enabled` | `false` |
| `evidence_closure_enabled` | `true` |
| `dogfood_backlog_control_enabled` | `true` |
| `reporting.max_issues_per_scan` | `0` |

## Artifacts not committed

`dist/` is gitignored — built package is local-only for operator distribution.
