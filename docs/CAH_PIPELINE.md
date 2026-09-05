# Repository Detective CAH Pipeline

**Spec and implementation notes.** For day-to-day setup, see [SETUP.md](SETUP.md).

---

## What is implemented today

| Stage | Status | Notes |
|-------|--------|-------|
| PREPARE | Partial | File tree + LLM attack surface mapping. No call graph or git history yet. |
| SCAN | Done | Static regex, **Trivy, Grype, linters**, then LLM auditors on flagged files only. |
| VALIDATE | Done | Advocate/counsel debate. Deterministic findings skip debate. |
| DEDUP | Partial | Line-block clustering with `cluster-000` IDs; forge issue dedup via fingerprints + SQLite mappings |
| PROVE | Partial | LLM-generated PoC (curl/scripts). No ASan/UBSan execution. |

Auditors running today: SQL, XSS, auth, injection, crypto, config (+ static rules; **Trivy, Grype, golangci-lint, ruff, shellcheck**).

See [SCANNERS.md](SCANNERS.md) for deterministic scanner configuration.

Not implemented: call graph builder, git history analyzer, memory/race auditors as separate agents, web dashboard.

---

## Overview (target design)

Repository-Detective's goal is to implement a CAH-style multi-agent security pipeline for Gitea repositories. Unlike simple AI code analysis tools that do a single-pass "analyze this code" prompt, Repository-Detective should orchestrate multiple specialized agents through a structured discovery → validation → proof pipeline.

---

## The 5 Stages

### Stage 1: PREPARE

**Purpose:** Understand the code before scanning it.

**What happens:**
1. Fetch repository structure (file tree, entry points, public APIs)
2. Build a language-aware call graph (which functions call which)
3. Map the attack surface (public-facing functions, I/O boundaries, auth checks)
4. Identify trust boundaries (user input → internal logic → sensitive operations)
5. Note historical vulnerability patterns from git history

**Output:** A `PrepareReport` containing:
- File structure with language classifications
- Call graph (functions → functions they call)
- Attack surface map (entry points, I/O, auth)
- Trust boundary crossings
- Historical vulnerability context

**Specialized agents:**
- `StructureMapper` — Understands repo layout, identifies key files
- `CallGraphBuilder` — Maps function-level dependencies
- `AttackSurfaceMapper` — Identifies public APIs, I/O, auth points
- `HistoryAnalyzer` — Reviews git history for past vulnerability patterns

---

### Stage 2: SCAN

**Purpose:** Find candidate vulnerabilities with evidence.

**What happens:**
1. Run specialized **auditor agents** for each vulnerability class
2. Each auditor examines the attack surface from its angle
3. Emit candidate findings with:
   - **Hypothesis** — "This code is vulnerable to [X] because [Y]"
   - **Evidence** — Code snippets, line references, call chain
   - **Reachability** — Can this actually be triggered from an entry point?
   - **Severity** — Critical/High/Medium/Low
   - **Confidence** — 0.0-1.0

**Auditor specializations (run in parallel):**
| Auditor | Looks For | Example Findings |
|---------|-----------|------------------|
| `SQLAuditor` | SQL injection, query construction | String concatenation into SQL |
| `XSSAuditor` | Cross-site scripting, HTML injection | Unescaped user input → HTML |
| `AuthAuditor` | Auth bypass, session management | Missing auth checks, token reuse |
| `InjectionAuditor` | Command injection, SSRF, LDAP injection | User input → system calls |
| `CryptoAuditor` | Hardcoded secrets, weak crypto, IV reuse | Static IV, md5 for passwords |
| `RaceAuditor` | Race conditions, TOCTOU | Shared state without locks |
| `MemoryAuditor` | Buffer overflow, use-after-free | Unsafe C calls, pointer arithmetic |
| `ConfigAuditor` | Misconfigurations, insecure defaults | Debug mode in production |

**Output:** A list of `CandidateFinding` objects with hypotheses, evidence, reachability, severity, confidence.

---

### Stage 3: VALIDATE

**Purpose:** Challenge each candidate — is it actually exploitable?

**What happens:**
1. Run **debater agents** on each candidate finding
2. One agent argues **FOR** exploitation (this is real)
3. One agent argues **AGAINST** exploitation (false positive)
4. Each must provide concrete reasoning with code references

**Debaters:**
- `ExploitationAdvocate` — "This CAN be exploited because..."
- `DefenseCounsel` — "This CANNOT be exploited because..."

**Resolution:**
- If `ExploitationAdvocate` confidence > `DefenseCounsel` confidence + threshold → Keep as `ValidatedFinding`
- Otherwise → Downgrade to `LowConfidence` or `Dismissed`

