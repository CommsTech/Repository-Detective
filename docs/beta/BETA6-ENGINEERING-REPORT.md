# Repository Detective — Beta 6 Engineering Report (2026-10-11)

**Release candidate:** branch `beta6-security-hardening` @ `13c6215` (canonical base `6d9f620d`)
**Binary:** `v0.1.0-beta.6-rc.1`, SHA-256 `2fdef5dbd34c9f5542e2c7566aac64c844a1bccd28cd6faa468608cc69165617`
**Classification:** **BETA — NOT for public release yet.** Three release blockers below are handled or documented; the decision to tag, push, and deploy sits with the owner.
**Worktree:** `/home/commstech/.openclaw/workspace/tmp/rd-beta6-p1` (isolated clone of git.commsnet.org canonical; production checkout on ClusterManager untouched)

---

## 1. Ground truth (evidence)

- Canonical repo = Gitea `git.commsnet.org/commstech/repository-detective`, `main` @ `6d9f620d`. GitHub mirror has a **byte-identical tree** (`5282f36a…`) but synthetic history (`58c7cec`, "Public mirror snapshot of Gitea main@6d9f620d"). Gitea remains authoritative. Verified via `git rev-parse`/`rev-list --left-right --count github/main...origin/main` = `1 536` with equal tree hashes.
- Production runtime: container `repository-detective` on image `repository-detective:v0.1.0-beta.5`, healthy, DB 2.37 GB WAL mode. **Note:** the container was redeployed at 13:48–13:55 UTC on 2026-10-11 (commit `6d9f620d`) by an agent/process outside this session (the concurrent Cursor task PID 1791977 working the dirty production checkout). Cause not attributable; logged as an observation, no action taken.

## 2. Live incident repaired today (production data, verified)

