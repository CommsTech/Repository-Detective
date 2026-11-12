# Repository Detective vs Cursor Bugbot

Evidence-based comparison — **no unverified superiority claims**.

## Public facts (Cursor Bugbot)

- Reviews pull requests for bugs, security issues, and code quality ([Cursor Bugbot docs](https://cursor.com/docs/bugbot)).
- Supports repository/project rules and team rules.
- Autofix spawns cloud agents to test proposed fixes ([Autofix blog](https://cursor.com/blog/bugbot-autofix)).
- Resolution rate improved via experiments and measurement ([Building Bugbot](https://cursor.com/blog/building-bugbot)).
- Primary forge integration: **GitHub-oriented**.

## Repository Detective positioning

Gitea-first, self-hosted, evidence-backed **full-repository lifecycle** inspection:

- Scheduled and manual full-repo scans (not PR-only)
- Deterministic scanner matrix + graph/reachability context
- Report-only dry runs without issue filing
- Issue idempotency, backlog-control, evidence closure
- SBOM generation + grype checking (beta)
- Per-repo calibration (not global overfit)
- **Continuous learning engine** — auditable events, repo-scoped recommendations, structural dedup (beta)
- Project groups for multi-repo applications (beta)

## Comparison matrix

| Dimension | Cursor Bugbot | Repository Detective |
|-----------|---------------|---------------------|
| Platform | GitHub (primary) | **Gitea-first**, self-hosted |
| PR review | **Core product** | Supported via PR scan path |
| Full repo scheduled scan | Limited / not primary | **Core** |
| Issue lifecycle in forge | GitHub issues/PRs | Gitea issues + idempotent filing |
| Evidence closure | Autofix + agent tests | Scan-verified closure + merge checks |
| Self-hosted | Cloud service | **Yes** |
| SBOM gen + check | Not advertised as core | **Beta** (syft/gomod + grype) |
| Scanner transparency | Proprietary stack | Configurable deterministic scanners |
| Per-repo calibration | Rules + team rules | **Suppressions + homelab profiles + learning events** |
| Continuous learning | Resolution experiments (cloud) | **Deterministic lifecycle learning, optional LLM gate off** |
| Structural dedup | Not advertised | **Pattern hash grouping (beta)** |
| Reachability priority | Not advertised | **Graph/path heuristics (beta)** |
| Project grouping | Project rules | **Beta project groups** |
| Report-only mode | N/A | **Dry-run without filing** |
| Auto-remediation | **Autofix agents** | Planner + gated PR (default off) |
| Public beta readiness | GA cloud product | **This sprint** |

## Where Cursor Bugbot is likely stronger (today)

- GitHub-native UX and Autofix agent loop
- Large-scale PR throughput on GitHub.com
- Mature resolution-rate optimization on cloud infra

## Where Repository Detective targets strength (needs benchmark proof)

- Gitea/homelab/private deployment
- Full-repo + SBOM + graph context in one operator UI
- Auditable closure without cloud agent dependency
- Safe dry-run calibration before enabling issue filing

## Benchmark plan

Results recorded in [CURSOR_BUGBOT_BENCHMARK_RESULTS.md](CURSOR_BUGBOT_BENCHMARK_RESULTS.md) (2026-06-02).

| Metric | Method |
|--------|--------|
| Precision / recall | Same repo, same injected bug set, compare findings |
| False positive rate | Operator disposition on known-clean paths |
| Actionability | Developer score 1–5 on top 20 findings |
| Time to useful report | Wall clock from trigger to report |
| Issue duplication rate | Repeat scans same commit |
| Verified closure | Fix commit + rescan behavior |

**Test repo:** controlled fixture repo (not production homelab).  
**PR path:** single PR with known vulnerabilities.  
**Output:** update this doc with measured numbers — do not claim “better” until filled.

## Recommendation

Publish capability differences now; run benchmark fixture before any competitive marketing.
