# Beta end-to-end smoke test

**Product:** Repository Detective — Inspect. Analyze. Improve.  
**Audience:** Operator validating a private beta deployment  
**Time:** ~60–90 minutes first run

Use generic placeholders only. Do not commit `.env` or real tokens.

Prerequisites: [QUICKSTART.md](QUICKSTART.md), [BETA_READINESS.md](BETA_READINESS.md) beta config.

---

## Automated pre-checks

```bash
./scripts/release-test.sh
export REPOSITORY_DETECTIVE_API_KEY='your-key'
export RD_BASE_URL='http://127.0.0.1:8081'
./scripts/operator-smoke-test.sh
```

| Step | Pass | Fail notes |
|------|:----:|------------|
| release-test.sh exits 0 | ☐ | |
| operator-smoke-test.sh exits 0 | ☐ | |
| `/health` status healthy | ☐ | |
| 10/10 scanners available (all-in-one) | ☐ | |

---

## Manual operator flow

### 1. Deploy all-in-one

```bash
git clone https://git.example.com/your-org/repository-detective.git
cd repository-detective
cp .env.example .env
# Edit: REPOSITORY_DETECTIVE_API_KEY, GITEA_*, WEBHOOK_SECRET
docker compose up -d --build
```

| Pass | Fail notes |
|:----:|------------|
| ☐ | |

### 2. Confirm health

```bash
curl -s http://127.0.0.1:8081/health | jq .
```

Expect `"status":"healthy"`, `"ready":true`, `tools_summary.available_count` ≥ 8.

| Pass | Fail notes |
|:----:|------------|
| ☐ | |

### 3. Confirm scanner availability

```bash
curl -s -H "X-Repository-Detective-API-Key: $REPOSITORY_DETECTIVE_API_KEY" \
  http://127.0.0.1:8081/api/v1/status | jq .
```

| Pass | Fail notes |
|:----:|------------|
| ☐ | |

### 4. Connect one Gitea repo

- Open `http://127.0.0.1:8081/onboard` (or `/ui` with API key)
- Enter Gitea URL + token + public URL
- Register webhook on **one** test repository

| Pass | Fail notes |
|:----:|------------|
| ☐ | |

### 5. Run manual scan

```bash
curl -s -X POST http://127.0.0.1:8081/api/v1/analyze \
  -H "X-Repository-Detective-API-Key: $REPOSITORY_DETECTIVE_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{"owner":"your-org","repository":"your-repo","ref":"main"}'
```

Or trigger from UI repo detail → Scan.

| Pass | Fail notes |
|:----:|------------|
| ☐ | |

### 6. Review findings

- Open `/ui` → Findings
- Confirm non-zero score if findings exist
- Open one finding detail — no raw secrets in evidence

| Pass | Fail notes |
|:----:|------------|
| ☐ | |

### 7. Suppress one known false positive

- Repo settings or finding detail → suppression rule
- Re-run scan or reconciliation preview — finding suppressed

| Pass | Fail notes |
|:----:|------------|
| ☐ | |

### 8. Generate remediation plan (if eligible finding exists)

- Finding detail → remediation plan (planner enabled; PR creation may stay off)

| Pass | Fail notes |
|:----:|------------|
| ☐ | N/A if no eligible finding |

### 9. Verify evidence closure behavior

- Confirm `evidence_closure_enabled: true`, `evidence_closure_close_issues: false`
- Closure adds comments/verification — does **not** auto-close Gitea issues

| Pass | Fail notes |
|:----:|------------|
| ☐ | |

### 10. Reconciliation preview

```bash
curl -s -X POST "http://127.0.0.1:8081/api/v1/repos/1/reconcile/preview" \
  -H "X-Repository-Detective-API-Key: $REPOSITORY_DETECTIVE_API_KEY"
```

(Replace repo ID.) Or use UI reconcile page.

| Pass | Fail notes |
|:----:|------------|
| ☐ | |

### 11. Pre-install audit (safe test repo)

Enable temporarily if needed: `preinstall_audit_enabled: true` in config.

Use a **small public HTTPS repo you control** (not production secrets).

- Run audit from UI/API
- Review shareable vs private disclosure drafts

| Pass | Fail notes |
|:----:|------------|
| ☐ | N/A if disabled for beta |

### 12. Confirm reports contain no secrets

- Search report output for `gitea_token`, `api_key`, `sk-`, internal hostnames
- Shareable report must use placeholders only

| Pass | Fail notes |
|:----:|------------|
| ☐ | |

### 13. Backup DB

```bash
docker compose stop repository-detective
cp data/repository-detective.db "data/repository-detective-backup-$(date +%F).db"
docker compose start repository-detective
```

See [BACKUP_RESTORE.md](BACKUP_RESTORE.md).

| Pass | Fail notes |
|:----:|------------|
| ☐ | |

### 14. Restore DB

```bash
docker compose stop repository-detective
cp data/repository-detective-backup-YYYY-MM-DD.db data/repository-detective.db
docker compose start repository-detective
```

| Pass | Fail notes |
|:----:|------------|
| ☐ | |

### 15. Confirm data survives restore

- Dashboard repo/finding counts match pre-backup
- Suppression rules still present

| Pass | Fail notes |
|:----:|------------|
| ☐ | |

---

## Sign-off

| Criterion | Pass |
|-----------|:----:|
| All automated pre-checks pass | ☐ |
| Manual steps 1–6 pass (minimum viable beta) | ☐ |
| Steps 7–15 pass or documented N/A | ☐ |
| Operator can repeat without developer help | ☐ |

**Operator name / date:** _______________

**Recommendation:** ☐ Go for private beta · ☐ Blockers remain (list in fail notes)

Blockers → [TROUBLESHOOTING.md](TROUBLESHOOTING.md) · [KNOWN_LIMITATIONS.md](KNOWN_LIMITATIONS.md)
