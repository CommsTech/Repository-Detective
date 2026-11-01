# Repository Detective benchmark fixture

Controlled workspace for comparing Repository Detective against documented Cursor Bugbot-style capabilities.
Not scanned in production homelab by default.

## Injected cases

| File | Expected signal |
|------|-----------------|
| `secret_hardcoded.go` | True positive — fake API key pattern |
| `sql_concat.go` | True positive — SQL string concat |
| `requirements.txt` | Dependency/SBOM candidate — pinned requests/urllib3 for lockfile shape |
| `mock_secret_test.go` | Likely false positive — mock token in test |
| `vendor/minified.js` | False positive candidate — minified vendor |
| `orphan_module.go` | Graph/dead-code candidate |
| `safe_internal_url.go` | False positive candidate — homelab URL |
| `env_fallback.go` | False positive candidate — env fallback template |
| `dup_pattern_a.go`, `dup_pattern_b.go` | Structural duplicate pattern |

## Usage

```bash
go test ./benchmark/... -count=1 -v
```

Report-only API scans of this fixture are not registered in Gitea; use the Go test harness for deterministic benchmark evidence.
