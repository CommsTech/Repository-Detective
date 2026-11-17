# Calibration review guide (beta)

## When to review

- Pending recommendations appear on `/ui/learning`
- False-positive rate exceeds operator tolerance
- New scanner or rule added to profile
- After non-product dry-run scans

## Review checklist

1. **Scope** — repo-scoped vs global (global requires multi-repo evidence)
2. **Severity protected** — never auto-downgrade high/critical
3. **Evidence count** — minimum findings before accept
4. **Expiry** — all repo rules should have review date
5. **Finding visibility** — calibrated findings remain in report (severity/confidence may change)

## Actions

| Action | Effect |
|---|---|
| `informational` | Downgrade low/medium to info; finding stays visible |
| `downgrade_confidence` | Lower confidence cap |
| `report_only` | Same as informational for filing purposes |

## API workflow

```bash
curl -H "X-Repository-Detective-API-Key: $KEY" \
  http://localhost:8081/api/v1/calibration/recommendations?status=pending

curl -X POST -H "X-Repository-Detective-API-Key: $KEY" \
  http://localhost:8081/api/v1/calibration/recommendations/1/accept
```

## Do not

- Globally suppress from product-repo evidence alone
- Hide findings from operators
- Enable `calibration_auto_apply` in beta without review
