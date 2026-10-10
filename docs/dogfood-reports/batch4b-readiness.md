# Batch 4b readiness

Generated: 2026-06-06  
Post-rescan: `db2d7061eaac8eb0`

## Current counts

| Bucket | Count |
|--------|------:|
| Open Gitea issues | 57 |
| Real active backlog | 11 |
| Resolved absent (open) | 14 |
| Duplicates | 0 |
| Needs human review | 2 |
| Out of scope (summary) | 30 |

## All-repo scan status

**Blocked.** Open count must drop further and remaining 11 active findings should be reduced before fleet scan.

## Next 10–12 code fixes (Batch 4b)

| Priority | Issue area | Rule | Notes |
|----------|------------|------|-------|
| 1 | scanners/archive_extract.go | G304 | Validated zip path; verify or harden OpenFile |
| 2 | ui/ui_helpers.go | G203 | Template HTML context review |
| 3 | store/findings_batch_sqlite.go | G201 | Document safe parameterized IN clause |
| 4 | config.env.template | CKV_SECRET_6 | Template-only placeholders |
| 5–8 | REL-INTERNAL-INFRA-REF | static | Defer / suppress with documented homelab scope |
| 9–12 | DL3018 | hadolint | Defer unless pinning policy changes |

## Next closure pass

Run `scripts/close-remaining-resolved-absent.py` against scan `db2d7061eaac8eb0` for 14 resolved-absent candidates.

## Docker rebuild

`apk-retry` merged in `28d5303`. Full `all-in-one` build still subject to builder-stage Go module download flakiness (IPv6); use `vendor/` or `--network=host` for CI/homelab builds until vendor is committed.
