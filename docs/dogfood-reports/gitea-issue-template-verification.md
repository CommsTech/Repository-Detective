# Gitea issue template verification

**Date:** 2026-06-12  
**Repo:** `commstech/Repository-Detective`  
**Commit:** `311e97c`  
**Gitea version:** 1.26.2

## Summary

| Check | Result |
|-------|--------|
| Templates in repository | **yes** — 15 form templates + `config.yml` |
| `config.yml` present | **yes** — `blank_issues_enabled: false` |
| Templates visible (authenticated API) | **yes** — `GET /api/v1/repos/commstech/Repository-Detective/issue_templates` returns 15 templates |
| Logged-in web UI picker | **not captured** — API token does not establish web session (redirect to login); server-side template API confirms picker data |
| Blank issues allowed | **no** (per `config.yml`; Gitea 1.26 supports this key) |
| Sensitive default text | **none observed** — security markdown in FP/beta templates only |
| Beta / FP scan ID + fingerprint fields | **yes** — required fields in API-parsed forms |
| Test issue created | **no** (product repo) |
| Screenshots | **not captured** — no headless browser / web session in validation environment |

## Authenticated API verification (2026-06-12)

```http
GET /api/v1/repos/commstech/Repository-Detective/issue_templates
Authorization: token <redacted>
→ 200, 15 templates
```

Templates returned by Gitea (display names):

| Template |
|----------|
| Accessibility |
| Beta feedback |
| Bug report |
| Compliance / privacy |
| Container scan issue |
| Documentation gap |
| Feature request |
| Missed detection |
| Operator task |
| Pre-install audit issue |
| SBOM issue |
| Scanner false positive |
| Scanner or parser bug |
| Security finding triage |
| UI or UX issue |

## `config.yml`

```yaml
blank_issues_enabled: false
contact_links:
  - name: Private beta scope
  - name: Feedback templates (docs)
  - name: Security — do not paste secrets
```

## Field verification (beta + false positive)

**Beta feedback** (required): `version`, `scan_id`, `repo` (+ provider, report-only, category fields).

**Scanner false positive** (required): `version`, `finding_id`, `fingerprint`, `scan_id`, `repo`, `scanner`, `rule`, `why_fp`, `expected`.

Security note present in both templates (markdown block).

## Web UI limitation

Unauthenticated and API-token requests to `/commstech/Repository-Detective/issues/new` redirect to login. The Gitea **issue_templates** API is the authoritative server-side representation of the logged-in template picker on Gitea 1.26.x.

**Operator follow-up:** capture one browser screenshot of the template picker when logged in (optional polish).

## Related

- `.gitea/ISSUE_TEMPLATE/`
- UI links: finding detail (false positive), scan detail (beta feedback)
- `docs/triage/ISSUE_TRIAGE_POLICY.md`
