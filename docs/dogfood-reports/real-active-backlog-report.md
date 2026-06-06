# Real active backlog — commstech/Bugbot

Generated: 2026-06-06 22:17 UTC

Scan: **`2b335070099f8936`**

## Summary

| Metric | Count |
|--------|------:|
| Gitea open (exported) | 283 |
| active_present_in_latest_scan | 48 |
| resolved_absent_from_latest_scan | 136 |
| resolved_verified_open_by_policy | 0 |
| duplicate_existing_fingerprint | 68 |
| out_of_scope_for_current_batch | 29 |
| needs_human_review | 2 |
| HEALTH-IGNORED-ERROR (active) | 9 |

## Why open count grows

- Scanner coverage expanded (more rules/tools per scan).

- Evidence closure keeps issues open when `close_issues=false`.

- Duplicates are labeled, not deleted.

- New scanner availability creates variance findings.


## Active code-fix queue (top 30 by issue #)

| # | Rule | Source | Title |
|--:|------|--------|-------|
| #53 | REL-INTERNAL-INFRA-REF | static | [MEDIUM] Possible internal infrastructure reference |
| #66 | REL-INTERNAL-INFRA-REF | static | [MEDIUM] Possible internal infrastructure reference |
| #83 | HEALTH-IGNORED-ERROR | reliability (AI auditor) | [MEDIUM] Potential reliability issue: ignored error ret |
| #143 | REL-INTERNAL-INFRA-REF | static | [MEDIUM] Possible internal infrastructure reference |
| #144 | REL-INTERNAL-INFRA-REF | static | [MEDIUM] Possible internal infrastructure reference |
| #145 | REL-INTERNAL-INFRA-REF | static | [MEDIUM] Possible internal infrastructure reference |
| #205 | CKV_SECRET_6 | checkov (AI auditor) | [MEDIUM] Base64 High Entropy String |
| #206 | CKV_SECRET_6 | checkov (AI auditor) | [MEDIUM] Base64 High Entropy String |
| #232 | DL3018 | hadolint (AI auditor) | [MEDIUM] Pin versions in apk add. Instead of `apk add < |
| #259 | DL3018 | hadolint (AI auditor) | [MEDIUM] Pin versions in apk add. Instead of `apk add < |
| #260 | DL3018 | hadolint (AI auditor) | [MEDIUM] Pin versions in apk add. Instead of `apk add < |
| #261 | DL3018 | hadolint (AI auditor) | [MEDIUM] Pin versions in apk add. Instead of `apk add < |
| #262 | DL3018 | hadolint (AI auditor) | [MEDIUM] Pin versions in apk add. Instead of `apk add < |
| #263 | HEALTH-IGNORED-ERROR | reliability (AI auditor) | [MEDIUM] Potential reliability issue: ignored error ret |
| #264 | HEALTH-IGNORED-ERROR | reliability (AI auditor) | [MEDIUM] Potential reliability issue: ignored error ret |
| #267 | HEALTH-IGNORED-ERROR | reliability (AI auditor) | [MEDIUM] Potential reliability issue: ignored error ret |
| #276 | HEALTH-IGNORED-ERROR | reliability (AI auditor) | [MEDIUM] Potential reliability issue: ignored error ret |
| #278 | HEALTH-IGNORED-ERROR | reliability (AI auditor) | [MEDIUM] Potential reliability issue: ignored error ret |
| #279 | HEALTH-IGNORED-ERROR | reliability (AI auditor) | [MEDIUM] Potential reliability issue: ignored error ret |
| #280 | REL-INTERNAL-INFRA-REF | static | [MEDIUM] Possible internal infrastructure reference |
| #292 | HEALTH-IGNORED-ERROR | reliability (AI auditor) | [MEDIUM] Potential reliability issue: ignored error ret |
| #296 | REL-INTERNAL-INFRA-REF | static | [MEDIUM] Possible internal infrastructure reference |
| #301 | G304 | gosec | [MEDIUM] Potential file inclusion via variable |
| #302 | G304 | gosec | [MEDIUM] Potential file inclusion via variable |
| #303 | G204 | gosec | [MEDIUM] Subprocess launched with a potential tainted i |
| #304 | G306 | gosec | [MEDIUM] Expect WriteFile permissions to be 0600 or les |
| #305 | G304 | gosec | [MEDIUM] Potential file inclusion via variable |
| #306 | G306 | gosec | [MEDIUM] Expect WriteFile permissions to be 0600 or les |
| #307 | G304 | gosec | [MEDIUM] Potential file inclusion via variable |
| #308 | G304 | gosec | [MEDIUM] Potential file inclusion via variable |
