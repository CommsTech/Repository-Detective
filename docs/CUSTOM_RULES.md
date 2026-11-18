# Custom repository rules (unreleased working tree)

Repository Detective supports version-controlled, **declarative** rules for each repository. This is for maintainers who know their own codebase's conventions; it does not replace SAST, dependency analysis, dependency-lock verification, secret detection, test coverage, or human review.

**For new users:** Open **Repositories → your repository → Policy settings → Build a custom rule**. Fill out the simple form and download the configuration file. Place the file at the top of the repository, review it in your normal code-review process, and commit it. The builder never updates your Gitea repository directly.

**For experienced users:** Create `.repository-detective.rules.json` at the repository root:

```json
{
  "version": 1,
  "rules": [
    {
      "id": "CUSTOM_DEBUG_LOG",
      "title": "Debug logging left in Go code",
      "description": "Review logging of potentially sensitive data before release",
      "category": "quality",
      "severity": "medium",
      "path_glob": "src/*.go",
      "pattern": "fmt\\.Print(ln|f)?\\("
    },
    {
      "id": "CUSTOM_UNSAFE_SUBPROCESS",
      "title": "Review shell execution",
      "category": "security",
      "severity": "high",
      "pattern": "exec\\.Command\\("
    }
  ]
}
```

The examples are intentionally broad and may report legitimate code. A matched pattern is a **candidate finding**, never proof of a vulnerability or exploitability. Review and tune before setting enforcement policy.

## Fields and limits

| Field | Requirement |
|---|---|
| `version` | Integer `1` |
| `rules` | 1–32 unique entries |
| `id` | Uppercase identifier, 3–48 characters, begins with a letter; letters, digits, `_` and `-` only |
| `title` | Required, up to 160 characters |
| `description` | Optional, up to 400 characters |
| `category` | `security`, `quality`, `performance`, `maintainability`, `architecture`, `misconfiguration`, `supply_chain`, `testing`, `tech_debt` |
| `severity` | `critical`, `high`, `medium`, `low`, `info` |
| `path_glob` | Optional, relative path, up to 200 characters; Go `path.Match` syntax, so `src/*.go` is **not** recursive. Empty means all scanned source files |
| `pattern` | Required Go RE2 regular expression, up to 320 characters; matched **within one line**, not across multiple lines |

The file may be at most **64 KiB**. Rule evaluation has a **30-second timeout**, a **2 MiB per-matching-file** limit, and a **2,000-finding** hard cap. Reaching a hard limit produces an explicit incomplete-scan error; it does not count as clean. These checks depend on the files actually provided by the configured scan backend; an incomplete upstream checkout or disabled scanner is still incomplete coverage.

Unknown fields, invalid globs/regexes, duplicate IDs, and unsupported categories or severities are rejected. No `command`, shell, script, remote include, arbitrary plugin, or executable action is accepted. This protects the scanner from a large class of repository-controlled execution attacks.

## Troubleshooting

- **No custom-rule findings:** Confirm the file exists **in the scanned commit**, the path matches the exact source path, and the regex matches within one source line. Inspect scan details for scanner failures or skipped files.
- **Config rejected:** Check that the JSON is valid (backslashes in regex must be double-escaped in JSON), uppercase IDs are unique, and all configured fields appear in the table above.
- **Scanning stopped at a budget:** Reduce rule breadth or inspect the oversized file; never interpret a failed/incomplete run as zero findings.
- **Many false alerts:** Narrow the path/pattern; do not rely on AI-driven self-suppression of this rule.

## Boundaries

Custom rules are intended for **deterministic string/line matching**. AST-aware architectural constraints, dataflow, recursion, complex performance regressions, dependency issues, and code execution require dedicated analyzers, tests, and reviewed workflows. User-defined rules do not automatically change policy gates, AI approval levels, forge permissions, or notification credentials.

See [Operator experience](OPERATOR_EXPERIENCE.md), [Trust and isolation](SECURITY_SANDBOX_AND_TRUST.md), and [Release verification](release/RELEASE_VALIDATION_2026-11-17.md).
