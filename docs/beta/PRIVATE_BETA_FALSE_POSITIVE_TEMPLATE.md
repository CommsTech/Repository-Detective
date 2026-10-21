# Private beta false positive template

Use when a finding appears incorrect. Helps calibration without global suppression.

**Preferred:** open Gitea template [`scanner_false_positive`](https://git.commsnet.org/commstech/repository-detective/issues/new?template=scanner_false_positive) on the product repo, or use **Report false positive** on the finding detail page in UI.

---

## Required fields

| Field | Value |
|-------|-------|
| **Finding ID** | from UI URL `/ui/findings/:id` |
| **Fingerprint** | from finding detail if shown |
| **Repository** | `owner/name` |
| **Scan ID** | |
| **Rule / source** | e.g. `static/REL-INTERNAL-INFRA-REF` |
| **Severity / confidence** | |
| **File path and line** | |

## Why this is a false positive

Explain in plain language:


## Context

- [ ] Test code / `_test.go` / fixture
- [ ] Documentation / markdown only
- [ ] Generated / vendor code
- [ ] Example / benchmark
- [ ] Intentional pattern (explain)
- [ ] Scanner self-match on product code
- [ ] Other: ___

## Expected behavior

What should Repository Detective do instead?

- [ ] Suppress with repo-scoped calibration (operator only)
- [ ] Fix scanner rule globally (product team)
- [ ] Accept as informational only
- [ ] No change — tester misunderstanding (explain)

## Evidence

- Code snippet (minimal, no secrets):
- Link to file in forge (if public):
- Screenshot optional

## Operator review

- Valid FP: [ ] Yes [ ] No [ ] Needs human review
- Action taken:
- Calibration rule ID (if any):
