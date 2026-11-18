# CVE Triage and AI-Assisted Remediation

**Scope:** Repository Detective, unreleased changes after `v0.1.0-beta.6`. This guide does **not** certify end-to-end production readiness. Gitea E2E, full CI, and the remediation sandbox still require validation on the deployment target.

## What it does now

1. The existing Trivy, Grype, and govulncheck scanner paths identify vulnerability findings in repositories/images. Scanner evidence remains authoritative about *what was found*; a CVE ID alone cannot establish asset reachability.
2. A canonical finding collects repeated hits; an unchanged finding reuses its existing remediation plan instead of generating another plan and consuming AI tokens each scan.
3. The finding details page can explicitly query [CISA's KEV catalog](https://www.cisa.gov/known-exploited-vulnerabilities-catalog) and [FIRST's EPSS API](https://www.first.org/epss/) for a single valid CVE. KEV is evidence of known exploitation. EPSS estimates probability of exploitation in the next 30 days; neither proves a particular deployment is exposed.
4. Remediation plans use deterministic guidance, and (if explicitly enabled) AI adds version-aware investigation, tests, and rollback considerations. AI output is **advisory and requires review**.
5. Existing deterministic safe-PR patchers remain limited to supported low-risk lint/misconfiguration recipes. **Dependency upgrades, lockfile modifications, secrets, and high-impact security changes do not receive unattended PRs.** Owners make the patch, then scan/verify closure.

## Minimum setup (isolated staging first)

Use a clean `.env` and staging Gitea repository; never expose production credentials in reports or AI prompts.

```dotenv
REPOSITORY_DETECTIVE_AUTH_MODE=local
REPOSITORY_DETECTIVE_API_KEY=<unique-generated-hex-value>
REPOSITORY_DETECTIVE_SESSION_SECRET=<different-unique-generated-hex-value>
REPOSITORY_DETECTIVE_GITEA_URL=https://gitea.example.net
REPOSITORY_DETECTIVE_GITEA_TOKEN=<scoped-token>
REPOSITORY_DETECTIVE_SCAN_PROFILE=standard
REPOSITORY_DETECTIVE_ENABLE_TRIVY=true
REPOSITORY_DETECTIVE_ENABLE_GRYPE=true
REPOSITORY_DETECTIVE_REMEDIATION_PLANNER_ENABLED=true
REPOSITORY_DETECTIVE_REMEDIATION_USE_AI=false
REPOSITORY_DETECTIVE_REMEDIATION_PR_ENABLED=false
REPOSITORY_DETECTIVE_AUTO_CREATE_ISSUES=false
REPOSITORY_DETECTIVE_CVE_INTELLIGENCE_ENABLED=true
```

For public CVE intelligence **opt in** with `REPOSITORY_DETECTIVE_CVE_INTELLIGENCE_ENABLED=true` and restart. Open a dependency finding and select **Check live signals**. Enrichment is on by default and still only runs on explicit operator action; the authenticated `/ui/findings/:id/intelligence` endpoint sends only the CVE ID to CISA and FIRST. There are no background lookups and no page-load requests. Set the flag to `false` for air-gapped or fully local-only deployments. KEV is cached for up to six hours; EPSS for up to an hour; upstream errors are explicitly marked as unavailable. Keep off for fully isolated networks.

For **local AI advisory remediation**, additionally set:

```dotenv
REPOSITORY_DETECTIVE_AI_PROVIDER=ollama
REPOSITORY_DETECTIVE_AI_BASE_URL=http://<ollama-reachable-host>:11434
REPOSITORY_DETECTIVE_AI_MODEL=<installed-model-name>
REPOSITORY_DETECTIVE_REMEDIATION_USE_AI=true
```

The AI endpoint must be reachable **from inside** the Repository Detective container. `127.0.0.1` inside that container is not the host's loopback. Match the actual model and network route; use a private authenticated gateway when crossing trust zones, or HTTPS with a trusted CA. Never enable insecure TLS verification for internet-facing endpoints. AI failure leaves a human-review deterministic plan, with an explicit warning. Keep `REMEDIATION_COMMENT_ON_ISSUE=false` to avoid repeated issue noise.

The `AI_RECOMMENDATIONS_*` settings are a *separate* scan advisory feature and do not enable remediation-plan AI. See [AI_PROVIDERS.md](AI_PROVIDERS.md) and [OPENCLAW_INTEGRATION.md](OPENCLAW_INTEGRATION.md).

## Operator queue and reporting

The dashboard's **Triage critical/high** action opens `/ui/findings?focus=1`: this shows open critical/high findings and prioritizes severity in SQL **before** page limits. Paginated queue views show 50 focus or 100 general findings per page and total match counts. CSV and JSON exports include all matching rows up to 20,000, not merely the visible page; larger requests fail explicitly and should be filtered. CSV guards spreadsheet-formula text, whereas JSON preserves raw finding text. See the [five-minute operator workflow](OPERATOR_TRIAGE_WORKFLOW.md).

## Operator decision flow

1. **Scanner evidence first:** Confirm package/version, image digest or lockfile, and the exact CVE. If the scanner failed or package presence is unknown, mark the investigation incomplete.
2. **Prioritize:** KEV match is an urgent signal; EPSS supports prioritization; correlate with deployed assets, public exposure, reachability, privilege, business impact, and available fixed versions. High CVSS alone is not proof of exploitation.
3. **Plan:** Generate the plan from finding detail. If AI is configured, inspect its suggestions against scanner/vendor evidence; do not blindly trust suggested versions or commands.
4. **Remediate:** Update the dependency/lockfile manually or through a separately approved, sandboxed workflow, test, build and redeploy the affected artifact. Unattended dependency PRs remain disabled.
5. **Verify:** Rescan the same artifact with the required scanners operational. Confirm the affected package version is gone, the finding is absent, and deployment coverage was complete. A merged PR alone is not closure evidence.
6. **Report:** Record evidence (scanner run ID, version/digest, PR, test results, accepted exceptions). Prefer one canonical issue per fingerprint over one ticket per repeated scan.

## Safety and privacy

- `LOCAL_ONLY` should normally keep `CVE_INTELLIGENCE_ENABLED=false` unless the operator intentionally allows external **public CVE ID** lookup. No source code or asset metadata is sent by the enrichment endpoint.
- The app must not mistake HTTP errors, missing EPSS scores, or stale signals for a safe or non-exploitable CVE.
- AI enrichment does not modify fix files, patch eligibility, auto-merge policy, or authority. The deterministic safe-PR checklist still applies.
- Keep Gitea/AI tokens outside git. Restrict application ingress and webhook privileges; do not publish the API key as a URL parameter.
- CISA's government remediation deadlines apply to relevant mandated entities. Here, the dates are informational, not an automatic compliance or patch deadline for every user.

## Verification gates before production

- [ ] Build with the `go1.26.10` toolchain specified in `go.mod`; pass `go test ./...`, `go vet ./...` and CI.
- [ ] Clean-install the built image and confirm `/health/ready` is ready.
- [ ] Execute a known disposable vulnerable package fixture and confirm Trivy/Grype/govulncheck parsing and deduplication.
- [ ] With enrichment off, confirm **no** CISA/FIRST requests; with it on, confirm the endpoint is authenticated, CVE-only and handles unreachable upstreams.
- [ ] With local AI off/on/offline, confirm deterministic output, AI advisory output, and explicit fallback with human review.
- [ ] Confirm repeated scans preserve existing proposed/approved/rejected plans and do not flood Gitea issues or AI.
- [ ] Validate a supported low-risk patch PR in a disposable Gitea repo. Confirm validation failure prevents a PR, and that no workflow auto-merges.
- [ ] Validate closure after merge/redeploy/rescan, including incomplete scanner coverage and regression/fingerprint changes.
- [ ] Check light/dark/mobile rendering, keyboard navigation, screen-reader status updates, auth and CSRF, proxy/TLS and secret hygiene.
- [ ] Capture an immutable image digest and completed E2E acceptance receipt before advertising a stable release.

## Rollback

Set `REPOSITORY_DETECTIVE_CVE_INTELLIGENCE_ENABLED=true`, `REPOSITORY_DETECTIVE_REMEDIATION_USE_AI=false`, and `REPOSITORY_DETECTIVE_REMEDIATION_PR_ENABLED=false`, then restart the container. Retain plans and findings as historical records; reverting the app image does not reverse any manually merged code changes.
