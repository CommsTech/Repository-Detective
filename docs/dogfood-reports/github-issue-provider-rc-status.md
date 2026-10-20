# GitHub issue provider RC status

**Date:** 2026-06-11  
**Decision:** Option B — demote to beta/unverified

## Rationale

- GitHub forge adapter exists (`issues.GitHubForge`, `Manager.CreateIssuesFromAnalysis`)
- Unit test `TestGitHubForgeCreateIssue` **PASS**
- Live startup check: **401 Bad credentials** with configured token
- No owned test repo live filing performed this pass

## Product status

| Surface | Label |
|---------|-------|
| `docs/ISSUE_PROVIDERS.md` | `implemented, not release-proven` |
| `docs/beta/ISSUE_FILING_POLICY.md` | beta-unproven |
| Configure / health | not_configured or unverified |
| Marketing docs | **must not claim GitHub issue filing works** |

## GitLab

**not_implemented** — unchanged.

## Default behavior

GitHub issue filing should remain **disabled** until operator configures valid token and completes controlled proof.

## Acceptance

Provider matrix is **honest** for private beta.
