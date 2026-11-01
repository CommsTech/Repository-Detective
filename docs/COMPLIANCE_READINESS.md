# Compliance readiness (evidence index)

Repository Detective supports **compliance readiness** activities. It does **not** replace legal review, DPIA, BAA, or formal certification.

## Evidence map

| Topic | Evidence | Limitation |
|-------|----------|------------|
| Privacy-aware handling | [PRIVACY_AND_DATA_PROTECTION.md](PRIVACY_AND_DATA_PROTECTION.md) | Heuristic redaction only |
| Secret masking in issues | `issues.SanitizeSecretEvidence`, template tests | Legacy issues unchanged |
| Secret masking in DB evidence | `store/recorder.go` redactSnippet (closeout) | Historical rows may exist |
| Log redaction | `internal/security/redact.go`, `scanners/log_redact.go` | Not all scanners migrated |
| Subprocess env isolation | `internal/security/env.go`, tests | Admin must verify deployment |
| AI opt-out | `enable_llm_auditors: false` in config examples | Default depends on deployment |
| Accessibility | [ACCESSIBILITY.md](ACCESSIBILITY.md), skip link, focus styles | No third-party audit |
| Scanner transparency | [SCANNER_HEALTH.md](SCANNER_HEALTH.md), `/api/v1/status` | Runtime probe ≠ historical scans |
| Audit trail | SQLite findings lifecycle, scan history | No immutable audit log |
| Admin responsibilities | [ADMIN_HARDENING.md](ADMIN_HARDENING.md), [DATA_RETENTION.md](DATA_RETENTION.md) | Manual enforcement |

## HIPAA

**Not HIPAA compliant.** Do not process PHI without organizational controls, BAA with subprocessors (Gitea, AI vendors), and legal sign-off.

## GDPR

**Not GDPR compliant as a product.** Supports administrator tasks: documentation of data flows, deletion via DB/issue cleanup, minimizing external transfers (disable LLM).

## Section 508 / WCAG

**WCAG-aligned improvements** documented; **not** VPAT or ACR available. Procuring organizations should run independent testing.

## FedRAMP / SOC2

No control matrix included. Use this index plus your own control framework.

## Self-attestation template

```text
We deploy Repository Detective with:
- LLM auditors: disabled
- API/UI: TLS + API key
- Retention: <policy>
- Gitea: private instance
Evidence reviewed: docs/COMPLIANCE_READINESS.md dated <date>
Legal review: pending | complete
```
