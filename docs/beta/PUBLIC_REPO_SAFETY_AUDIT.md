# Public repository safety audit

Generated: 2026-06-07  
Scope: prepare `/home/commstech/Repository-Detective` for external review without leaking homelab secrets.

## Scan commands run

| Tool | Result |
|------|--------|
| `gitleaks detect` | Not installed on host — manual grep + `.gitignore` review used |
| `grep` for token patterns | No live tokens in tracked files |
| `find … -size +5M` | Large files are vendored sqlite libs only (expected) |

## Findings and disposition

| Item | Status | Action |
|------|--------|--------|
| `.env` with live token | **Not tracked** | Listed in `.gitignore`; operators use `.env.example` |
| `config/config.yaml` | **Not tracked** | `.gitignore` protects local config |
| `repository-detective-build` / `repository-detective` ELF | **Not tracked** | Added to `.gitignore` |
| `data/*.db` | **Not tracked** | `.gitignore` |
| Dogfood raw exports | **Ignored by default** | Whitelist only sanitized reports |
| `deploy.sh` env var names | Safe | References env names only, no values |
| `.env.example` | Safe | Placeholder values only |
| Docs with example keys | Safe | `change-me`, `your-key`, `REDACTED` patterns |
| Internal hostnames in dogfood | Partial | Whitelisted reports use redacted/sample names; unsanitized exports stay local |

## README / policy files

| File | Status |
|------|--------|
| `README.md` | Uses public-safe install instructions |
| `.env.example` | Documents secrets via environment |
| `SECURITY.md` | Present (if missing, add before public beta) |
| `LICENSE` | Verify before public release |

## Acceptance

- No committed secrets in git index at sprint time.
- No live credentials in tracked docs or scripts.
- Homelab references in committable dogfood reports are redacted or clearly sample-only.
- External testers can clone and configure via `.env.example` without inheriting operator tokens.

## Residual risks

1. Operator must never `git add .env`, local DBs, or `repository-detective-build`.
2. New dogfood exports default to gitignored — sanitize before whitelisting.
3. Install gitleaks in CI for ongoing public-repo gate.
