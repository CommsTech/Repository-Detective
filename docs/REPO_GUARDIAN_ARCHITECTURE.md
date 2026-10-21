# Repository Detective — Product & Technical Architecture Plan

**Status:** Living architecture plan. **Phase 5–6 complete** (DB + control-plane API/UI). Later phases are planned — not all implemented.

> **Naming:** [Repository Detective](NAMING.md) is the product name. **Repository-Detective** remains the internal/legacy service name for config, labels, and API compatibility.

**Commercial positioning:**

> **Repository Detective** — continuous security, quality, supply-chain, and maintenance assessment for Gitea, Forgejo, GitHub, and GitLab repos.

**Product framing:**

> A self-hosted, forge-native repository detective that continuously assesses repositories, blocks unsafe releases, tracks issues, and safely proposes tested fixes.

**Differentiator vs generic AI code review:**

1. Evidence-first, AI second
2. Self-hostable on homelab / air-gapped
3. Deep Gitea/Forgejo integration (status gates, issues, runners)
4. Third-party pre-install audit before deploying unknown software
5. Runner delegation so the core stays lightweight
6. Tested fix PRs only — never direct push to main
7. Low false-report tolerance via fingerprints, lifecycle, and policy gates

### Core behavioral split

Repo Guardian does **not** send emails. It generates **copy/paste disclosure reports** for third-party repos and automatically remediates only repos the user controls.

```text
Connected / owned repo:
detect → file issue → plan fix → test → remediation PR → rescan → evidence-based closure

Third-party repo:
audit → private report → disclosure draft → user approval → user sends/submits manually
```

**Product rule:**

> Owned repos can be acted on automatically under policy. Third-party repos get reports and drafts, not automatic action. Security-sensitive third-party findings default to private disclosure drafts.

---

## Current foundation (Phase 0–4B)

| Capability | Location today |
|------------|----------------|
| Webhook + manual scan | `main.go`, `handlers/webhook.go` |
| CAH pipeline | `analyzers/engine.go` |
| Scanner registry | `scanners/registry.go` (trivy → grype → gitleaks → semgrep → linters) |
| Workspace modes | `scanners/workspace_prepare.go` (`api` / `archive` / `auto`) |
| Commit status gates | `gitea/status.go`, `gitea/reporter.go` |
| Structured issues + fingerprints | `issues/template.go`, `issues/fingerprint.go` |
| Lifecycle labels/comments | `issues/lifecycle.go`, `issues/manager.go` |
| Semantic dedup (optional) | `memory/qdrant/`, `issues/semantic.go` |
| Remediation metadata (no patches) | `issues/enrich.go`, `ai/types.go` |
| Onboarding wizard only | `web/static/`, `handlers/onboarding.go` |
| **Phase 5** — SQLite persistence | `store/`, scans/findings/settings index |
| **Phase 6** — Control-plane API + UI | `api/`, `ui/`, `/api/v1/*`, `/ui/*` |

**Gaps (remaining):** repo settings not yet enforced on scans, scheduler, runners, pre-install audit, remediation engine, disclosure reports, community feed.

---

## 1. Product modes

### 1.1 Maintainer Mode

**Audience:** repos the operator owns or has write access to.

**Goals:** continuous safety + maintenance with configurable policy.

| Capability | Description | Current state |
|------------|-------------|---------------|
| Continuous webhook scans | Push/PR triggers analysis | ✅ Implemented |
| Scheduled full scans | Cron / Gitea schedule / internal ticker | ❌ Missing |
| PR/push status gates | Commit status from findings + scanner health | ✅ Phase 4A |
| Issue filing | Structured Gitea issues | ✅ Phase 4B |
| Lifecycle tracking | fingerprint, still-present, not-reproduced | ✅ Phase 4B (partial) |
| Repo-specific policy | scanners, thresholds, AI, remediation | ❌ Missing |
| Runner-based deep scans | offload Semgrep/Trivy/full tree | ❌ Missing |
| Safe remediation PRs | branch + patch + test + PR | ❌ Missing |
| Evidence-based closure | close only after merge + rescan | ❌ Missing |

**Default posture:** file issues, gate PRs on high severity, suggest fixes — no auto-merge.

**Connected-repo automation flow (under policy):**

```text
detect → file/update tracked issue → create remediation plan
→ create bounded fix branch → run tests before/after
→ open remediation PR only if tests pass → update issue with evidence
→ evidence-based closure only after merge + rescan proves fingerprint disappeared
```

### 1.2 Pre-Install Audit Mode

**Audience:** operator evaluating a third-party repo URL before install/deploy.

**Goals:** trust assessment with minimal upstream noise.

| Capability | Description |
|------------|-------------|
| Safe clone/archive | Isolated sandbox, size/time limits, no host secrets |
| Language/runtime detection | from tree + manifests |
| Dependency review | lockfiles, known CVEs, typosquat signals |
| Install script review | `postinstall`, `setup.py`, shell hooks, CI scripts |
| Secrets / malware patterns | gitleaks + static heuristics |
| CI/workflow review | dangerous triggers, secret exfil, unpinned actions |
| Container/IaC review | Dockerfile, compose, K8s manifests |
| Maintainer trust signals | activity, bus factor, signed releases, org age |
| License / supply-chain | SPDX, copyleft conflicts, missing license |
| Risk score + recommendation | install / caution / do-not-install |
| Disclosure reports | copy/paste markdown for email, security form, or forge issue |
| Issue draft | generated only when appropriate — **never auto-submitted** |

**Critical rule:** Pre-Install Audit Mode must **not** take automatic external action. No automatic third-party issue creation. No automatic security vulnerability public posting. No email sending.

