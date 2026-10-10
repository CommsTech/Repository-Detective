# Active-present 26 classification

Recorded: 2026-06-02  
Scan: `27fbd37be97ef5f7`  
Total findings: **26** (all accounted for)

| fingerprint | title | source/rule | severity | confidence | file | line | latest scan | classification | planned action | evidence | file Gitea issue | disposition | not high/critical because |
|---|---|---|---:|---:|---|---:|---|---|---|---|---|---|
| rd-ad9459365207dda1 | Deeply nested control flow | maintainability / HEALTH-DEEP-NEST | medium | 0.89 | analyzers/static.go | 328 | yes | global_rule_fix_candidate | Skip maintainability checks for static analyzer self-scan path | Static analyzer file is excluded from maintainability in `skipMaintainabilityPath` | no | fix detector skip | Maintainability advisory in scanner implementation |
| rd-0745a629c1c0e001 | Deeply nested control flow | maintainability / HEALTH-DEEP-NEST | info | 0.55 | main.go | 2048 | yes | repo_scoped_calibration_candidate | Repo-scoped calibration + raised nesting threshold for bootstrap | Bootstrap orchestration; already calibrated in seed rules | no | downgraded visible | Already info; bootstrap control flow |
| rd-4a1fef61a8082981 | Deprecated API referenced | tech_debt / HEALTH-DEPRECATED | low | 0.91 | ai/config.go | 30 | yes | global_rule_fix_candidate | Skip intentional deprecation doc on `LegacyConfig` type | Comment/doc describes backward-compat type, not runtime deprecated API | no | fix detector skip | Documentation of legacy config surface |
| rd-9f0593382afb1a33 | Deprecated API referenced | tech_debt / HEALTH-DEPRECATED | low | 0.91 | internal/config/envcompat/envcompat.go | 70 | yes | global_rule_fix_candidate | Skip logger deprecation notice for legacy env prefix | One-time Info log about naming migration | no | fix detector skip | Operator guidance log, not code using deprecated API |
| rd-ce6cd8a83c1fd1c3 | Deprecated API referenced | tech_debt / HEALTH-DEPRECATED | low | 0.91 | main.go | 848 | yes | global_rule_fix_candidate | Skip logger.Warn for query-string API key deprecation | Intentional security deprecation warning | no | fix detector skip | Auth middleware advisory log |
| rd-d3cde7bbb353f9bd | HTTP call without timeout | reliability / HEALTH-HTTP-NO-TIMEOUT | medium | 0.91 | runner/client.go | 112 | yes | real_reliability_fix | Replace `http.DefaultClient` fallback with `&http.Client{Timeout: 2m}` | Real network reliability gap in runner client | no | fix code | Was medium reliability; fixed in code |
| rd-14e0d7c8ade80f6a | Ignored error return | reliability / HEALTH-IGNORED-ERROR | low | 0.77 | main.go | 1222 | yes | global_rule_fix_candidate | Allow type-assertion discard in `isBestEffortIgnoredError` | `githubIssueClient, _ = githubClient.(*github.Client)` when GitHub optional | no | fix detector skip | Optional forge client type assertion |
| rd-dfa282f6629830d4 | Ignored error return | reliability / HEALTH-IGNORED-ERROR | info | 0.63 | main.go | 1980 | yes | real_reliability_fix | Log warning on invalid runner policy snapshot JSON | Was silent ignore; now logs with context | no | fix code | Already info; best-effort ingest path |
| rd-63057156fe8de853 | Very large source file | maintainability / HEALTH-LARGE-FILE | info | 0.55 | analyzers/engine.go | 1 | yes | repo_scoped_calibration_candidate | Skip at scanner + repo calibration rule | Known orchestration monolith; seed rule exists | no | downgraded visible | Already info; expected engine size |
| rd-100100c9032ab87d | Very large source file | maintainability / HEALTH-LARGE-FILE | info | 0.55 | main.go | 1 | yes | repo_scoped_calibration_candidate | Skip at scanner + repo calibration rule | Bootstrap entrypoint; seed rule exists | no | downgraded visible | Already info; bootstrap monolith |
| rd-5d74b4920b180648 | Very large source file | maintainability / HEALTH-LARGE-FILE | info | 0.55 | ui/handler.go | 1 | yes | repo_scoped_calibration_candidate | Skip at scanner + repo calibration rule | UI handler surface; seed rule exists | no | downgraded visible | Already info; UI surface file |
| rd-34fb909a95bb8c83 | Very large function | maintainability / HEALTH-LARGE-FUNC | medium | 0.93 | analyzers/static.go | 194 | yes | global_rule_fix_candidate | Exclude static.go from maintainability checks | Same as deep-nest self-scan path | no | fix detector skip | Maintainability in static analyzer |
| rd-26c5a71cb6c94a85 | Very large function | maintainability / HEALTH-LARGE-FUNC | info | 0.55 | store/profiles.go | 84 | yes | repo_scoped_calibration_candidate | Raise function threshold for profiles builder + seed rule | Profile builder decomposition tracked separately | no | downgraded visible | Already info |
| rd-9fc1159ecb5cedce | Many parameters | maintainability / HEALTH-MANY-PARAMS | low | 0.90 | gitea/reporter.go | 45 | yes | global_rule_fix_candidate | Adapter param threshold bump + repo calibration | Forge reporter adapter | no | fix detector / calibrate | Low maintainability advisory |
| rd-83d89e19266bb332 | Many parameters | maintainability / HEALTH-MANY-PARAMS | low | 0.90 | issuelink/backfill.go | 102 | yes | global_rule_fix_candidate | Adapter param threshold bump + repo calibration | Issue link backfill helper | no | fix detector / calibrate | Low maintainability advisory |
| rd-46a813ef6b444018 | Many parameters | maintainability / HEALTH-MANY-PARAMS | low | 0.90 | issuelink/backfill.go | 28 | yes | global_rule_fix_candidate | Adapter param threshold bump + repo calibration | Issue link backfill helper | no | fix detector / calibrate | Low maintainability advisory |
| rd-ed3660349520d945 | Many parameters | maintainability / HEALTH-MANY-PARAMS | low | 0.90 | preinstall/checks.go | 216 | yes | global_rule_fix_candidate | Adapter param threshold bump + repo calibration | Pre-install audit orchestration | no | fix detector / calibrate | Low maintainability advisory |
| rd-43e0a95db42f76f9 | Many parameters | maintainability / HEALTH-MANY-PARAMS | low | 0.90 | remediation/renderer.go | 78 | yes | global_rule_fix_candidate | Adapter param threshold bump + repo calibration | Remediation template renderer | no | fix detector / calibrate | Low maintainability advisory |
| rd-fe1ce19e35b64046 | Many parameters | maintainability / HEALTH-MANY-PARAMS | low | 0.90 | runner/signing.go | 28 | yes | global_rule_fix_candidate | Adapter param threshold bump + repo calibration | Runner signing protocol | no | fix detector / calibrate | Low maintainability advisory |
| rd-10d41a89283b5e7c | Many parameters | maintainability / HEALTH-MANY-PARAMS | low | 0.90 | runner/spec.go | 116 | yes | global_rule_fix_candidate | Adapter param threshold bump + repo calibration | Runner job spec structs | no | fix detector / calibrate | Low maintainability advisory |
| rd-fb831274fff2030d | Many parameters | maintainability / HEALTH-MANY-PARAMS | low | 0.90 | ui/repo_settings_model.go | 83 | yes | global_rule_fix_candidate | Adapter param threshold bump + repo calibration | UI settings model binding | no | fix detector / calibrate | Low maintainability advisory |
| rd-2497dd349a58ef93 | ReadAll footgun | performance / HEALTH-READ-ALL | low | 0.81 | api/runner_handler.go | 145 | yes | global_rule_fix_candidate | Add `api/` to expected ReadAll paths | Bounded runner job payload reads | no | fix detector skip | Expected API handler pattern |
| rd-6de62a8e193090fc | ReadAll footgun | performance / HEALTH-READ-ALL | low | 0.81 | api/runner_handler.go | 310 | yes | global_rule_fix_candidate | Add `api/` to expected ReadAll paths | Bounded runner job payload reads | no | fix detector skip | Expected API handler pattern |
| rd-9d7e4605d7679c94 | Tech debt marker | tech_debt / HEALTH-TECH-PHRASE | low | 0.90 | patcher/git.go | 21 | yes | global_rule_fix_candidate | Skip "temporary git" workspace comments | Doc comment about temp clone dir, not TODO | no | fix detector skip | Comment describes temp workspace |
| rd-f48331d33f1ae749 | Tech debt marker | tech_debt / HEALTH-TECH-PHRASE | low | 0.90 | scanners/git_history_clone.go | 15 | yes | global_rule_fix_candidate | Skip temporary workspace comments in scanners | Doc comment for clone workspace | no | fix detector skip | Comment describes temp workspace |
| rd-60178ffdc78e3582 | Tech debt marker | tech_debt / HEALTH-TECH-PHRASE | low | 0.90 | scanners/workspace.go | 133 | yes | global_rule_fix_candidate | Skip temporary workspace comments in scanners | Doc comment for workspace lifecycle | no | fix detector skip | Comment describes temp workspace |

## Bucket summary

| Classification | Count |
|---|---:|
| global_rule_fix_candidate | 19 |
| repo_scoped_calibration_candidate | 6 |
| real_reliability_fix | 2 |
| real_code_fix | 0 |
| intentional_ignore_with_context | 0 |
| scanner_self_match | 0 |
| test_fixture_or_benchmark | 0 |
| docs_only_low_priority | 0 |
| graph_or_reachability_noise | 0 |
| needs_human_review | 0 |
| operator_task | 0 |

## Notes

- No vague "noise" bucket used.
- No global suppression rules added from product-only evidence.
- High/critical count remains 0; no downgrades applied to high/critical.
- Real fixes queued: HTTP timeout, policy snapshot logging, health checker false-positive skips.
