# Repository Detective v0.1.0-beta.5

**Public Beta** — CAH trust-boundary hardening, issue-create race fix, SQLite test-mode, SBOM UX, learning/AI assist.

## Identity

| Field | Value |
|-------|-------|
| Version | `v0.1.0-beta.5` |
| Image digest | `sha256:e26caafb19e3252230119929bd89681feb6b91166b9427c8253bc0e372130215` |
| E2E forge | Gitea **1.22.3** (same baseline as beta.3/4) |
| License | AGPL-3.0-or-later |
| Release posture | Public beta — forge issue filing **on** by default (`auto_create_issues: true`); severity/confidence gates still apply |

## What's new since v0.1.0-beta.4

### CAH / AI trust boundary
- System-level **data-only** guardrails on auditor, analysis, attack-surface, debater, and PoC prompts
- Stable untrusted-file delimiters (`RD_UNTRUSTED_REPO_FILE_*`) around repository source
- OpenClaw advisory system prompt documents the same trust boundary for redacted packets

### Issue filing
- Default is **on** (`auto_create_issues: true`) — filing forge issues is the product path so findings can be corrected
- In-process serialization of fingerprint lookup → create (prevents duplicate forge issues under concurrency)
- Cross-process forge deduplication remains **NOT_PROVEN** (fingerprint + local mappings still apply; multi-process proof pending)
- Per-scan report-only dry-run remains available when you need a no-forge pass

### SQLite test reliability
- Test binaries select `journal_mode=MEMORY` **before** any WAL transition (fixes suite timeouts)

### Self-scan / learning / SBOM (dogfood)
- `golang.org/x/net` **v0.60.0** (CVE-2026-78669)
- SBOM detects `pyproject.toml` / `setup.py` / `setup.cfg`
- UI no longer claims “SBOM file missing on disk” for `sbom_no_supported_manifest`
- AI-assisted learning: advisory ingest → calibration auto-apply; U1000 blocked from auto-downgrade

## Install (container)

```bash
docker pull git.commsnet.org/commstech/repository-detective:v0.1.0-beta.5
# optional public mirror
docker pull ghcr.io/commstech/repository-detective:v0.1.0-beta.5
```

Also tagged `:v0.1.0-beta.5-all-in-one` when published via the release pipeline.

Pin with compose:

```bash
export RD_IMAGE=git.commsnet.org/commstech/repository-detective:v0.1.0-beta.5
docker compose up -d
```

## Known limitations (honest)

- Remediation PR execution remains **disabled by default**
- LLM auditors remain **off by default** (cost control); advisory AI is optional and budget-capped
- Cross-process issue idempotency and ClusterMGR restart/recovery proofs remain **NOT_PROVEN**
- Image signing remains **CHECKSUM_ONLY** (no cosign yet)
- CAH fallback scanner still reports fixture/example noise (zero weaponizability)

## Security

Report privately — see [SECURITY.md](https://github.com/CommsTech/Repository-Detective/blob/main/SECURITY.md).

## Mirror note

Canonical development history lives on Gitea. GitHub is the public mirror; tag SHAs may differ while trees match.
