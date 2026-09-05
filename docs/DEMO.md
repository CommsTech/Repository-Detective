# Demo walkthrough (disposable)

**Audience:** first-time operators  
**Requirement:** a **disposable** Gitea + Repository Detective — never production forges or private repos.

Use synthetic names such as `demo/repository-detective-test`. Never capture real secrets, private hostnames, or live vulnerability data from customer code.

## Goal

See the full path:

1. Install / start Repository Detective  
2. Connect disposable Gitea  
3. Select a repository  
4. Use **Observe** mode (`monitor_only`)  
5. Push a **harmless synthetic** finding fixture  
6. Webhook delivery  
7. Deterministic scanner identifies it  
8. Canonical issue appears  
9. Policy result appears  
10. PR summary links to findings  
11. Fix the fixture  
12. Understand reconciliation (no naive auto-close on partial scans)

## Prerequisites

- Docker Compose  
- Image pin: `v0.1.0-beta.3` (or digest in [ACCEPTANCE_v0.1.0-beta.3.md](release/ACCEPTANCE_v0.1.0-beta.3.md))  
- Empty storage volume  

Optional disposable topology: `docker-compose.e2e.yml` + [E2E_GITEA_ACCEPTANCE.md](E2E_GITEA_ACCEPTANCE.md).

## 1. Start Repository Detective

```bash
cp .env.example .env
# set REPOSITORY_DETECTIVE_API_KEY
# point GITEA_* at your disposable forge
docker compose pull && docker compose up -d
curl -s http://127.0.0.1:8081/health
```

Open `/onboard` and complete Connect → Select → Protect → Verify.

## 2. Observe mode

On the repo settings (UI or `PUT /api/v1/repos/:id/settings`), set:

- `policy_level`: `monitor_only` (Observe)

Findings should be recorded without Repository Detective enforcing a blocking gate.

## 3. Synthetic finding fixture

Use the same **synthetic Slack-bot-shaped** fixture family proven in E2E — **not a real credential**. Build at runtime so static scanners do not treat docs as secrets:

```bash
# In a demo clone of demo/repository-detective-test
python3 - <<'PY'
prefix = "xoxb-"
mid = "123456789012-123456789012-"
suffix = "zzdemofixturetokzz"
open("demo_leak.go", "w").write(
    "package main\n// synthetic demo fixture — not a real credential\nvar demo = %r\n" % (prefix + mid + suffix)
)
PY
git add demo_leak.go && git commit -m "demo: synthetic secret fixture" && git push
```

## 4. What you should see

| Step | Expect |
|------|--------|
| Webhook | Doctor / proofs show delivery (when configured) |
| Scan | gitleaks (or configured required scanners) runs |
| Issue | One canonical Gitea issue with fingerprint; secret redacted in bodies |
| Policy | In Observe mode: `OBSERVATION_ONLY` on PR summary |
| PR | Single compact summary comment (upsert), not one comment per finding |

Screenshots: [assets/screenshots/README.md](assets/screenshots/README.md).

## 5. Fix and reconcile

Remove the fixture, push again.

**Do not expect** immediate auto-close from a changed-files webhook alone.  
Resolution requires sufficient scope / reconcile or evidence-closure — see [FINDING_RESOLUTION_SEMANTICS.md](FINDING_RESOLUTION_SEMANTICS.md).

Preview: `GET /api/v1/repos/:id/reconcile-issues/preview`  
Apply (when configured): `POST /api/v1/repos/:id/reconcile-issues`

## 6. Cleanup

Tear down disposable compose volumes. Never leave demo tokens in a shared forge.

## Related

- [PUBLIC_BETA.md](PUBLIC_BETA.md)  
- [QUICKSTART.md](QUICKSTART.md)  
- [POLICY.md](POLICY.md)  
- [VERIFY_RELEASE.md](VERIFY_RELEASE.md)
