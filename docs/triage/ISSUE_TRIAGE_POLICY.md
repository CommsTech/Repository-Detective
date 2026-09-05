# Issue triage policy

Repository Detective — Inspect. Analyze. Improve.

This policy applies to **product repo** issues (`commstech/Repository-Detective`) filed via Gitea templates during the private beta.

## Triage SLA (beta feedback)

| Priority | Target first response | Target resolution path |
|----------|----------------------|-------------------------|
| Blocks beta testing | 1 business day | Hotfix or documented workaround |
| High/critical product finding | 2 business days | Triage + owner assigned |
| False positive / calibration | 3 business days | Accept/reject + rule draft |
| Docs / UI polish | 5 business days | Backlog or quick fix |
| Feature requests | 5 business days | Accept/defer with rationale |

## Required evidence

Every actionable issue must include:

- Repository Detective version/commit (or `/api/v1/about`)
- Scan ID (when scanner-related)
- Repository under test (`owner/name`)
- Finding fingerprint and rule/source (when applicable)
- Expected vs actual behavior
- Redacted reproduction steps — **no raw secrets**

Issues missing scan ID + fingerprint for scanner feedback receive `status/needs-repro`.

## False positives — acceptance criteria

Accept as false positive when:

1. Tester shows the match is test/fixture/docs/generated/vendor noise.
2. Rule fires on intentional pattern with documented rationale.
3. Confidence is low and evidence does not support exploitability.
4. Repo-scoped calibration is sufficient (no global suppression needed).

Reject when:

- Secret material is plausibly real (treat as true positive until rotated).
- Dependency/CVE is reachable in deployment context.
- Pattern matches known vulnerable API usage.

**Action:** create repo-scoped calibration draft; global suppressions require operator review.

## Missed detections — acceptance criteria

Accept when:

1. Repro steps show a real vulnerability or policy violation in scope.
2. Scanner was enabled for that scan profile.
3. File/path was in scan scope (not ignored).

**Action:** file scanner/parser bug or add detection rule; do not silently hide similar future findings.

## Scanner/parser bugs

Fix scanner logic when:

- Parser crash or wrong file attribution
- Systematic misclassification across repos
- Evidence redaction failure (treat as **critical**)

## Keep visible but not issue-worthy

Some findings stay on the dashboard/report but do not file forge issues:

- Report-only scans (beta default)
- Below confidence/severity gate
- Backlog-control blocked (low severity during burn-down)
- Suppressed/calibrated fingerprints

## When to request more information

Apply `status/needs-repro` when:

- No scan ID or fingerprint for scanner feedback
- Cannot reproduce without secrets (ask for redacted substitute)
- Ambiguous expected behavior

## Security note

Never attach `.env`, tokens, private keys, registry credentials, PHI/PII, or customer data. Use redacted snippets only.