**Root cause of "repeated issues" on repo 53 (commstech/wiki):** commit `ce74a73d` (2026-08-02) renamed fingerprint prefix `bugbot-` → `rd-` without any DB migration. From Aug 3 every scan re-detected existing findings as brand-new fingerprints; issue filing re-opened duplicates (wiki issues #23–#32 vs the June originals #1–#18), and all learning/calibration rows written under the old prefix stopped matching.

Repaired (single transaction, pre/post integrity + FK-checked, backup `~/rd-backup-prefix-migration-20261011.db` on ClusterManager, restore-tested to tmpfs: integrity ok, 28.9 s):

- Merged **9,015** duplicate finding rows into their originals; renamed **15,938** legacy fingerprints to `rd-`; repointed **477,537** finding-instances, **339** issue links, **9,338** lifecycle events, **6,692** learning events, **13,588** suppression rules; folded `last_seen` forward; deleted the 9,015 twins.
- Post-check: `bugbot-` rows = 0; repo 53 duplicate (rule, file, line) groups = 0; `integrity_check` ok; FK violations unchanged from baseline (see §5 blockers).
- Reconciliation preview now correctly maps every refiled issue to its canonical twin (e.g. #24→#1, #25→#23, #30→#20); the wiki's 7 proven duplicate issues (#24–#30) were closed with a canonical pointer + comment and `repository-detective/duplicate` label via the Gitea API, with local mapping synced after verified success. Open wiki issues: 93 (was 100).
- Service remained healthy throughout; live auth matrix and security headers verified post-restart (no_key=401, bogus=401, valid=200, headers CSP/nosniff/DENY/no-referrer present).

## 3. Beta 6 P0 — implemented, tested (commits `b0d9dc6`, `13c6215`)

1. **Durable cross-process issue-filing idempotency** (`issues/manager.go`, `store/issue_filing_claims.go`, schema v26 `issue_filing_claims`): one filing slot per (repository, forge, fingerprint); a filed claim carries the remote issue number so a crash after remote-create recovers without duplicates; losers converge via remote search or the filed row instead of double-filing. In-process mutex preserved underneath.
2. **Startup route/readiness contract** (`main.go`): all routes — including the control plane — are registered **before** `ListenAndServe` (the old code mutated the Gin radix tree while serving, a data race); blocking forge/AI checks moved after the listener; **`/health/ready` added** (503 until ready, 200 after; `/health/live` unchanged and immediate).
3. **Gitea label verification** (`gitea/issue_labels.go`): `AddIssueLabels` now reads back the issue's labels and fails closed unless every requested label is observed; missing names are ensured via `ResolveLabelIDs` (creates if absent) and retried once. This kills the silent HTTP-200 no-op that made reconciliation report success while Gitea labels never changed (reproduced live).

Tests added: 4-manager cross-process convergence (exactly 1 create), filed-claim crash recovery (0 creates), silent-label-no-op → ensure+verify, unresolvable-label → error, startup readiness/concurrent probing (incl. `-race`).

## 4. Beta 6 P1 — implemented, tested (commit `d2576b7`)

1. **Webhook replay protection** (`store/webhook_deliveries.go`, schema v26 `webhook_deliveries`): signed deliveries deduped by (forge, delivery-ID, repository, event kind), durable across restarts; duplicate → 200 "duplicate" without dispatch; claim failure → **503 fail-closed**; deliveries without a forge delivery-ID header are never claimed (documented: dispatch proceeds). Bounded retention 7d / 100k rows, hourly pruning. Default ON.
2. **Query-string API keys rejected by default** (`reject_query_string_api_key=true`): live-verified `…?api_key=…` → 401. Operators with legacy integrations can set the env override explicitly.
3. **Trusted proxies default to none** (IP/CIDR allowlist validation via `normalizeTrustedProxies`): X-Forwarded-For no longer trusted implicitly; `ClientIP` pins the direct peer unless an operator allowlists a proxy.
4. AI prompt-injection posture verified already sound (untrusted-content wrapping + guardrail tests); session cookies HttpOnly/SameSite=Lax/signed + SQLite-backed sessions + CSRF confirmed; webhook HMAC + body limits + per-IP rate limiting confirmed.

## 5. Validation evidence

- Full unit suite on RC: **43 packages ok / 0 FAIL** (`/tmp/rc-full-suite.log`).
- Full `-race` suite: **43 ok / 0 FAIL**, no data races (`/tmp/rc-race-suite.log`).
- `go vet ./...` clean; `gofmt -s` clean (2 files fixed in `13c6215`).
- `govulncheck ./...`: **0 vulnerabilities in code** (2 in un-called module requirements — noted, non-blocking).
- `scripts/check-public-release-secrets.sh`: PASS. `scripts/validate-github-release-tags.sh`: `v0.1.0-beta.2` tag **missing on GitHub mirror** (pre-existing; mirrors are snapshots).
- Live RC smoke (ephemeral instance on :18099, then cleaned up): `/health/live` 200 immediately; `/health/ready` truthful; auth matrix `no_key=401 bogus=401 valid=200 query=401`; version reported `v0.1.0-beta.6-rc.1`.
- Baseline pre-change (canonical HEAD): `go test ./...` and `go test -race ./...` both 100% pass — the failures we fixed were behavioral gaps, not test debt.

## 6. Release blockers / NOT_PROVEN (owner decisions needed)

1. **60,611 pre-existing FK violations** in the production DB (`lifecycle_events→scans` 60,606; `scanner_results→scans` 3; `runner_jobs→scans` 2; `integrity_check` ok). Canonical `OpenSQLite` never enables `PRAGMA foreign_keys=ON`. RC flags this; enabling enforcement now would break scan-lifecycle writes until a non-destructive cleanup/archival migration is written. **Recommend:** dedicated CMB change (archive orphans, then enforce FK for new writes) before public release.
2. **Single SQLite connection** (`SetMaxOpenConns(1)`): explains slow multi-second CLI/RPC under load (20-way read benchmark p50 59 ms, p95 74 ms vs 0.8 ms solo). WAL allows concurrent readers; moving to a small read pool + single writer is the P2 candidate. Not changed in RC (needs perf work + soak).
3. **Deploy/push not done:** RC is local-only (branch + binary above). Tagging `v0.1.0-beta.6`, pushing to Gitea, container publish, and the production redeploy are owner-authorized steps (CMB applies per workflow). The **production checkout on ClusterManager is dirty with another agent's uncommitted work** (`forge/client.go`, `main.go`, `learning/*`, tip-commit skip feature) — do not deploy over it until reconciled; the concurrent Cursor agent (PID 1791977, running since Oct 8) is still active there.
4. Root disk on ClusterManager at 95% (12 GB free) and swap previously full — the backup + migration made this worse temporarily; a nightly backup job and disk review remain open CMB items from the overnight audit.
5. Historical gitleaks timeouts on wiki history scans and the `REL-INTERNAL-INFRA-REF` noise (internal infra links in a private wiki are expected, not defects) — candidates for calibrated `report_only` rules, already partly learned by RD itself.

## 7. Process note

Four delegated engineer runs (2× Cursor broker, 2× software-engineer subagent) all failed with empty results (TLS failures / "no usable profiles" for `openai/gpt-5.6-sol`). Their worktrees contained partial, uncompilable drafts; I completed and tested both phases inline rather than re-spawning a fifth time. Delegation tooling for explicit OpenAI model overrides needs attention separately.