**Third-party disclosure workflow:**

```text
pre-install audit
→ classify finding
→ determine sensitivity
→ generate private report
→ generate disclosure draft (public issue draft OR security disclosure draft)
→ user reviews/edits
→ user manually sends (copy/paste) OR approves optional forge submission
```

---

## 2. Architecture proposal

### 2.1 Component map

```
┌─────────────────────────────────────────────────────────────────┐
│                        Web UI (web/)                            │
│  Dashboard │ Repo Settings │ Findings │ Audit │ Issue │ Remediation │
└────────────────────────────┬────────────────────────────────────┘
                             │ REST + SSE
┌────────────────────────────▼────────────────────────────────────┐
│                   Core API Service (api/)                       │
│  auth │ repos │ scans │ findings │ policies │ audits │ jobs     │
└─────┬──────────┬──────────┬──────────┬──────────┬───────────────┘
      │          │          │          │          │
      ▼          ▼          ▼          ▼          ▼
┌─────────┐ ┌─────────┐ ┌─────────┐ ┌─────────┐ ┌──────────────┐
│ Policy  │ │ Scan    │ │ Issue   │ │ Runner  │ │ Pre-install  │
│ engine  │ │orch.    │ │ tracker │ │dispatch │ │ audit sandbox│
│policy/  │ │orch/    │ │issues/  │ │runner/  │ │audit/        │
└────┬────┘ └────┬────┘ └────┬────┘ └────┬────┘ └──────┬───────┘
     │           │           │           │             │
     └───────────┴───────────┴───────────┴─────────────┘
                             │
              ┌──────────────┼──────────────┐
              ▼              ▼              ▼
        ┌──────────┐  ┌──────────┐  ┌──────────┐
        │ Finding  │  │ Forge    │  │ Storage  │
        │ engine   │  │ adapters │  │ store/   │
        │analyzers/│  │forge/    │  │ (+Qdrant)│
        │scanners/ │  │gitea/…   │  │          │
        └──────────┘  └──────────┘  └──────────┘
```

### 2.2 Components (purpose, I/O, owner, storage, security)

| Component | Purpose | Inputs → Outputs | Go package | Persists | Security |
|-----------|---------|------------------|------------|----------|----------|
| **Core API** | HTTP API, auth, orchestration entry | HTTP → jobs, responses | `api/` (new) | audit logs | API keys, RBAC, rate limits |
| **Web UI** | Operator control plane | API ↔ browser | `web/` (extend) | none | session/auth, CSRF |
| **Forge adapters** | Gitea/Forgejo/GitHub/GitLab API | forge ops ↔ domain types | `forge/` (new), migrate `gitea/` | tokens encrypted | least-privilege tokens per forge account |
| **Scan orchestrator** | enqueue scans, correlate scan_id | webhook/cron/audit → ScanJob | `orch/` (new) | scans, jobs | sandbox boundaries |
| **Finding engine** | CAH pipeline + scanners | workspace → Findings | `analyzers/`, `scanners/` | via store | redact secrets in all outputs |
| **Policy engine** | per-repo rules, gates, reporting | repo + finding → allow/deny/action | `policy/` (new) | policies | prevent public disclosure mistakes |
| **Issue tracker** | Gitea issues + local index | Finding → issue/comment/label | `issues/` | issues, lifecycle_events | no secrets in bodies |
| **Runner dispatcher** | delegate heavy jobs | JobSpec → JobResult | `runner/` (new) | runner_jobs | signed results, no repo secrets on audit workers |
| **Remediation planner** | plan fixes, never apply yet | Finding → RemediationPlan | `remediation/` (new) | remediation_plans | bounded diffs |
| **Patch generator** | generate bounded patches (future) | plan → diff | `remediation/patch/` | patch_attempts | no secret rotation without SM integration |
| **Test executor** | run tests in runner/sandbox | patch + repo → pass/fail | `runner/testexec/` | patch_attempts | resource limits |
| **Pre-install audit sandbox** | isolated third-party audit | URL → AuditReport | `audit/` (new) | audit_requests | **no privileged secrets**, network egress controls |
| **Storage layer** | canonical state beyond Gitea | CRUD | `store/` (new) | PostgreSQL/SQLite | encryption at rest for tokens |
| **Qdrant (optional)** | semantic dedup **local instance only** | vectors | `memory/qdrant/` | vectors | **never share vectors or embeddings across instances** |
| **Disclosure generator** | copy/paste reports and issue drafts for third-party repos | finding + audit → markdown drafts | `report/disclosure/` (new) | disclosure_reports | no raw secrets, no weaponized exploits |
| **Report/export** | markdown/JSON/SARIF export | scan → report | `report/` (new) | ephemeral | sanitize exports |

---

## 3. Runner delegation design

### 3.1 Principles

- **Core decides policy and job type**; core never runs unbounded Semgrep on huge monorepos if a runner is available.
- **Runner executes** in repo context with declared resource limits.
- **Runner returns signed, structured JSON** (`JobResult`) — core validates signature + schema before filing issues.
- **Maintainer repos** may use repo-scoped credentials via Gitea Actions secrets.
- **Third-party audits** run on **isolated workers** with no access to operator forge tokens.

### 3.2 Job types

| Job type | Typical runner | Timeout | Resources |
|----------|----------------|---------|-----------|
| `quick_pr_scan` | core or runner | 5–10 min | changed files + manifests |
| `full_repo_scan` | runner preferred | 30–60 min | full archive workspace |
| `dependency_audit` | runner | 15 min | lockfiles only |
| `container_iac_audit` | runner | 15 min | Dockerfile/K8s/compose |
| `test_execution` | runner | 30 min | `go test`, `npm test`, etc. |
| `remediation_validation` | runner | 45 min | patch + before/after tests |
| `pre_install_audit` | isolated runner | 20 min | clone + scan, no secrets |

