# SC2034: unused variable i in sync-gitea-to-github.sh

> **Automated fix available** — Repository Detective can open a safe remediation PR for this finding.

## Summary

shellcheck `SC2034` detected in `sync-gitea-to-github.sh`.

**Severity:** Low  
**Detection confidence:** 95%  
**First seen:** Sep 8, 2026  
**Last seen:** Sep 8, 2026

## Why this matters

ShellCheck reports an unused variable. Confirm the variable is truly unused (or export it if it is consumed externally). If Repository Detective marks this as auto-fixable, prefer the remediation PR path instead of hand-editing.

## Location

`scripts/sync-gitea-to-github.sh:94`

Commit: `4aa38593deadbeef`

## Evidence

```bash
tmp="$(mktemp -d)"
sha="$(git rev-parse --short HEAD)"
i=1
echo "building snapshot"
```

## Recommended action

Apply the lint-guided fix (or let Repository Detective open a remediation PR), keep behavior unchanged, and re-scan.

## Verification

Run the relevant project tests and re-scan with the same scanner.

Repository Detective will consider this resolved only when the fingerprint no longer appears, or when it has been explicitly triaged as false positive or accepted risk.

<details>
<summary>Repository Detective details</summary>

- Scanner: shellcheck
- Rule: `SC2034`
- Fingerprint: `rd-dogfood-sc2034`
- Scan: `dogfood-scan-1`
- Finding ID: `1202`
- Fix complexity: small
- Safe for auto PR: true

</details>
