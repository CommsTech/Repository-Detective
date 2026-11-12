# Repository Detective Doctor (RD-014)

Answers: **Is Repository Detective configured and operational, and if not, exactly what is wrong?**

Shared with onboarding Verify.

## Interfaces

| Surface | How |
|---------|-----|
| CLI | `repository-detective doctor` · `--json` · `--bundle` · `--repo owner/name` |
| API | `GET /api/v1/doctor` · `GET /api/v1/doctor/bundle` |
| UI | `/ui/doctor` |

## Overall results

| Result | Meaning |
|--------|---------|
| HEALTHY | All requirements for configured mode satisfied |
| DEGRADED | Core can work; optional warnings / NOT_PROVEN items |
| NOT_READY | Required component ERROR |

Component states: `PASS` `WARNING` `ERROR` `NOT_CONFIGURED` `NOT_APPLICABLE` `NOT_PROVEN`.

These are **readiness** states — not repository security.

## Support bundle

`doctor --bundle` / `/api/v1/doctor/bundle` includes build metadata, checks, scanner versions, sanitized config.

Excludes: API keys, tokens, webhook/session secrets, source, diffs, secret evidence, full env.

## Examples

Human:

```text
Repository Detective Doctor
Overall: DEGRADED
...
== AI ==
  [PASS] AI Analysis: DISABLED — Status: VALID
```

JSON check:

```json
{
  "id": "scanner.gitleaks.available",
  "category": "scanner",
  "state": "PASS",
  "summary": "gitleaks available",
  "proof": "RUNTIME_CHECK"
}
```