### 3.3 JobSpec / JobResult (proposed)

```go
// runner/types.go
type JobSpec struct {
    ID           string
    Type         JobType
    Repository   RepositoryRef  // forge-agnostic
    Ref          string
    WorkspaceMode string
    Scanners     []string
    PolicyID     string
    SandboxClass SandboxClass   // maintainer | third_party
    Limits       ResourceLimits
    CallbackURL  string          // core webhook for completion
    SignedBy     string          // core instance ID
}

type JobResult struct {
    JobID          string
    ScanID         string
    Status         string        // success | failed | timed_out
    ScannerResults []ScannerResultSummary
    Findings       []FindingRecord
    TestResults    *TestResultSummary
    Signature      string        // HMAC over canonical JSON
    RunnerID       string
    FinishedAt     time.Time
}
```

### 3.4 Forge runner priority

1. **Gitea Actions** (Phase 8) — `.gitea/workflows/repository-detective-runner.yml` triggered by `repository_dispatch` or labels
2. **GitHub Actions** (Phase 11+) — same contract, different adapter
3. **GitLab CI** (Phase 11+) — trigger pipeline with artifact upload
4. **Fallback:** in-process execution (current behavior) when `runner_preference: core`

### 3.5 Security controls

- Job signing: `HMAC-SHA256(core_secret, canonical_job_result)`
- Runner registration: one-time token per runner pool
- Third-party sandbox: separate network namespace, read-only root FS, egress allowlist (registry mirrors only)
- Secrets: never inject operator `gitea_token` into audit sandbox jobs

---

## 4. UI requirements

### 4.1 Dashboard (`/dashboard`)

- Repos monitored (count, health badge)
- Recent scans (scan_id, mode, duration, finding counts)
- Failing gates (repos with red commit status)
- Scanner failures (binary_missing, timed_out)
- Quick actions: run full scan, open repo settings

### 4.2 Repo settings (`/repos/:owner/:name/settings`)

Per-repo overrides (merge on global defaults):

| Setting | Type | Maps to |
|---------|------|---------|
| enabled | bool | webhook + schedule |
| scanners | map[string]bool | `enable_*` |
| workspace_mode | enum | api/archive/auto |
| analysis_depth | int | 1–3 |
| enable_llm_auditors | bool | AI policy |
| semgrep_config | string | operator ruleset |
| status_gate | bool + thresholds | `enable_gitea_status`, fail/warn |
| issue_filing | enum | off / fingerprint-only / all |
| auto_remediation | enum | policy level (see §6) |
| runner_preference | enum | core / gitea_actions / auto |
| schedule_cron | string | full scan cadence |
| allowed_fix_paths | []glob | remediation scope |

### 4.3 Findings (`/findings`)

- Filter: repo, severity, category, lifecycle, source, date range
- Columns: fingerprint, title, severity, first/last seen, linked issue #, linked PR #
- Actions: mark false-positive, request remediation, export evidence

### 4.4 Pre-install audit result (`/audit/:id`)

- Input: repo URL (https/git), optional ref, audit depth (quick / standard / deep)
- Output: risk score 0–100, recommendation, evidence tabs (deps, secrets, CI, container, license, trust)
- **Buttons (copy/export only — no “send email”):**
  - Copy summary
  - Copy install-risk report (supply-chain/pre-install template)
  - Copy security disclosure draft (if security-sensitive)
  - Copy public issue draft (if non-security and policy allows)
  - Export markdown
  - Mark as reviewed
- Optional later: **Approve submission** → forge adapter creates issue only after explicit approval (default off)

### 4.5 Connected repo issue page (`/repos/:owner/:name/findings/:fingerprint`)

For repos connected to the operator's forge account:

- Finding lifecycle state and history
- Linked Gitea/GitHub/GitLab issue
- Remediation readiness (`Fixable`, `SafeForAutoPR`, complexity, regression risk)
- Proposed remediation plan
- Test requirements (discovered + required)
- Approve remediation PR creation (when policy requires approval)
- Rescan result (fingerprint present/absent)
- Evidence-based closure record (merge + rescan proof)

### 4.6 Remediation (`/remediation`)

- Queue of `remediation-candidate` findings (**connected repos only**)
- Show: plan, files, tests required, risk, diff preview (future)
- Actions: approve remediation PR creation, reject, defer

**UI stack recommendation:** extend embedded `web/static/` with a small SPA (Alpine or vanilla JS initially) → later React/Vue if needed. API under `/api/v1/`.

---

## 5. Data model

**Recommendation:** PostgreSQL for production; SQLite for homelab single-node. Gitea remains issue SoT; DB is index + policy + audit history.

### 5.1 Tables / structs

#### `repositories`

```go
type Repository struct {
    ID          int64
    ForgeType   string    // gitea | forgejo | github | gitlab
    Owner       string
    Name        string
    FullName    string    // owner/name
    DefaultRef  string
    HTMLURL     string
    Enabled     bool
    CreatedAt   time.Time
    UpdatedAt   time.Time
}
```

#### `repo_settings`

```go
type RepoSettings struct {
    RepositoryID     int64
    PolicyLevel      string            // monitor_only | issue_only | gate_pr | suggest_fix | auto_pr_with_approval | auto_pr_low_risk
    Scanners         json.RawMessage   // per-scanner bools
    WorkspaceMode    string
    AnalysisDepth    int
    EnableLLM        bool
    StatusGate       bool
    FailOnSeverity   string
    WarnOnSeverity   string
    IssueFiling      string            // off | fingerprint | all
    RunnerPreference string
    ScheduleCron     string
    AllowedFixGlobs  []string
    CustomJSON       json.RawMessage   // forward-compatible
    UpdatedAt        time.Time
}
```

