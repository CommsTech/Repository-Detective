# Reducing false positives

Repository Detective combines **static heuristics**, **external scanners**, **repo health checks**, and optional **LLM auditors**. Heuristic rules are fast but can mis-classify safe patterns.

**Repo-structure awareness** (see [REPORTING.md](REPORTING.md)) runs before scanners and normalizes findings before Gitea issue creation. Generated, vendor, test, docs, and example paths are downgraded or suppressed by default while remaining auditable in scan data.

## Common false positives (fixed in static analyzer)

| Pattern | Example | Why it is usually safe |
|---------|---------|------------------------|
| Hardcoded secret in `deploy.sh` | `local api_key="${REPOSITORY_DETECTIVE_API_KEY:-}"` | Reads from environment, not a literal secret |
| `data-api-key` in HTML templates | `data-api-key="{{.APIKey}}"` | UI passes the operator API key to JS — not embedded credentials |
| SQL concat in Go store layer | `query := base + \` WHERE status = ?\`` | Appends a **constant** fragment with `?` placeholders |
| Orphan file warnings | `ui/templates/*.html`, `scripts/*.sh` | Not part of the Go import graph by design |

## Tuning policy

| Setting | Effect |
|---------|--------|
| `reporting.mode` | `high_signal` (default), `monitor_only`, `standard`, `strict`, `compliance` |
| `reporting.default_issue_min_confidence` | Minimum confidence for auto-issue (`medium` ≈ 0.7) |
| `reporting.default_issue_min_severity` | Minimum severity for auto-issue (default `high`) |
| `false_positive_reduction.enabled` | Master switch for path-based confidence adjustments |
| `false_positive_reduction.suppress_vendor` | Suppress vendor-path findings by default |
| `min_issue_confidence` | Legacy gate; also applied at issue manager |
| `max_issues_per_run` / `reporting.max_issues_per_scan` | Cap issues created per scan (default `25`) |
| `skip_low_severity` | Omit `low` findings from forge issues |
| `repository_exclude_patterns` | Skip noisy repos (see `handlers/repo_filter.go`) |

## Source-type defaults

| Source type | Default action |
|-------------|----------------|
| `source` | `auto_issue` (when severity/confidence pass gates) |
| `test`, `docs`, `example` | `report_only` |
| `generated`, `vendor` | `suppressed_with_reason` |
| `config`, `dependency` | `manual_review` / `auto_issue` for CVEs |

Override per repo via `reporting.source_type_overrides` and `reporting.create_issues_for_*` flags.

## Scanner tools in Docker

The default image sets `INSTALL_EXTERNAL_TOOLS=false`. Dashboard **“Tools not in container”** counts are expected — not failed scans. Missing scanners are recorded as platform warnings with applicability `skipped_tool_unavailable`, not as repository findings.

Build with tools when you need Trivy/Semgrep/Gitleaks in-container:

```bash
INSTALL_EXTERNAL_TOOLS=true ./deploy.sh
```

## Summary rollup issues

Gitea **“Code Review Summary”** issues are only opened when a scan produces **five or more** findings, to avoid an extra ticket for tiny runs.

## After rule changes

Re-run scans so fingerprints and Gitea issues reflect the updated logic:

```bash
./deploy.sh --scan-all
```

Close stale false-positive tickets manually or let fingerprint dedup update them on the next scan.
