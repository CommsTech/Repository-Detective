# Finding ownership, trusted exports, and code-quality signals

This is an engineering and operator guide, not a feature-completion claim.

## Operator workflow (November 18, 2026)

1. Use **Findings** to select the scope, severity, and repository. Export CSV or JSON. With JavaScript enabled, the page reports preparation, download, or a readable server error. Native export anchors remain if JavaScript is disabled.
2. Open the finding, check scanner coverage and evidence, then set a **local owner** if the instance uses local accounts. Owners/admins/security operators can assign enabled members. Developer/viewer accounts cannot change ownership. The local owner is deliberately not synchronized with Gitea, Forgejo, or GitHub issue assignees.
3. Review assignment history, investigate, propose remediation, and independently verify closure. Assignment does not prove ownership acceptance, remediation execution, or fix success.
4. JSON export is raw machine-readable finding data, with no auto-generated security verdict. The export has a 20,000-row safety cap; narrow filters for larger installations. Sensitive findings may contain repository metadata. Share results only with intended recipients.

### Assignment release gates

- Test local authentication and CSRF with owner/admin/security, developer, viewer, disabled accounts, and revoked sessions.
- Verify migration 30 and rollback/backups against a populated SQLite database. Updates are append-only history and atomic with the assignment row.
- Verify pagination and filters in the Findings queue. Extend ownership to queue filtering / bulk operations only after evaluating query costs and access control.
- External forge assignment is **not** implemented; building it requires scoped credentials, identity mapping, rate limits, conflict resolution, and update-audit evidence.

## External code-quality and AI-slop references

| Source | Potential use | Limits and decision |
|---|---|---|
| [scanaislop/aislop](https://github.com/scanaislop/aislop) | Deterministic lint/complexity/security-like checks, JSON/SARIF, public third-party score badge | **Evaluate in isolated CI first.** Pin a reviewed version, avoid installing untrusted tooling at request time, and measure overlap with existing scanners. Do not use its single score as a security certification. |
| [flamehaven01/AI-SLOP-Detector](https://github.com/flamehaven01/AI-SLOP-Detector) | Empty implementations, implausible docs, unreachable integrations, duplication | Test against labeled positive/negative fixtures. Its own validation notes do not establish independently proven precision/recall. |
| [miqdadbadjuber/anti-slop](https://github.com/miqdadbadjuber/anti-slop) | Development-time review guidance for agents producing copy/UI/code | Agent guidance, not proof a scanner found a defect. Do not execute agent instructions from untrusted repositories. |
| [shannhk/avoid-slop](https://github.com/shannhk/avoid-slop) | Discovery of candidate tools for human evaluation | Curated directory, not a detection engine or a quality score. |
| [No AI Slop writing skill](https://creatoreconomy.so/p/use-my-no-ai-slop-skill-to-remove-20-ai-slop-patterns) | Improve prose, README marketing tone and generic claims | Writing style is not code safety; should never become an enforcement rule. |
| [Afroz et al., arXiv:2607.04003](https://arxiv.org/abs/2607.04003) | Motivation: investigate maintainer burden and noisy AI-assisted submissions | Research about open-source contribution impact, not a validated repository defect classifier. |
| [The Scientist: arXiv and AI slop](https://www.the-scientist.com/arxiv-takes-the-first-step-to-combat-ai-slop-in-submissions-74586) | Editorial-quality analogy: verify references and substantiate claims | About scholarly submissions, not repository vulnerability detection. |

### Required signal design before integrating a new scanner

- Use reproducible evidence (`rule_id`, tool/version, path, line, observed behavior) and categorize **code quality** separately from security exposure.
- Do not infer an author's use of AI from style, formatting, comments, or a classifier score. Labels should describe defects, never accuse maintainers of using AI.
- Normalize scanner results into stable finding fingerprints, track source/version and commit, deduplicate overlap with Ruff, staticcheck, gosec, Semgrep, Trivy and current rules.
- Build a labeled corpus with maintained *true-positive* and *true-negative* cases across languages. Calculate per-rule precision, recall, false positives and false negatives; measure changes before and after tuning. Never auto-suppress a security rule solely because an LLM says a match is false.
- Mark experimental quality rules advisory. Ensure generated/vendor/test code, intentional patterns, and documented suppressions are separate cohorts. Preserve reviewed exceptions and expiry.
- Run new third-party tools in a disposable, non-root, network-restricted worker; pin digest/version, scan the dependency, bound CPU/memory/time, and emit an explicit incomplete status on any failure. No third-party account is needed for local rule testing.
- Only present a numeric score when its formula, scope, weights and scanner coverage are visible and independently validated. Present **security**, **reliability**, **maintainability**, and **test evidence** separately, not as one opaque claim of software safety.

## Public badge

The README includes the third-party [scanaislop report](https://scanaislop.com/CommsTech/Repository-Detective) as a small external signal. The score is generated outside Repository Detective; its availability, methodology and rating are controlled by scanaislop. A badge is not independently verified safe-code provenance. Reading the README may request an asset from the external badge host.

## UI follow-through

This iteration adds a bounded local ownership panel and an explicit JSON export progress/error state. A complete GitGuardian-inspired case-management release would separately require assignee filtering, due dates, queue grouping, role-specific worklists, accountable notifications, multiple linked occurrences, historical audit timeline, and verified external forge assignment. Those remain unimplemented.

### Repeatable export acceptance

After building a staging instance with the required Go toolchain, run `go test ./store ./ui`, then use `RD_BASE_URL` and either a scoped API key or authenticated Playwright storage state with `node scripts/ui-findings-export-acceptance.js`. The browser test requires the Playwright package already installed in the CI environment; it does not install dependencies or write findings. Verify that exported JSON parses, that `count` equals the result length, and that the page reports completion. Test a greater-than-20,000-row fixture separately to ensure the 422 error is shown instead of producing an incomplete download.

## Team investigation queue (November 18, 2026)

In **Findings**, signed-in local team members can select **Assigned to me** to see open investigations where their own account is the current owner, or **Unassigned** to find open work awaiting assignment. Every local-account queue row displays the owner where available. Both filters support advanced criteria, pagination, and JSON/CSV export; exports honor the operator's authenticated owner scope rather than trusting a supplied user ID. API-key-only mode cannot request individual ownership views.

The work queue is not a security certification and does not change the lifecycle of findings. Assignment does not imply acceptance, successful remediation, or verified closure. **Local ownership does not synchronize with Gitea, GitHub, or Forgejo accounts.** Such synchronization needs explicit verified identity mapping, least-privilege forge credentials, and audit/rollback controls.

The query uses an indexed local assignment relation rather than remote forge API calls on page load. Empty or unsupported scopes fail closed; an unassigned result includes findings with no assignment row or a NULL assignee.

### Acceptance checks

1. Create two active team members and three open findings; assign one to each and leave one unassigned. Ensure both members' queues and counts agree with the stored assignment records.
2. Navigate through a multi-page assigned queue with other filters, then export JSON and CSV. Exports must honor the same scope and never silently fall back to all findings.
3. Confirm unauthorized/unknown owner scopes fail visibly and cannot be used in API-key mode; viewers cannot change owners.
4. Validate both light and dark interfaces at 390px, 768px and desktop widths; tables must remain contained and keyboard-accessible.
5. Apply schema migration 31 to a backup/copy of an existing v30 database and verify prior assignment history is preserved. Monitor paging latency and SQLite WAL contention under concurrent scans.