#### `scans`

```go
type Scan struct {
    ID           string    // scan_id (uuid)
    RepositoryID int64
    Mode         string    // webhook_push | webhook_pr | scheduled | manual | pre_install
    Ref          string
    CommitSHA    string
    Status       string    // pending | running | completed | failed
    StartedAt    time.Time
    FinishedAt   *time.Time
    SummaryJSON  json.RawMessage
}
```

#### `scanner_results`

```go
type ScannerResultRecord struct {
    ScanID   string
    Scanner  string
    Status   string
    Detail   string
    Findings int
}
```

#### `findings`

```go
type Finding struct {
    ID              int64
    Fingerprint     string    // unique per repo
    RepositoryID    int64
    Category        string
    Severity        string
    Source          string
    RuleID          string
    Title           string
    Description     string
    Fixable         string
    FixComplexity   string
    RegressionRisk  string
    SafeForAutoPR   bool
    FirstSeenScanID string
    LastSeenScanID  string
    FirstSeenAt     time.Time
    LastSeenAt      time.Time
    LifecycleState  string
    Confidence      float64
}
```

#### `finding_instances`

Per-scan occurrence (supports line drift, confidence changes):

```go
type FindingInstance struct {
    ID          int64
    FindingID   int64
    ScanID      string
    File        string
    Line        int
    LineBlock   int
    EvidenceHash string
    Confidence  float64
    CreatedAt   time.Time
}
```

#### `issues` (local index mirroring Gitea)

```go
type IssueLink struct {
    ID            int64
    FindingID     int64
    ForgeIssueNum int
    IssueURL      string
    CreatedAt     time.Time
}
```

#### `lifecycle_events`

```go
type LifecycleEvent struct {
    ID         int64
    FindingID  int64
    ScanID     string
    Event      string    // open | still_present | not_reproduced | needs_review | remediation_candidate | fixed | false_positive
    Detail     string
    Actor      string    // repository-detective | user@email
    CreatedAt  time.Time
}
```

#### `runner_jobs`

```go
type RunnerJob struct {
    ID          string
    ScanID      string
    Type        string
    Status      string
    RunnerID    string
    SpecJSON    json.RawMessage
    ResultJSON  json.RawMessage
    StartedAt   time.Time
    FinishedAt  *time.Time
}
```

#### `remediation_plans` / `patch_attempts`

```go
type RemediationPlan struct {
    ID                   int64
    FindingID            int64
    ConnectedRepoOnly    bool      // always true — never created for third-party audits
    RequiresUserApproval bool      // true when policy is auto_pr_with_approval or higher gate
    SafeForAutoPR        bool      // from finding + policy bounds check
    Strategy             string
    Files                []string
    RequiredTests        []string
    RiskLevel            string
    ApprovedBy           string
    Status               string    // draft | approved | rejected | applied
}

type PatchAttempt struct {
    ID              int64
    PlanID          int64
    BranchName      string
    PRNumber        int
    PRURL           string
    TestsBefore     json.RawMessage
    TestsAfter      json.RawMessage
    Status          string
    CreatedAt       time.Time
}
```

#### `audit_requests`

```go
type AuditRequest struct {
    ID              string
    SourceURL       string
    Ref             string
    Depth           string
    Status          string
    RiskScore       int
    Recommendation  string
    ReportJSON      json.RawMessage
    RequestedBy     string
    CreatedAt       time.Time
    CompletedAt     *time.Time
}
```

#### `forge_accounts`

```go
type ForgeAccount struct {
    ID          int64
    ForgeType   string
    BaseURL     string
    TokenEnc    []byte    // encrypted at rest
    Label       string
    CreatedAt   time.Time
}
```

#### `disclosure_reports`

Copy/paste disclosure artifacts for third-party audits. Repo Guardian never sends these — the user copies or exports manually.

```go
type DisclosureReport struct {
    ID                   int64
    AuditRequestID       string
    FindingID            *int64    // nil for whole-audit supply-chain reports
    ReportType           string    // general_bug | security_disclosure | supply_chain_risk
    Sensitivity          string    // public | security | supply_chain
    Title                string
    BodyMarkdown         string
    Confidence           float64
    GeneratedAt          time.Time
    ApprovedByUser       string    // set when user approves optional forge submission
    SubmittedExternally  bool      // user marked as sent/submitted
    SubmissionTarget     string    // email | security_form | github_issue | gitea_issue | gitlab_issue | other
    SubmissionNotes      string    // user notes: where/how they sent it
}
```

#### `policies` (optional templates)

```go
type PolicyTemplate struct {
    ID          int64
    Name        string
    Description string
    SettingsJSON json.RawMessage
}
```

### 5.2 Keys and indexes

- `findings`: unique `(repository_id, fingerprint)`
- `finding_instances`: index `(finding_id, scan_id)`
- `scans`: index `(repository_id, started_at DESC)`
- `lifecycle_events`: index `(finding_id, created_at)`
- `audit_requests`: index `(created_at DESC)`
- `disclosure_reports`: index `(audit_request_id, report_type)`

---

## 6. Policy model

### 6.1 Policy levels (connected / owned repos only)

Third-party pre-install audits are **never** subject to remediation or auto-issue policy. They produce reports and drafts only.

