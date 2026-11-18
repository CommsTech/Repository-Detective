# Doc Detective review (Gitea #47)

Review of [doc-detective](https://github.com/doc-detective/doc-detective) for Repository Detective — **June 2026**.

## Summary

Doc Detective is a **documentation testing** framework (runbooks, how-to verification). It complements security scanners; it does **not** replace SAST, secret detection, or dependency scanning.

## Useful ideas adopted / documented

| Doc Detective concept | Repository Detective equivalent |
|----------------------|--------------------------------|
| Executable runbook steps | [DOGFOODING.md](DOGFOODING.md), `scripts/dogfood-self-scan.sh`, `scripts/verify-all.sh` |
| Reproducible doc tests | CI workflow + [RELEASE_READINESS.md](RELEASE_READINESS.md) |
| Spec → command verification | Operator UI smoke via `/health`, `/api/v1/status` |

## Not integrated (by design)

- No Doc Detective runner in container image (extra Python/Node stack)
- No automatic UI browser tests (would need Playwright/cypress — out of scope)

## Recommendation

- Keep **manual + script** verification for operator docs
- Optional future: add `docs/test/` Doc Detective specs for onboarding wizard — track as P3 if needed

**Closeout:** Review complete; no Doc Detective dependency added to production image.
