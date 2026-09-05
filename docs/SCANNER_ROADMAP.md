# Deterministic-First Static Analysis Roadmap (Planning)

Repository Detective — **Inspect. Analyze. Improve.**

> **Status:** Planning only. Do not implement until Phase 12 (runner delegation) is stable and Phase 12B (branding) is scheduled separately.

## Principle

Use the [Wikipedia static analysis tool catalog](https://en.wikipedia.org/wiki/List_of_tools_for_static_code_analysis) as **discovery inspiration**, not an integration checklist. Repository Detective integrates tools that are:

- **Deterministic** — reproducible output, no LLM required
- **Actionable** — file/line/rule_id suitable for issues and gates
- **Operable** — reasonable install story for core and runner hosts
- **Policy-controlled** — per-repo and global toggles

## Assessment layers

```text
Layer 1: Deterministic scanners     ← expand here first
Layer 2: Language-native analyzers
Layer 3: Graph / reachability context  (Phase 11 — done)
Layer 4: Test / build evidence        ← runners enable later
Layer 5: AI                           ← explanation, triage, remediation planning only
```

### Current foundation (shipped)

| Capability | Status |
|------------|--------|
| Trivy | Shipped |
| Grype | Shipped |
| Gitleaks | Shipped (optional) |
| Semgrep | Shipped (optional) |
| Linters (golangci-lint, ruff, shellcheck) | Shipped |
| govulncheck, gosec, staticcheck (Go trio) | **Shipped (Phase 13A)** |
| hadolint, checkov (IaC/container) | **Shipped (Phase 13B)** |
| Built-in static regex rules | Shipped |
| Health checks (Phase 10) | Shipped |
| Repository graph (Phase 11) | Shipped |
| Pre-install static checks | Shipped |
| Runner delegation (Phase 12) | Shipped |

---

## AI boundaries (policy — enforce in code over time)

| AI may | AI may not |
|--------|------------|
| Summarize complex findings | Invent vulnerabilities without evidence |
| Explain impact and blast radius | File **verified** issues without deterministic support |
| Correlate weak signals (advisory) | Auto-fix without tests |
| Propose remediation plans | Claim code is AI-written |
| Suggest tests | Override scanner proof |
| Classify architecture (advisory label) | Create commit status failures on AI-only findings |

### Issue creation policy (target)

```text
No deterministic evidence → AI may create "review suggestion" (advisory), not verified issue.
Deterministic evidence + AI explanation → verified issue allowed.
Deterministic evidence + passing tests → remediation PR allowed (future phase).
```

Implement as `source` + `confidence` gates in issue policy (extend Phase 8 controls).

---

## Recommended next 5 integrations

After Phase 12 stabilizes:

| Priority | Tool | Why |
|----------|------|-----|
| **1** | **govulncheck** | High-value Go vuln signal with reachability; complements Trivy/Grype |
| **2** | **gosec** | Go security rules; fills SAST gap for Go repos |
| **3** | **staticcheck** (first-class) | Already run in CI/dev; normalize into scanner_results, issues, gates |
| **4** | **hadolint** | Dockerfile quality/security; fits pre-install + container repos |
| **5** | **checkov** | IaC/config security (Terraform, K8s, CloudFormation) |

These fit owned-repo scans, pre-install audits, and runner execution without LLM.

---

## Scanner selection matrix

| Tool | Lang/ecosystem | Runner? | Dep install? | Noise | Output | Default | FP risk | Priority |
|------|----------------|---------|--------------|-------|--------|---------|---------|----------|
| **govulncheck** | Go | Optional | `go mod` | Low–med | JSON/text | On for Go | Low | **P1** |
| **gosec** | Go | Optional | No | Med | JSON | On for Go | Med | **P1** |
| **staticcheck** | Go | Optional | No | Med | JSON | On for Go | Low–med | **P1** |
| **hadolint** | Docker | Core/runner | No | Low | JSON | On | Low | **P2** |
| **checkov** | IaC | Core/runner | No | Med–high | JSON | Off | Med | **P2** |
| npm/pnpm/yarn audit | JS/TS | Runner | Yes | Med | JSON | Off | Med | P3 |
| ESLint | JS/TS | Runner | Yes | Med–high | JSON | Off | Med | P3 |
| tsc --noEmit | TS | Runner | Yes | Med | text | Off | Low | P3 |
| pip-audit | Python | Runner | Yes | Low–med | JSON | Off | Low | P3 |
| bandit | Python | Runner | Optional | Med | JSON | Off | Med | P3 |
| mypy | Python | Runner | Yes | Med | text | Off | Med | P4 |
| SpotBugs | Java | Runner | Yes | Med | XML/JSON | Off | Med | P4 |
| PMD / Checkstyle | Java | Runner | Yes | High | XML | Off | High | P4 |
| OWASP Dependency-Check | Java | Runner | Yes | Med | JSON/XML | Off | Med | P4 |
| cppcheck | C/C++ | Runner | No | Med | XML | Off | Med | P4 |
| clang-tidy | C/C++ | Runner | Yes | Med | YAML/text | Off | Med | P4 |
| tfsec / successor | Terraform | Core | No | Med | JSON | Off | Med | P4 |
| kube-linter | K8s | Core | No | Med | JSON | Off | Med | P4 |

**Legend:** P1 = next sprint after stabilization; P2 = following batch; P3+ = ecosystem expansion.

---

## Ecosystem detail

### Go (integrate first — product is Go)

| Tool | Value | Runner? | Dep install? | Normalization |
|------|-------|---------|--------------|---------------|
| govulncheck | OSV + import graph reachability | Yes if no module cache | `go mod download` | Map to `dependency` / `security`; rule_id = GOVULN-* |
| gosec | Security antipatterns | Yes | No | Map to `security`; SARIF/JSON |
| staticcheck | Bugs, performance, style | Yes | No | Map to `code_quality` / `reliability`; register as scanner |
| go test ingestion | Test failures as signals | Runner | Yes | `reliability` finding; no issue unless policy allows |
| go test -race | Data races | Runner only | Yes | High FP cost; off by default; label advisory |

### JavaScript / TypeScript

| Tool | Value | Runner? | Dep install? | Notes |
|------|-------|---------|--------------|-------|
| npm/pnpm/yarn audit | Supply chain | Runner | lockfile + node_modules | Prefer lockfile-only mode first |
| ESLint | Lint/security plugins | Runner | Yes | High volume; severity gate required |
| tsc --noEmit | Type errors | Runner | Yes | Quality signal, not security |
| lockfile checks | Tampering | Core | No | Deterministic heuristics |

### Python

| Tool | Value | Notes |
|------|-------|-------|
| pip-audit | CVEs | Complements Trivy |
| bandit | Security AST | Overlap with Semgrep — policy pick one default |
| ruff | Lint | **Already present** via linters |
| mypy | Types | Optional; noisy |

### Java / Kotlin

| Tool | Value | Notes |
|------|-------|-------|
| SpotBugs | Bug patterns | Needs bytecode build on runner |
| PMD / Checkstyle | Style/rules | Very noisy; off by default |
| OWASP Dependency-Check | CVEs | Heavy; runner-only |
| Maven/Gradle audit | CVEs | Via plugin or lockfile parsers |

### C/C++

| Tool | Value | Notes |
|------|-------|-------|
| cppcheck | Static analysis | No build required for basic pass |
| clang-tidy | LLVM checks | Needs compile_commands.json |
| Compiler warnings | Build output | Runner + build stage later |
| Sanitizers (ASan/UBSan) | Runtime bugs | Runner + test phase; Phase 14+ |

### IaC / containers

| Tool | Value | Default |
|------|-------|---------|
| hadolint | Dockerfile | On after P1 Go batch |
| checkov | Multi-IaC | Off; high value for pre-install |
| tfsec / OpenTofu scanners | Terraform | Off |
| kube-linter | K8s manifests | Off |

---

## Normalization contract (all new scanners)

Every scanner must produce:

```text
scanner_name
status (clean | found | failed | binary_missing | timed_out)
rule_id
severity
file, line (when available)
title, detail
source (scanner id for deterministic registry)
fingerprint inputs (rule + file + line block)
```

Register in `scanners.RegisterDeterministicSource(name)`.

Persist via existing `scanner_results` + finding pipeline — no parallel issue path.

---

## Runner vs core execution

| Needs | Run on |
|-------|--------|
| `go mod`, `npm ci`, compile, tests | **Runner** (Phase 12+) |
| Single binary, filesystem only (gosec, staticcheck, hadolint, checkov) | **Core or runner** |
| Large monorepo + install | **Runner** |

Runners must **not** gain forge tokens or issue APIs — unchanged from [RUNNERS.md](RUNNERS.md).

---

## Policy defaults (proposed)

| Scanner | Global default | Per-repo override field |
|---------|----------------|-------------------------|
| govulncheck | `true` when Go detected | `enable_govulncheck` |
| gosec | `true` when Go detected | `enable_gosec` |
| staticcheck | `true` when Go detected | `enable_staticcheck` |
| hadolint | `true` | `enable_hadolint` |
| checkov | `false` | `enable_checkov` |

Auto-enable when language detected only if binary present — same pattern as Trivy today.

Pre-install: inherit global scanner config (like graph in 11B).

---

## Implementation phases (scanner track)

| Phase | Scope |
|-------|--------|
| **13A** | govulncheck + gosec + staticcheck first-class (Go) |
| **13B** | hadolint + checkov (IaC/containers) |
| **13C** | JS audit + ESLint (runner-only) |
| **13D** | Python pip-audit + bandit |
| **14+** | Java/C++ build-assisted; test/race ingestion |

**Gate:** Phase 12 runner stable; binary packaging documented for core and runner images.

---

## Tests required (per new scanner)

- [ ] Parser golden files (sample CLI output → normalized findings)
- [ ] `binary_missing` graceful degradation
- [ ] Timeout behavior
- [ ] Fingerprint stability
- [ ] Per-repo disable
- [ ] Runner JobResult includes scanner_results
- [ ] Issue gates respect severity/confidence
- [ ] Pre-install audit includes scanner when enabled
- [ ] No LLM invoked for deterministic-only findings

---

## Docs to update (when implementing)

| Doc | Change |
|-----|--------|
| [SCANNERS.md](SCANNERS.md) | Tool list, config, enable flags |
| [POLICY.md](POLICY.md) | Per-repo scanner toggles |
| [RUNNERS.md](RUNNERS.md) | Required binaries on runner image |
| [PREINSTALL_AUDIT.md](PREINSTALL_AUDIT.md) | IaC/container scanners |
| [DATABASE.md](DATABASE.md) | Only if new settings columns |
| [config/config.yaml](config.yaml) | Defaults |
| [.env.example](../.env.example) | New env vars |
| Docker image docs | Binary install list |

---

## Commercial / heavy tools (explicitly later)

CodeQL, Coverity, Infer, Frama-C, and similar tools are **out of scope** until:

- Open deterministic adapter exists, or
- Operator provides structured SARIF upload path (future "SARIF ingest" phase)

Do not block P1–P2 open-source integrations on commercial adapters.

---

## Related documents

- [SCANNERS.md](SCANNERS.md) — current scanner behavior
- [HEALTH_CHECKS.md](HEALTH_CHECKS.md) — deterministic health layer
- [CODE_GRAPH.md](CODE_GRAPH.md) — graph layer
- [RUNNERS.md](RUNNERS.md) — runner execution model
- [PHASE_12B_BRANDING_PLAN.md](PHASE_12B_BRANDING_PLAN.md) — branding migration (separate track)
- [CAH_PIPELINE.md](CAH_PIPELINE.md) — AI pipeline boundaries