| Level | Webhooks | Status gate | Issues | Remediation |
|-------|----------|-------------|--------|-------------|
| `monitor_only` | scan + log | off | off | off |
| `issue_only` | scan | off | on | off |
| `gate_pr` | scan | on (configurable thresholds) | on | off |
| `suggest_fix` | scan | on | on | remediation plan in UI only |
| `auto_pr_with_approval` | scan | on | on | remediation PR after user approval + passing tests |
| `auto_pr_low_risk` | scan | on | on | automatic remediation PR for bounded low-risk fixes + passing tests |

### 6.1.1 `auto_pr_low_risk` constraints

Automatic remediation PRs allowed only when **all** of:

- high-confidence finding (meets `MinConfidenceIssue` and `SafeForAutoPR`)
- small bounded diff (within `AllowedFixGlobs`, max files/lines)
- tests discovered and runnable in runner
- before/after tests pass with no new failures
- no push to protected/default branch (PR only)
- no secret rotation
- no broad refactors
- no dependency major-version jumps unless explicitly allowed in repo policy

### 6.2 Policy dimensions

```go
type PolicyRules struct {
    MinConfidenceIssue    float64
    MinConfidenceGate     float64
    FailOnSeverity        string
    WarnOnSeverity      string
    BlockCategories       []string
    AllowCategories       []string
    ScannerFailureIsError bool
    AllowLLM              bool
    AllowRunners          bool
    PublicIssueRules      PublicIssueRules
    PreInstallRules       PreInstallRules
}

type PublicIssueRules struct {
    AllowExternalSubmit    bool     // default false — copy/paste only
    MinConfidence          float64  // default 0.95
    RequireDeterministic   bool
    RequireUserApproval    bool     // always true for third-party
    SecurityDefaultPrivate bool     // security → security disclosure draft, not public issue
    NoWeaponizedExploit    bool     // proof without exploit unless strictly necessary
    NoMalwareAccusations   bool     // no "malware" unless evidence is conclusive
}
```

### 6.3 Merge order

```
global config.yaml → policy template → repo_settings → .repository-detective.yaml in repo (optional, maintainer repos only)
```

Pre-install audits ignore repo `.repository-detective.yaml` (untrusted input).

---

## 7. Third-party disclosure workflow

Repo Guardian does **not** send emails or automatically post to third-party forges. It generates high-quality **disclosure reports** the user copy/pastes into an email, security contact form, GitHub/GitLab/Gitea issue, or maintainer discussion.

### 7.1 Flow

```text
pre-install audit complete
  → classify finding (deterministic? security? supply-chain?)
  → determine sensitivity
  → generate private report (stored in disclosure_reports)
  → generate appropriate disclosure draft:
      - security disclosure draft (default for vulnerabilities)
      - public issue draft (non-security, if policy allows)
      - supply-chain/pre-install risk report (audit-level)
  → user reviews/edits in UI
  → user copies markdown OR exports OR marks as reviewed
  → optional: user approves forge submission → adapter creates issue (default off)
  → user records submission_target + submission_notes when sent manually
```

**Defaults:** no automatic third-party issue creation. No automatic security vulnerability public posting.

### 7.2 Disclosure report templates

#### A. General quality/bug report (`report_type: general_bug`)

For non-security issues. Fields:

- title
- affected repo / version / commit
- summary
- observed behavior
- expected behavior
- reproduction steps
- evidence (redacted)
- impact
- suggested fix
- environment
- generated-by Repo Guardian notice + timestamp

#### B. Security disclosure report (`report_type: security_disclosure`)

For vulnerabilities. Fields:

- title
- affected repo / version / commit
- vulnerability class
- severity estimate
- confidence
- private reproduction steps
- impact
- affected files/functions
- proof/evidence — **no weaponized exploit unless strictly necessary**
- suggested remediation
- disclosure handling recommendation (coordinated disclosure)
- request for security contact
- generated timestamp
- **no raw secrets**

#### C. Supply-chain / pre-install risk report (`report_type: supply_chain_risk`)

For unsafe dependency/install risk (whole-audit or finding-level). Fields:

- repo URL
- commit audited
- install method reviewed
- risky scripts or workflows
- suspicious dependencies
- network / file / system access concerns
- recommendation: safe / caution / do not install
- evidence

### 7.3 Third-party submission policy

**Public issue draft allowed only when:**

- non-sensitive **OR** maintainer has no private disclosure path **and** user approves
- deterministic or independently validated
- confidence ≥ 0.95
- reproducible steps included
- no raw secrets
- no weaponized exploit
- no private/third-party data exposed
- no accusations like "malware" unless evidence is conclusive
- not stylistic/low-value categories

**Security-sensitive findings:**

- default to **security disclosure draft** (private tone)
- do not auto-open public issue
- recommend checking `SECURITY.md`, repo security policy, maintainer website, or project contact
- if no contact exists: generate a careful neutral message asking for a security contact
- user copy/pastes into appropriate channel manually

### 7.4 Optional approved forge submission

If user explicitly approves (future Phase 15):

- [ ] all gates in §7.3 pass
- [ ] `SanitizeSecretEvidence()` passes on all text fields
- [ ] `approved_by_user` recorded on `disclosure_reports`
- [ ] forge adapter creates issue (security advisory if supported)
- [ ] log submission in `disclosure_reports.submitted_externally`

This is **optional** and off by default. Primary UX is copy/paste.

---

## 8. Connected-repo remediation safety process

**Scope:** connected/owned repos only. Third-party audits never enter this flow.

### 8.1 State machine

