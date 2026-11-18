# Repository Detective — production-readiness verification record

**Working-tree update: 2026-11-17.** Base: `repository-detective-main(4).zip` supplied by the owner. This is **not** a production acceptance report, a penetration-test claim, or a release certification. The current tagged beta's earlier passing Gitea tests do not automatically apply to this new code.

## Verified in this constrained workspace

| Check | Outcome | Evidence/limitations |
|---|---|---|
| Repository-scoped declarative rule compiler and line scanner | **PASS** | `GOTOOLCHAIN=local GO111MODULE=off go test ./customrules -count=1`; pure stdlib isolated tests include malformed rules, dangerous unknown fields, size limits, cancellation and non-disclosure of raw source lines |
| UI application JavaScript syntax | **PASS** | `node --check ui/static/app.js` |
| Local UI rule-builder interaction contract | **PASS (mock DOM)** | `node scripts/test-ui-rule-builder.cjs` verifies add/duplicate/unsafe path/multiple rules/reset; not real browser/a11y coverage |
| Go source formatting | **PASS for changed Go files** | `gofmt` on changed Go files and new tests |
| All Go HTML template declarations parse | **PASS (parser only)** | Standalone `html/template` parse of the `ui/templates/*.html` files with stubbed template helper signatures; **not a live handler render or browser test** |
| CSS stylesheet parse | **PASS** | `tinycss2` parse yielded no stylesheet-level errors; not proof of browser layout/contrast |
| Native worker Compose YAML | **PASS (syntax only)** | PyYAML parsed the overlay, verified nonroot UID/GID, allowed job types; Docker Compose execution **not tested** |
| Working patch whitespace | **PASS** | `git diff --check` and clean patch application to the uploaded baseline (verify before release) |
| Browser UI automation | **NOT VERIFIED** | Headless Chromium did not complete in this environment; no passing visual/accessibility evidence |
| Full Go module tests and race suite | **NOT RUN** | Workspace has Go 1.23.2; module specifies Go **1.26.10** and external toolchain fetching is blocked. Added runner/store tests require target toolchain and dependencies |
| Build, Docker/OCI, runtime profiling, Gitea/GitHub end to end | **NOT RUN** | No disposable forge, worker VM, reproducible toolchain, or Docker daemon in this workspace |
| AI remediation token costs, actual patches/PRs, lifecycle closure | **NOT VERIFIED** | The application remains advisory-first; no independently verified unattended dependency-upgrade engine |

## Implemented this pass

- Introduced `.repository-detective.rules.json` per repository, no shell execution, strict JSON schema, resource budgets, deterministic results on core/native worker, and a local-only rule builder in repository settings.
- Pinned native-worker Git checkout to an explicitly supplied commit; rejected credential-bearing URLs, unsafe protocols and branch arguments; bounded worker file enumeration and made excess files, oversized sources, and symlinks explicit coverage errors. Added tests.
- Set native delegation to fail closed on enqueue failures instead of silently moving the scan onto core; made unsupported Gitea Actions mode explicitly unsupported and prevented old Actions jobs from being claimed as native work. Added tests.
- Made unattended AI false-positive classifications advisory only, guarded custom rule IDs from automatic suppression, and added regression tests.
- Reorganized dashboard/repository policy UX, operator trust/evidence path, and safe background scan refresh so errors do not trigger a full page navigation/white flash. Visual browser verification remains open.
- Corrected runner onboarding and backend documentation to stop advertising unsupported Gitea Actions dispatch. CI now includes JavaScript syntax, a Node rule-builder contract test, and targeted custom rule/worker/store tests in addition to the existing full race/vet/staticcheck suites.

## Mandatory pre-public release acceptance

| Gate | Minimum measurable proof | Status |
|---|---|---|
| **Build correctness** | Go 1.26.10: `go mod verify`; `go test -race -count=1 ./...`; `go vet ./...`; staticcheck; govulncheck; reproducible release builds | OPEN |
| **Untrusted repository isolation** | Disposable per-job VM, no Docker socket, enforced egress policy, nonroot/cap-drop, secrets unavailable to code, resource quotas, workspace destruction after job | OPEN |
| **Gitea + GitHub CI backends** | Authenticated dispatch, commit-bound job spec, polling, artifacts, status, timeout, retry/cancel, tamper detection, failure tests | OPEN (NOT IMPLEMENTED) |
| **Scanner correctness** | Fixtures for each language/tool, pinned scanner DB versions, adversarial parser tests, completeness states for disabled/missing/broken scanners | OPEN |
| **Code quality/architecture** | Known-bad and known-good fixtures for loops, complexity, architecture, unmaintained packages, coverage gaps, lint errors, IaC, git history | OPEN |
| **AI changes** | Minimal evidence context, token ceilings measured, per-request cost audit, isolated diff/test/lint/rescan, PR only after approval, rollback and repeat-run idempotence | OPEN |
| **Learning** | Confirmed true/false positive ground truth, A/B false-negative regression suite, no automatic evidence downgrades, independently reviewed changes | OPEN |
| **Supply-chain trust** | Exact commit, hashes/signatures/provenance, SBOM, verifiable scanner coverage, results of tests, explicit unknown/incomplete flags, no blanket safety guarantee | OPEN |
| **Webhooks** | Signed payload/identity tests, SSRF and replay tests, retries/rate-limits, opt-in event routing to human and AI, failure notification | OPEN |
| **UI and accessibility** | Real browser at 320/768/1440px, dark/light/system and reduced-motion, 200–400% zoom, keyboard and screen-reader paths, Core Web Vitals/performance, dark-refresh flash regression | OPEN |
| **Resource utilization** | CPU/memory/disk/latency benchmarks on 5, 500, 5,000 and >limit source-file fixtures, DB/index query plan, cache and cleanup evidence | OPEN |
| **Documentation** | Fresh install on a disposable machine with only the README/Quickstart; feature matrix independently compared to current code; repeatable CI receipts | OPEN |

## Required staging commands

```bash
# In a dedicated CI runner or sandbox with Go 1.26.10:
go mod verify
go test -race -count=1 ./...
go vet ./...
node --check ui/static/app.js

docker compose -f docker-compose.yml -f docker-compose.runner.example.yml config
# Start only on a disposable isolated host, never with sensitive Git/AI credentials.
# Run a known-safe and a deliberately malicious fixture at a fixed SHA.
# Confirm JobSpec commit, scanner completeness, no fallback to core,
# no code execution in core, no broad host/network access, and cleanup.
```

Release decision: **DO NOT LABEL PRODUCTION VERIFIED** until the gates above are evidenced and assigned reviewer sign-off. Partial scanner results, AI advice, and old acceptance receipts must not be conflated with a current clean bill of health.
