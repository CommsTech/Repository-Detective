# Public tester readiness — evidence matrix

Release status: **STAGING / NOT YET CERTIFIED**. This checklist applies to the attached source changes and must be repeated against the exact pushed commit and image digest. A passing unit test alone does not authorize public release.

| Gate | Required evidence | Status |
| --- | --- | --- |
| Filtered CSV/JSON parity | Exact matching IDs/counts for all queue filters; no silent truncation; injection checks | Source changes and tests written; Go CI pending |
| Human authentication | No default password; secure first-owner bootstrap; session rotation; auth failures and logout verified | Existing feature; staging verification pending |
| Read-only RBAC | Viewer and unknown roles blocked from all UI mutation paths; authorized reads unaffected | New middleware/unit tests; Go CI pending |
| Repository isolation | Users cannot access other repos via list, export, detail, graph, API, or forged POST | **Not proven** — blocker for multi-tenant/public hosting |
| Authentik / Gitea login | Complete OIDC/OAuth flow and negative tests | **Not implemented** |
| CI Gitea | Tests/race/vet/image build, scanner availability, clean tag release, rollback | Test gate added to publisher; live run pending |
| CI GitHub | Independent source-test workflow and test-gated mirror | Added source-test job/gate; live run pending |
| Performance | p50/p95 first render, page navigation, DB query timings, scan duration, CPU/RAM/disk on representative repo sizes | Baseline pending; no unsupported speed claim |
| Scanner accuracy | Versioned known-positive/known-negative corpus; precision, recall, false negative regression; coverage failures remain visible | Pending |
| Learning calibration | Reviewer-confirmed labels, scanner-version provenance, known-positive canaries, rollback, refusal to learn security findings away | Pending independent validation |
| AI remediation | Validate 4B/medium/large models with identical task harness: bounded evidence retrieval, schema checks, tool budgets, test/scan re-evaluation, reviewable PR | Real worker/forge E2E pending |
| UI actions and accessibility | Authenticated Playwright crawl in light/dark, phone/tablet/desktop; every button, failure state, keyboard and screen reader flows | Existing scripts; real staging execution pending |
| Distribution | Fresh install with only `.env.example`, no seeded personal repos, no real credentials/default hostnames; setup guide links valid | Partial source sweep; fresh install pending |
| Release integrity | Verified source commit, tested image digest, scanner versions, signed/SBOM/provenance documentation | Pending |

## Repeatable commands

```bash
# Run in Go 1.26.11 CI (not in this offline environment)
go mod tidy && git diff --exit-code -- go.mod go.sum
go test -race -count=1 ./...
go vet ./...
node --check ui/static/app.js
node scripts/test-ui-rule-builder.cjs
docker build --target core -t repository-detective:acceptance .

# Use a disposable staging instance with test repositories and valid credentials.
python3 scripts/bench-http-latency.py --base http://127.0.0.1:8081 --path /health/ready --iterations 30
```

The HTTP benchmark is read-only and measures response latency, not client paint time or scanner throughput. Run authenticated UI/browser QA separately. Do not collect private keys or real secrets in screenshots and export fixtures.

## Follow-up engineering backlog (not delivered here)

- Per-repository authorization and external SSO, then owner-aware forge issue assignment.
- Scan pipeline profiling by stage and input size; cached vulnerability DB provenance, bounded concurrent jobs, queue latency and resource ceilings.
- Model-agnostic AI remediation with staged tasks (classification → file selection → minimal patch → isolated tests → independent verification), token budgets, and strict evidence contracts.
- Calibration holdout corpus and independently verified true-positive/false-negative baselines, with explicit rollback for each accepted tuning update.
- Human-centered UI walkthroughs: first login, first scan, triage, custom rule, export, assignment, patch review, audit report, and recovery from failed scanner/runner.