```
finding (remediation-candidate) on connected repo
  → planner creates RemediationPlan (ConnectedRepoOnly=true)
  → policy check (level, SafeForAutoPR, fix complexity)
  → if auto_pr_with_approval: wait for user approval
  → test discovery (existing tests for affected package?)
  → create branch repository-detective/fix/<fingerprint-short>
  → apply bounded patch (max N files, max M lines)
  → runner: tests BEFORE
  → runner: tests AFTER
  → compare: no new failures, target test passes
  → open remediation PR with linked issue + scan evidence
  → update tracked issue with evidence
  → on merge: schedule rescan
  → rescan: fingerprint absent → evidence-based closure (Phase 14)
```

### 8.2 Hard constraints

| Rule | Enforcement |
|------|-------------|
| Never push to default/protected branch | forge API: always via PR from `repository-detective/fix/*` |
| Max diff size | e.g. 3 files, 80 lines — policy configurable |
| No broad rewrite | reject plans touching > X% of repo |
| No secret rotation without SM | `category=secret` → manual only |
| No fix claim without rescan | `fixed` lifecycle requires fingerprint miss on 2 consecutive scans (configurable) |
| Tests must pass | `patch_attempts.tests_after` all green |

### 8.3 Remediation planner output

```go
type RemediationPlan struct {
    FindingFingerprint string
    ProposedFix        string
    FilesToChange      []string
    TestsToRun         []string
    RiskLevel          string
    RollbackPlan       string
    WhySafe            string
    EstimatedLines     int
}
```

---

## 9. Forge abstraction

### 9.1 Interface (new package `forge/`)

```go
type ForgeClient interface {
    Type() ForgeType
    TestConnection(ctx context.Context) error

    // Repository
    GetRepository(ctx context.Context, owner, repo string) (*Repository, error)
    DownloadArchive(ctx context.Context, owner, repo, ref string, opts ArchiveOpts) (Archive, error)
    ListFiles(ctx context.Context, owner, repo, ref, path string) ([]FileEntry, error)
    GetFile(ctx context.Context, owner, repo, ref, path string) ([]byte, error)

    // Pull requests
    GetPullRequest(ctx context.Context, owner, repo string, number int) (*PullRequest, error)
    ListChangedFiles(ctx context.Context, owner, repo string, prNumber int) ([]string, error)
    CreatePullRequest(ctx context.Context, owner, repo string, req CreatePRRequest) (*PullRequest, error)

    // Issues
    CreateIssue(ctx context.Context, owner, repo string, req CreateIssueRequest) (*Issue, error)
    ListIssues(ctx context.Context, owner, repo string, opts ListIssuesOpts) ([]Issue, error)
    CreateIssueComment(ctx context.Context, owner, repo string, issueNumber int, body string) error
    AddIssueLabels(ctx context.Context, owner, repo string, issueNumber int, labels []string) error

    // Status
    CreateCommitStatus(ctx context.Context, owner, repo, sha string, status CommitStatus) error

    // Webhooks
    CreateWebhook(ctx context.Context, owner, repo string, hook WebhookConfig) error

    // Runners (optional capability)
    RunnerSupport() RunnerCapability
    DispatchRunnerJob(ctx context.Context, job RunnerJobSpec) error
}
```

### 9.2 Migration path

| Phase | Action |
|-------|--------|
| Now | `gitea/` implements concrete client |
| Phase 11 | Extract `forge.ForgeClient`; `gitea.Client` wraps adapter |
| Phase 11 | Add `forgejo/` (likely thin wrapper over Gitea adapter) |
| Phase 12+ | `github/`, `gitlab/` adapters |

Naming: use `Repository`, `Issue`, `PullRequest` in `forge/` — not Gitea-specific types in business logic.

---

## 10. Implementation phases

### Completed

| Phase | Status | Summary |
|-------|--------|---------|
| **5** | ✅ Done | SQLite persistence, repo registry, scans, findings, settings |
| **6** | ✅ Done | Control-plane API + operator UI; settings stored but **not enforced** on scans yet |

### Planned (in recommended order)

### Phase 7 — Scheduled full scans

| | |
|--|--|
| **Goal** | Cron per `repo_settings.schedule_cron`; full archive scans |
| **Packages** | `orch/scheduler.go`, `main.go` background worker |
| **Tests** | scheduler tick, dedup overlapping scans, scan record persistence |
| **Risk** | Medium — load on large repos |
| **Rollback** | Disable scheduler via config |

### Phase 8 — Apply per-repo settings to scan behavior

| | |
|--|--|
| **Goal** | Wire `ResolveRepoSettings()` into webhook, manual, and scheduled scan paths so stored policy/scanners/workspace actually drive runtime |
| **Packages** | `policy/`, `main.go`, `analyzers/engine.go` (read effective settings) |
| **Tests** | per-repo scanner toggles, workspace mode, confidence gate; global fallback when unset |
| **Risk** | Medium — behavior change for existing deployments |
| **Rollback** | Feature flag `enforce_repo_settings: false` (global config only) |

### Phase 9 — Pre-install audit mode

| | |
|--|--|
| **Goal** | `/audit` UI + isolated scan pipeline + risk score |
| **Packages** | `audit/`, `audit/scorer.go`, `audit/trust.go`, sandbox runner job type |
| **Tests** | no secret leakage, URL validation, sandbox class enforcement |
| **Risk** | High — untrusted code execution surface |
| **Rollback** | Disable audit feature flag |

### Phase 10 — Richer non-security analysis

| | |
|--|--|
| **Goal** | Tech debt, test gaps, AI-generated-risk heuristics (evidence-based) |
| **Packages** | `scanners/` (complexity, dead code), `analyzers/heuristics/` |
| **Tests** | category mapping, no overclaim wording |
| **Risk** | Medium — noise if thresholds wrong |
| **Rollback** | Disable new scanners per repo |

### Phase 11 — Runner job abstraction

