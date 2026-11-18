# Release evidence gate — implementation and independent acceptance

**Source baseline:** `repository-detective-main(20261118-193222).zip` (18 November 2026).
**Status:** implemented in source; **not verified on production Gitea, GitHub, or Docker registries**. Do not treat this document as a release certification.

## Purpose

The existing source-push CI jobs create a fresh, commit-pinned executive self-scan report. Before this change, tag-driven Gitea Docker publication, Gitea binary releases, and GitHub GHCR mirroring could publish without validating that report.

The three publication paths now run their **own release-time self-scan** against the checked-out commit. This is deliberate: reusing an unrelated CI artifact by name risks accepting a stale scan. The release gate validates both the scan and its artifact checksum **before any image push or release creation**. A self-scan failure produces an incomplete artifact for diagnostics, but the publication gate fails closed.

The validated report must have:

- full SHA-1 Git commit identity equal to the checked-out `GITHUB_SHA`;
- a matching authenticated repository and expected forge (`gitea` or `github`);
- completed scan status and identified scan execution;
- nonempty required-scanner policy, positive execution evidence for **every** required scanner, and no failed/duplicate/invalid scanner outcomes;
- coherent finding counts and executive decision, with no evidence-collection blockers;
- SHA-256 checksums covering the two exact files: `executive-selfscan.md` and `executive-selfscan.json`.

The accepted outcomes are `REVIEW_FINDINGS` and `REVIEW_COVERAGE`. **Neither proves the release is safe.** `REVIEW_FINDINGS` remains a visible human risk-review task; it is *not* an automatic zero-findings security gate. An optional future policy can refuse to ship with independently confirmed critical findings, but needs an explicit, audited exception workflow.

The Gitea binary release now includes three source-commit-specific downloadable files:

- `executive-selfscan.md`
- `executive-selfscan.json`
- `executive-selfscan.SHA256SUMS`

The general `SHA256SUMS` also includes the reports and binaries. The Gitea release upload must fail on missing assets or HTTP errors rather than ignoring them. The Docker publish and GHCR mirror workflows upload their evidence as workflow artifacts. This is **not** independent artifact signing, build reproducibility, a container SBOM, or proof of sandbox isolation.

## Operator configuration

For **each** forge CI context, set:

| Setting | Requirement |
|---|---|
| `RD_SELFSCAN_URL` | Authenticated Repository Detective scan service reachable from that runner; use HTTPS or a securely connected loopback endpoint |
| `RD_SELFSCAN_API_KEY` | Scoped read/scan API credential, kept only as an Actions secret |
| `RD_SELFSCAN_OWNER` | Owner of the actual source repository on that forge |
| `RD_SELFSCAN_REPO` | Exact repository name |
| `RD_SELFSCAN_REQUIRED` | Explicit scanner identifiers, e.g. `gitleaks,govulncheck,staticcheck`, actually installed and enabled |

The repository identity in the returned scan must **also** match the publishing workflow's own `GITHUB_REPOSITORY` value. Never silence the gate by disabling scanners, spoofing a healthy JSON file, or reducing `RD_SELFSCAN_REQUIRED` without security review. The GitHub public mirror requires the corresponding repository connected in the scan service as a GitHub forge; the canonical Gitea scan alone is not a substitute for a GitHub-identity assertion.

## Local checks (executed in this source pass)

```bash
python3 -m unittest discover scripts/tests -v
python3 -m py_compile scripts/ci-release-evidence-gate.py scripts/ci-selfscan-executive.py
# Validates sample report generation and fails on wrong commit or checksum.
```

## Independent-agent acceptance: tests that require real infrastructure

Use a disposable Gitea/GitHub staging repository, staged credentials, and a nonproduction tag. Return **PASS**, **FAIL**, or **BLOCKED** for every item with sanitized logs, exact SHA, workflow URL, and relevant artifact checksum. Never upload raw secrets, tokens, or private repository content to public issues.

- [ ] **G1 — Full Go and UI test suite:** On Go **1.26.11**, run `go mod verify`, `go vet ./...`, `go test -race -count=1 ./...`, staticcheck, the Python script suite, browser UI acceptance, and a clean Docker build. Include exit codes.
- [ ] **G2 — Gitea Docker release happy path:** Push a disposable `v*` tag. Confirm `verify` runs self-scan at the tag's SHA; evidence gate passes; subsequent `publish` pushes exactly the tag expected. Record image digest, SHA, and workflow URL.
- [ ] **G3 — Gitea binary release happy path:** Confirm self-scan and gate execute before publication. Download the three executive assets plus `SHA256SUMS`; verify both checksums and that the scan ID, repo, forge and commit match the release.
- [ ] **G4 — GitHub mirror happy path:** Use a matching public mirror SHA and verified GitHub forge identity; ensure mirror does not run before its independent evidence gate succeeds. Check OCI digest equivalence when mirroring rather than rebuilding.
- [ ] **G5 — Missing secret:** Remove `RD_SELFSCAN_API_KEY` from a disposable run. Require `EVIDENCE_INCOMPLETE`, artifact upload attempt, and **no new image/tag/release**.
- [ ] **G6 — Wrong commit/repository/forge:** Simulate mismatched revision, repository mapping, or forge with a staging scan API. Require release rejection and usable diagnostic output; do not modify production records.
- [ ] **G7 — Scanner failure:** Simulate missing tool or `binary_missing` / `failed` result, empty required-scanner list and zero scanner outcomes. All must block publication.
- [ ] **G8 — Tampered artifact:** Change one report byte or inject an invalid manifest entry. The local gate must reject the artifact even when the report's JSON claims success.
- [ ] **G9 — Findings are not silently ignored:** Produce a successful scan reporting finding events. The gate accepts **evidence completeness** but the report must explicitly say `REVIEW_FINDINGS`, identify review needed, and never claim code is secure.
- [ ] **G10 — Artifact/upload failure:** Deny Gitea Actions artifact upload and separately simulate release asset HTTP 401/500. Confirm publication fails without silent success. Check whether a partially created release or registry tag needs cleanup.
- [ ] **G11 — Security/operational overhead:** Confirm CI secrets do not appear in logs or downloadable artifacts, permissions are minimal, scan requests use report-only mode, and duplicate tag workflows do not exhaust scanner CPU/disk or interfere with production workloads.
- [ ] **G12 — Release record:** Record exact source SHA, report SHA-256, image digest, workflow run, scanner versions, pass/fail matrix, known unresolved findings, and reviewer approval. Do not call this a security certification.

## Known limitations

- A passing evidence gate validates **what was recorded**, not whether scanners found every defect. The source code and CI runner may still be compromised.
- Registry publishing and Gitea release attachment are not one global transaction. A network failure can leave some assets published; the external agent must test and document rollback/cleanup.
- Gitea's artifact uploader compatibility depends on the installed server and runner, and has not been tested here.
- Full Go 1.26.11 tests, production API protocol, live scanner coverage, multi-user RBAC, sandboxed remediation, and public release authenticity remain unverified in this environment.
- The source-push reports still remain separate CI artifacts. This change adds a **release-time** gate rather than retroactively proving a prior push's artifact.
