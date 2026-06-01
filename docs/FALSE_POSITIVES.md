# Reducing false positives

Repository Detective combines **static heuristics**, **external scanners**, **repo health checks**, and optional **LLM auditors**. Heuristic rules are fast but can mis-classify safe patterns.

## Common false positives (fixed in static analyzer)

| Pattern | Example | Why it is usually safe |
|---------|---------|------------------------|
| Hardcoded secret in `deploy.sh` | `local api_key="${BUGBOT_API_KEY:-}"` | Reads from environment, not a literal secret |
| `data-api-key` in HTML templates | `data-api-key="{{.APIKey}}"` | UI passes the operator API key to JS — not embedded credentials |
| SQL concat in Go store layer | `query := base + \` WHERE status = ?\`` | Appends a **constant** fragment with `?` placeholders |
| Orphan file warnings | `ui/templates/*.html`, `scripts/*.sh` | Not part of the Go import graph by design |

## Tuning policy

| Setting | Effect |
|---------|--------|
| `min_issue_confidence` | Raise (e.g. `0.7`) to drop low-confidence findings before Gitea filing |
| `max_issues_per_run` | Cap issues created per scan (default `50`) |
| `skip_low_severity` | Omit `low` findings from forge issues |
| `repository_exclude_patterns` | Skip noisy repos (see `handlers/repo_filter.go`) |

## Scanner tools in Docker

The default image sets `INSTALL_EXTERNAL_TOOLS=false`. Dashboard **“Tools not in container”** counts are expected — not failed scans. Build with tools when you need Trivy/Semgrep/Gitleaks in-container:

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