| | |
|--|--|
| **Goal** | JobSpec/JobResult contract; Gitea Actions dispatch; core fallback |
| **Packages** | `runner/`, `.gitea/workflows/repository-detective-runner.yml`, `api/jobs.go` |
| **Tests** | signature validation, timeout, result schema |
| **Risk** | High — runner security |
| **Rollback** | `runner_preference: core` |

### Phase 12 — Forge abstraction

| | |
|--|--|
| **Goal** | `forge.ForgeClient`; Gitea adapter; Forgejo compat |
| **Packages** | `forge/`, refactor `gitea/` → `forge/gitea/` |
| **Tests** | adapter contract tests, mock forge |
| **Risk** | Medium — refactor blast radius |
| **Rollback** | Keep Gitea-only adapter behind interface |

### Phase 13 — Remediation planner

| | |
|--|--|
| **Goal** | Plans from findings; UI queue; no patches yet |
| **Packages** | `remediation/planner.go`, `api/remediation.go` |
| **Tests** | plan bounds, SafeForAutoPR gating |
| **Risk** | Low |
| **Rollback** | Disable remediation UI |

### Phase 14 — Safe patch PRs

| | |
|--|--|
| **Goal** | branch + bounded patch + test gate + PR creation |
| **Packages** | `remediation/patch/`, `runner/testexec/`, forge PR API |
| **Tests** | diff limits, test before/after, no main push |
| **Risk** | **Critical** |
| **Rollback** | Policy level max `remediation_suggest` |

### Phase 15 — Evidence-based closure

| | |
|--|--|
| **Goal** | Auto-close (optional) when fingerprint gone + tests pass + merge detected |
| **Packages** | `issues/lifecycle.go`, webhook on merge, rescan hook |
| **Tests** | two-scan confirmation, no close on flaky miss |
| **Risk** | Medium |
| **Rollback** | `auto_close: false` default |

### Phase 16 — Public upstream disclosure workflow

| | |
|--|--|
| **Goal** | Disclosure report generator, issue draft generator, copy/export UX, optional approved submission — **not** automatic public reporting or email |
| **Packages** | `report/disclosure/`, `audit/disclosure.go`, `policy/public.go`, UI copy/export + approval flow |
| **Deliverables** | Templates A/B/C, `disclosure_reports` store, copy buttons, markdown export |
| **Tests** | gate rejects low confidence; security → private disclosure draft; no auto-submit |
| **Risk** | **Critical** — reputational |
| **Rollback** | `allow_external_submit: false`; copy/paste-only mode |

### Phase 17 — Community Intelligence Feed

| | |
|--|--|
| **Goal** | Signed, sanitized, opt-in intelligence sharing — **not** shared Qdrant or peer-to-peer vectors |
| **Packages** | `intel/`, `intel/sanitize.go`, `intel/feed.go`, optional managed feed service |
| **Prerequisite** | Phases 7–15 mature; product safety proven in production |
| **Risk** | **Critical** — data leakage if sanitization fails |
| **Rollback** | Disable feed pull/push; local-only mode |

*(Product milestone: community feed = Phase 16 in product roadmap; implementation Phase 17 because Forge abstraction is Phase 12.)*

**Do not build before core product safety is mature.**

---

## 11. Community Intelligence Feed (Phase 17 — product milestone Phase 16)

### Design principle

```text
Do not share code.
Do not share raw vectors.
Do not share private repo metadata.
Share only sanitized intelligence packages.
```

**Not:** a shared Qdrant between Repository-Detective instances. Embeddings can leak semantic information about private code, architecture, filenames, or secrets if redaction fails. Local Qdrant remains **per-instance, private dedup only**.

**Instead:** a **Repository-Detective Intelligence Feed** — federated trust, one-way pull by default, not peer-to-peer shared memory.

### Architecture

```text
Local Repository-Detective
  → sanitizes candidate intelligence
  → strips repo identity
  → operator opt-in required
  → private repos excluded by default
  → signs payload
  → submits to community/managed feed

Community Feed
  → validates schema
  → rejects sensitive-data patterns
  → aggregates + signs updates
  → publishes versioned intel packages
  → supports revocation

Other scanners
  → pull signed updates (one-way)
  → local policy decides whether to apply
  → never auto-execute without policy gate
```

### Shareable record types

| Type | Example use |
|------|-------------|
| `false_positive_pattern` | Semgrep rule + safe condition → suppress noise |
| `scanner_rule_improvement` | Improved rule pack metadata |
| `remediation_recipe` | Package upgrade path + test recommendations |
| `package_vulnerability_metadata` | Public CVE + affected range (no private context) |
| `dependency_upgrade_recipe` | Safe version bump for known bad range |
| `public_repo_risk_signal` | OpenSSF Scorecard-style supply-chain signal |
| `tool_reliability_metadata` | Scanner timeout/false-positive rates (aggregated) |
| `finding_hash_public` | Hash of public-repo finding for dedup across community |

Example record:

```json
{
  "type": "false_positive_pattern",
  "scanner": "semgrep",
  "rule_id": "python.lang.security.audit.dangerous-subprocess-use",
  "language": "python",
  "condition": "subprocess called with fixed argv and shell=false",
  "confidence": 0.92,
  "source_count": 37,
  "created_by": "signed-community-feed",
  "version": 1
}
```

### Never share

- Private repo code, embeddings, or Qdrant vectors
- Raw scanner output from private repos
- Secret findings, file paths, issue bodies from private repos
- Dependency graphs tied to private orgs
- Author names/emails, internal URLs
- Private repo names or forge metadata

### Security requirements

