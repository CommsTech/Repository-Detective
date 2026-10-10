# RC active-present 21 classification

**Scan ID:** `e42b3e175e313904`  
**Repository:** commstech/Repository-Detective (id=1)

All 21 findings classified. Actionable = medium/high/critical per reconciliation (`actionable_active_open`).

| ID | Fingerprint (short) | Title | Source/Rule | Sev | Conf | Path:Line | Actionable | Classification | Planned action | File issue? |
|----|----------------------|-------|-------------|-----|------|-----------|------------|----------------|----------------|-------------|
| 37361 | ab24918f… | Internal infra ref | static/REL-INTERNAL-INFRA-REF | medium | 0.80 | containers/discover.go:169 | yes | scanner_self_match | FP rule: registry classifier | no |
| 37362 | c85cdefd… | Internal infra ref | static/REL-INTERNAL-INFRA-REF | medium | 0.80 | containers/types.go:247 | yes | scanner_self_match | FP rule: private registry detect | no |
| 37365 | 806eefe3… | Very large function | maintainability/HEALTH-LARGE-FUNC | medium | 0.93 | analyzers/static.go:201 | yes | scanner_self_match | Skip analyzers/static in health | no |
| 37366 | a232c49b… | Deep nesting | maintainability/HEALTH-DEEP-NEST | medium | 0.89 | analyzers/static.go:342 | yes | scanner_self_match | Skip analyzers/static in health | no |
| 70 | 4a1fef61… | Deprecated API | tech_debt/HEALTH-DEPRECATED | low | 0.91 | ai/config.go:30 | no | needs_human_review | Track mapstructure alias | no |
| 12084 | 9f059338… | Deprecated API | tech_debt/HEALTH-DEPRECATED | low | 0.91 | internal/config/envcompat/envcompat.go:70 | no | needs_human_review | Legacy env compat by design | no |
| 37360 | 2c49ca41… | HTTP client per call | optimization/OPT-HTTP-CLIENT-PER-CALL | low | 0.80 | health/reliability.go:187 | no | scanner_self_match | Already FP in health/ path — verify | no |
| 37377 | e54092fa… | Ignored error | reliability/HEALTH-IGNORED-ERROR | low | 0.77 | main.go:1284 | no | needs_human_review | Review intentional discard | no |
| 37379 | b39f39e9… | Deprecated API | tech_debt/HEALTH-DEPRECATED | low | 0.91 | main.go:901 | no | needs_human_review | viper legacy key | no |
| 37380 | 084dd0c3… | Ignored error | reliability/HEALTH-IGNORED-ERROR | low | 0.77 | openclaw/service.go:115 | no | needs_human_review | Advisory-only path | no |
| 37381 | a0990d86… | Ignored error | reliability/HEALTH-IGNORED-ERROR | low | 0.77 | openclaw/service.go:94 | no | needs_human_review | Advisory-only path | no |
| 6855 | 9d7e4605… | Tech debt marker | tech_debt/HEALTH-TECH-PHRASE | info | 0.55 | patcher/git.go:21 | no | repo_scoped_calibration_candidate | Accept info | no |
| 36697 | 63057156… | Large file | maintainability/HEALTH-LARGE-FILE | info | 0.55 | analyzers/engine.go:1 | no | docs_only_low_priority | Accept info for orchestrator | no |
| 36701 | 2497dd34… | Read-all footgun | performance/HEALTH-READ-ALL | info | 0.55 | api/runner_handler.go:145 | no | needs_human_review | Bounded read in follow-up | no |
| 36702 | 6de62a8e… | Read-all footgun | performance/HEALTH-READ-ALL | info | 0.55 | api/runner_handler.go:310 | no | needs_human_review | Bounded read in follow-up | no |
| 36707 | 9fc1159e… | Many params | maintainability/HEALTH-MANY-PARAMS | info | 0.55 | gitea/reporter.go:45 | no | docs_only_low_priority | Accept info | no |
| 36709 | 83d89e19… | Many params | maintainability/HEALTH-MANY-PARAMS | info | 0.55 | issuelink/backfill.go:102 | no | docs_only_low_priority | Accept info | no |
| 36710 | 46a813ef… | Many params | maintainability/HEALTH-MANY-PARAMS | info | 0.55 | issuelink/backfill.go:28 | no | docs_only_low_priority | Accept info | no |
| 36711 | 100100c9… | Large file | maintainability/HEALTH-LARGE-FILE | info | 0.55 | main.go:1 | no | docs_only_low_priority | Accept info for entrypoint | no |
| 36722 | ed366034… | Many params | maintainability/HEALTH-MANY-PARAMS | info | 0.55 | preinstall/checks.go:216 | no | docs_only_low_priority | Accept info | no |
| 37378 | 2781001c… | Deep nesting | maintainability/HEALTH-DEEP-NEST | info | 0.55 | main.go:2118 | no | docs_only_low_priority | Accept info | no |

## Summary by bucket

| Bucket | Count |
|--------|-------|
| scanner_self_match | 5 (4 medium + 1 low OPT) |
| docs_only_low_priority / info accepted | 10 |
| needs_human_review | 6 |
| real_code_fix | 0 |

## Why no Gitea issues filed

Issue filing enabled but **no mapped open issues** — findings below confidence/severity gates for auto-file, or advisory/info severity excluded by reporting policy. Historical DB has 2457 open finding rows without forge links (expected when gates block filing).

## Acceptance

- All 21 accounted for
- All 4 actionable explained (scanner self-match — fix in Phase 2)
- No blanket suppression
- No high/critical hidden
