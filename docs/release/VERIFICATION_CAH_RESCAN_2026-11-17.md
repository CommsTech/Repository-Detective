# CAH rescan receipt — 2026-11-17

**Status:** informational. This records a self-hosted rescan after the TLS and
dependency hardening commit; it is not a release acceptance certificate.

## Inputs

- Commit scanned: `02c181b7` (adds TLS `MinVersion` enforcement, `golang.org/x/net`
  v0.62.0, operator triage queue and findings workflow).
- Profile: `deep`, report-only dry run (no issues filed, no repository changes).
- Deployment: self-hosted single-operator instance, local session authentication.

## Result

The scan completed analysis and recorded canonical findings. Open findings in
the operator queue after the run:

| Severity | Count |
|---|---|
| high | 69 |
| medium | 87 |
| low | 10 |
| info | 34 |

Counts reflect the first 200 queued rows; use the queue's total-match counter
and exports for a complete figure.

## What changed relative to the previous scan

- Two `G304` (file inclusion via variable) findings moved to `resolved_verified`,
  meaning the rescan no longer reproduced them.
- Still open: `G402` (TLS `InsecureSkipVerify` opt-in site), a semgrep
  `missing-ssl-minversion` finding, `TRIVY-CVE-2026-78661`, and three remaining
  `G304` findings.

## Remaining known debt

- `errcheck` (`LINT-GO-errcheck`): 134 open findings.
- `gosec` `G104` (unhandled errors): 15 open findings.
- These dominate the high/medium totals. They are mechanical correctness fixes,
  not proven vulnerabilities, but they are not acceptable as shipped defaults
  for a security product and require a package-by-package remediation pass.

## What this proves and does not prove

Proves: the scanners ran against the stated commit, findings were normalized and
deduplicated, and two previously reported findings no longer reproduce.

Does not prove: that the remaining findings are exploitable or benign; that the
remediation sandbox is safe; that a disposable-forge end-to-end acceptance run
passed; or that any release tag inherits these results.