- Operator opt-in for contribute and consume
- Private repos excluded from contribution by default
- Schema validation + sensitive-data rejection at feed
- Signed updates; trust levels (community vs verified vendor feed)
- Revocation support for bad intel packages
- No automatic rule execution without local policy approval
- Audit log of applied intel packages

---

## 12. Monetization model

### Differentiators vs generic AI code review (Sourcery-style PR comments)

| Repo Guardian | Generic AI review |
|---------------|-------------------|
| Gitea/Forgejo-first | GitHub-centric |
| Self-hosted or managed | SaaS-only |
| Evidence-backed findings + lifecycle | One-shot comments |
| Pre-install repo audits | N/A |
| Runner-based heavy scans | In-service only |
| Safe remediation PRs + test gates | Suggestions only |
| Disclosure report generation | N/A |
| Security + reliability + tech debt | Often security-only |

### Offerings

#### Community / self-hosted (open core)

Free tier target:

- Gitea support
- Local SQLite
- Basic scanners (trivy, grype, linters)
- Manual + webhook scans
- Issue filing + fingerprint lifecycle
- Basic UI + local pre-install audit
- No community feed push by default

#### Pro self-hosted (paid license)

- Advanced policies + enforced per-repo settings
- Scheduled scans
- Runner delegation
- Advanced dashboards
- Remediation planning + disclosure templates
- Multi-forge (Forgejo, GitHub, GitLab)
- PostgreSQL support
- Community intelligence feed (pull/consume)

#### Managed SaaS

Hosted Repo Guardian subscription:

- **Pricing sketch:** $15–30/developer/month **or** $49–299/org/month by repo count
- **Starter:** ~10 repos, weekly scans, issue filing
- **Team:** ~50 repos, daily scans, PR gates, dashboards
- **Business:** 200+ repos, runner delegation, remediation PRs
- **Enterprise:** SSO, dedicated instance, compliance exports, private intel feed channel

#### Professional audit service (early revenue)

One-time paid assessments — aligns with pre-install audit mode:

| Tier | Price (sketch) | Deliverables |
|------|----------------|--------------|
| Small repo | $99 | Risk report, evidence summary, install recommendation |
| Serious repo | $299 | Full supply-chain review, dependency audit, remediation plan |
| Organization | $999+ | Multi-repo assessment, compliance-style export |

Deliverables: markdown/PDF report, risk score, supply-chain review, disclosure drafts when needed, remediation plan.

**Recommendation:** Start monetization with **paid repo assessments** before polished SaaS. Managed subscriptions follow once Phase 7–11 are stable.

---

## Strategic recommendation (updated post Phase 6)

**Do not build community sharing yet. Do not share Qdrant between instances.**

Implement next:

1. **Phase 7** — scheduled full scans
2. **Phase 8** — **apply per-repo settings to scan behavior** (settings are stored in Phase 6 but not enforced)
3. **Phase 9** — pre-install audit
4. **Phase 11** — runner delegation (after audit + settings enforcement)

Phase 6 delivered the control plane. The highest-value gap is **making stored settings actually drive scans**, then **scheduled maintenance scans**.

Community Intelligence Feed comes **after** disclosure workflow, remediation PRs, and evidence-based closure are proven safe — **not before Phase 7–8**.

---

## Appendix A — Package layout (target)

```
git.commsnet.org/commstech/repository-detective/
├── api/              # REST handlers (new)
├── audit/            # pre-install audit (new)
├── forge/            # forge abstraction (new)
│   ├── gitea/        # migrated from gitea/
│   ├── forgejo/
│   ├── github/
│   └── gitlab/
├── orch/             # scan orchestrator + scheduler (new)
├── policy/           # policy engine (new)
├── remediation/      # planner + patch (new)
├── runner/           # job dispatch + results (new)
├── report/           # export formats (new)
│   └── disclosure/   # copy/paste disclosure reports + issue drafts
├── store/            # SQL persistence (new)
├── analyzers/        # existing CAH pipeline
├── scanners/         # existing registry
├── issues/           # existing + DB sync
├── memory/qdrant/    # optional LOCAL semantic dedup only — never shared
├── intel/            # community intelligence feed client (Phase 17)
├── ui/               # operator UI (Phase 6)
└── main.go
```

---

## Appendix B — AI-generated code risk (Phase 10 wording)

Never claim authorship. Use evidence-based phrasing:

> Possible low-context or AI-generated code risk, based on:
> - weak error handling
> - hallucinated API usage
> - generic comments mismatched to behavior
> - missing tests around generated-looking blocks
> - unused abstractions / repetitive boilerplate

Category: `ai_generated_risk`. Confidence capped below deterministic secrets/CVEs unless multiple independent heuristics fire.

---

## Appendix C — Terminology

Use consistently across code, UI, and docs:

| Term | Meaning |
|------|---------|
| **Disclosure report** | Copy/paste markdown for third-party communication (replaces "email") |
| **Issue draft** | Forge-ready markdown for third-party repos — not auto-created |
| **Security disclosure draft** | Private-tone vulnerability report — default for security findings |
| **Remediation PR** | Fix PR on connected/owned repos only |
| **Evidence-based closure** | Close tracked issue only after merge + rescan proves fingerprint gone |
| **Connected repo** | Repo linked to operator forge account with write access |
| **Third-party repo** | Pre-install audit target — reports and drafts only |
| **Intelligence feed** | Signed sanitized intel packages — not shared Qdrant |
| **Local Qdrant** | Per-instance semantic dedup only — private vectors never exported |

**Avoid:** "send email", "auto issue" (for third-party), "automatic public reporting", "shared Qdrant", "peer-to-peer embeddings".

---

*Document version: 1.2 — monetization + community intelligence feed roadmap (post Phase 6).*
