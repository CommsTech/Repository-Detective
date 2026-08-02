# First tester announcement (draft)

**Subject:** Repository Detective — private beta invitation (report-only)

---

Hi,

You're invited to the **private beta** of **Repository Detective** — *Inspect. Analyze. Improve.*

This is a **self-hosted** repository analysis tool for Gitea (GitHub supported). It runs security, quality, and health scanners, surfaces findings in a dashboard, and can optionally file issues — but **issue filing is off** for this beta cohort.

## What it does

- Connects to your forge with an API token
- Scans one repository with deterministic scanners (Trivy, Grype, Gitleaks, Semgrep, Go/IaC tools, linters)
- Shows findings, executive reports, code graph, and learning/calibration recommendations
- Supports **report-only** scans with **zero** forge side effects

## What it does not do (yet)

- File issues in your repos (disabled by default)
- Open remediation PRs
- Run all-repo bulk scans
- Require LLM / AI review
- Public beta — this cohort is **1–3 trusted testers only**

## Safety-first defaults

- First scan must use **report-only** mode
- One test repo per tester initially
- Secrets stay in your local `.env` — never commit or send them to us
- Feedback should use redacted logs only

## Install

Operator will provide the beta bundle: `repository-detective-beta/`  
Install path: `[OPERATOR_DISTRIBUTION_PATH — e.g. secure file share / internal artifact store]`

Quick steps:

1. Verify checksum: `sha256sum -c checksums.txt`
2. Copy `config.example.yaml` → `config/config.yaml`
3. Copy `.env.example` → `.env` and set your API key + Gitea token
4. `docker compose -f docker-compose.beta.yml up -d --build`
5. Open `http://127.0.0.1:8081/ui` and unlock with your API key

Full guide: `PRIVATE_BETA_TESTER_GUIDE.md` (included in source repo / operator pack).

## First scan

Use report-only when triggering analysis:

```json
{
  "owner": "your-org",
  "repo": "your-test-repo",
  "report_only_dry_run": true
}
```

Confirm **zero** new issues in Gitea after the scan.

## Known limitations

- SBOM in bundle may be absent unless operator built with cyclonedx tooling
- Large repos may hit scanner timeouts (document in feedback)
- macOS/Windows: use Docker; native binary is Linux-focused in this bundle
- Not public beta — do not redistribute the bundle

## Feedback

Please complete `PRIVATE_BETA_FEEDBACK_TEMPLATE.md` after your first scan and send to `[OPERATOR_CONTACT]`.

We especially want to hear about:

- Install friction
- False positives / missed findings
- Scanner failures
- UI confusion
- Executive report usefulness

## Do not send

- `.env` files
- Forge tokens or API keys
- `repository-detective.db` or full database dumps
- Unredacted logs containing credentials

## Questions?

Contact: `[OPERATOR_NAME / EMAIL / CHAT]`

Thank you for helping validate Repository Detective before wider release.

— Repository Detective operator team