**Output:** `ValidatedFinding` list with debaters' analysis attached.

---

### Stage 4: DEDUP

**Purpose:** Collapse semantically equivalent findings.

**What happens:**
1. Group findings by root cause (same vulnerable function, same fix)
2. For each group, keep the highest-confidence finding
3. Merge evidence from all equivalent findings into one
4. Annotate with "also found in: [files]"

**Output:** Deduplicated `FinalFinding` list.

---

### Stage 5: PROVE

**Purpose:** Generate triggering inputs that demonstrate the vulnerability.

**What happens:**
1. For each `FinalFinding`, generate a concrete triggering input
2. For languages that support it (C/C++), use ASan/UBSan to prove memory safety violations
3. For interpreted languages (Python, JS), generate PoC scripts
4. Attach proof-of-concept code to the finding

**Output:** `ProvenFinding` list with PoC attachments.

---

## Final Output: Repository-Detective Report

```json
{
  "repository": "owner/repo",
  "commit": "abc123",
  "scan_time_ms": 45230,
  "stages_completed": ["prepare", "scan", "validate", "dedup", "prove"],
  
  "findings": [
    {
      "id": "SQL-001",
      "severity": "critical",
      "category": "sql_injection",
      "title": "SQL injection in user search",
      "confidence": 0.94,
      "file": "src/handlers/user.go",
      "line": 42,
      "description": "User-controlled input directly concatenated into SQL query",
      "evidence": {
        "code": "query := \"SELECT * FROM users WHERE id = \" + userID",
        "call_chain": ["handleSearch() → db.Query()"]
      },
      "attack_surface_reachability": "Reachable via /api/users/search endpoint",
      "debate_result": {
        "advocate_confidence": 0.94,
        "counsel_confidence": 0.12,
        "outcome": "validated"
      },
      "poc": "curl 'http://app/api/users/search?id=1%20OR%201=1'",
      "fix_suggestion": "Use parameterized queries: db.Query(\"SELECT * FROM users WHERE id = ?\", userID)"
    }
  ],
  
  "stats": {
    "files_analyzed": 156,
    "candidates_found": 23,
    "validated_findings": 8,
    "deduped_findings": 6,
    "proven_findings": 4
  }
}
```

---

## Original implementation plan (historical)

The checklist below was the initial roadmap. Most of Phase 1–3 is done; Phase 4 dashboard is not.

### Phase 1: Infrastructure
- [x] Refactor `analyzers/engine.go` to support multi-stage pipeline
- [x] Add `PrepareStage` — repo structure, attack surface mapping
- [x] Add stage result structs (PrepareReport, CandidateFinding, etc.)
- [x] Update `main.go` to wire up new pipeline

### Phase 2: Scanner Agents
- [x] Implement auditor agents (SQL, Auth, Injection, Config, XSS, Crypto)
- [x] LLM prompt templates for each auditor
- [x] Run auditors in parallel, collect candidates
- [x] Static pre-scan layer (`analyzers/static.go`)

### Phase 3: Validator + Dedup
- [x] Implement debater agents
- [x] Add deduplication logic (basic)
- [x] Add prove stage with PoC generation

### Phase 4: Integration + Polish
- [x] Integrate with Gitea issue creation
- [ ] Add configurable severity thresholds
- [ ] Build web dashboard for viewing reports
- [ ] Performance: add caching, incremental scan support

---

## Agent Prompts Reference

Each auditor agent prompt follows this structure:

```
You are a [SPECIALIZATION] security auditor.
Analyze the following code for [VULNERABILITY_CLASS].

REPOSITORY: {repo}
FILE: {file}
LANGUAGE: {lang}
ATTACK_SURFACE: {attack_surface}

CODE:
```{lang}
{code}
```

TASK:
1. Identify all instances of [VULNERABILITY_CLASS]
2. For each instance provide:
   - Exact location (file, line, function)
   - Evidence (the vulnerable code)
   - Call chain from entry point to vulnerable code
   - Whether it can be triggered from outside
   - Severity (critical/high/medium/low)
   - Confidence (0.0-1.0)

Respond with JSON array of findings.
```

Each debater prompt follows this structure:

```
You are a security vulnerability debater.

CLAIM: A [SEVERITY] [CATEGORY] vulnerability exists in {file}:{line}
CLAIM EVIDENCE: {evidence}

Your job:
- If you represent the DEFENSE: Argue why this is NOT exploitable
- If you represent the ADVOCATE: Argue why this IS exploitable

Analyze the code, the call chain, and the attack surface.
State your confidence (0.0-1.0) and provide reasoning.

Respond with: {verdict: "exploitable|not_exploitable", confidence: 0.0-1.0, reasoning: "..."}
```
