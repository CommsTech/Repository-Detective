# Known limitations

Honest constraints as of the closeout sprint. Update when shipping fixes.

## Product

- Single-tenant operator model — no built-in multi-user RBAC in UI
- SQLite default — not HA clustered DB
- Issue dedup is fingerprint-based (monitor SQLite mappings + forge issues)
- Remediation PRs off by default

## Scanners

- Optional binaries may be missing — **degraded coverage**, not hard failure
- Version strings may be unparsed (`unknown`) or vendor-specific (`dev`)
- Vendor directories may produce noise — use `suppress_vendor` in profile
- Scanner stderr may contain sensitive paths — log redaction is heuristic and incomplete on some scanners

## Privacy

- Redaction patterns do not catch all secret formats
- Gitea issues already created are not retroactively redacted
- LLM prompts may include code if enabled — administrator must disable
- Reports/exports may contain sensitive snippets — treat as confidential

## Accessibility

- No VPAT; manual testing checklist only
- Charts require supplementary text tables (provided on dashboard)
- No keyboard shortcut layer

## Documentation / wiki

- Wiki is **not** auto-synced on every commit — publish with `scripts/publish-gitea-wiki-api.py` (preferred) or `scripts/publish-gitea-wiki.sh`
- Live wiki: https://git.commsnet.org/commstech/repository-detective/wiki
- Prefer [QUICKSTART.md](QUICKSTART.md) for new operators; [SETUP.md](SETUP.md) for full walkthrough
- Gitea HTML/wiki slug is lowercase `repository-detective`; see [NAMING.md](NAMING.md) and [WIKI_PUBLISHING.md](WIKI_PUBLISHING.md)

## Scanner / image (homelab)

- All-in-one image must ship a Go toolchain matching `go.mod` (currently **1.25**). Older container Go (e.g. 1.23) causes `cyclonedx-gomod` / staticcheck / golangci friction; Syft is the SBOM fallback.
- Grype vulnerability DB can become malformed on long-lived volumes — rebuild with the **same** `XDG_CACHE_HOME` the container uses (live default `/app/data/cache`): `docker exec -u root … rm -rf /app/data/cache/grype && chown …` then `XDG_CACHE_HOME=/app/data/cache grype db update`. Updating only `$HOME/.cache/grype` will not fix scanner runs.

## Beta validation

- [TEST_MATRIX.md](TEST_MATRIX.md) — regression areas
- [BETA_SMOKE_TEST.md](BETA_SMOKE_TEST.md) — end-to-end checklist
- [DOCS_AUDIT.md](DOCS_AUDIT.md) — doc completeness

## Integrations

- Gitea-specific; GitHub/GitLab not first-class
- Runner delegation requires separate runner deployment

## Tracking

See [issues/README.md](issues/README.md) and [ISSUE_BACKLOG.md](ISSUE_BACKLOG.md).
