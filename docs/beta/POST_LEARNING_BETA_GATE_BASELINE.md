# Post-learning beta gate baseline

Generated: 2026-06-02

## Latest commit

`f44dde0` — docs(beta): update readiness with learning engine

## Product repo status

| Gate | Status |
|------|--------|
| Open issues | 1 (#48 operator task) |
| Active-present findings | 0 |
| Non-product issue filing | Disabled |
| All-repo scan | Not started |
| Report-only dry-run | Available via API |
| LLM sanity gate | Disabled by default (`llm_sanity_gate_enabled: false`) |
| Backlog-control | Active |

## Learning engine status

| Component | Status |
|-----------|--------|
| Data model (schema v20) | Shipped in `f44dde0` |
| Lifecycle learning events | Wired |
| Per-repo calibration recommendations | Shipped |
| Safe accept/reject API | Shipped — global accept blocked |
| Structural dedup | Shipped |
| Reachability heuristics | Shipped |
| Learning health UI | `/ui/learning` + dashboard |

## Deployment note (post-sprint)

Production `data/` is owned by container user. Use `docker run ... go run ./cmd/rd-migrate` for migrations and operator review via documented reports when host user lacks write access.

## Remaining blockers (start of sprint)

1. `make beta-release` — root-owned `dist/repository-detective-beta`
2. Cursor Repository-Detective benchmark fixture run
3. staticcheck CI confirmation
4. Operator calibration accept/reject on homelab dry-run repos

## Prior sprint beta readiness recommendation

**Private beta ready** for homelab/internal testers with continuous learning observability.

Public beta pending: benchmark fixture, staticcheck CI, operator calibration review, beta packaging fix.
