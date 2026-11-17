# Deployment guide (operator)

**Repository Detective** — Inspect. Analyze. Improve.

Central index for deploying private beta instances. For fastest path see [QUICKSTART.md](QUICKSTART.md).

---

## Choose a path

| Path | Doc | When |
|------|-----|------|
| **Quick homelab** | [QUICKSTART.md](QUICKSTART.md) | Single host, all-in-one, Gitea |
| **Step-by-step setup** | [SETUP.md](SETUP.md) | First production-like install |
| **Docker details** | [DOCKER.md](DOCKER.md) | core / runner / all-in-one images |
| **Root deploy script** | [../DEPLOYMENT.md](../DEPLOYMENT.md) | `./deploy.sh`, DNS-filtered builds |
| **Networking / webhooks** | [NETWORKING.md](NETWORKING.md) | Gitea must reach your instance |
| **Upgrade** | [UPGRADE.md](UPGRADE.md) | Pull new image, migrations |

---

## Recommended private beta stack

```text
Image:     repository-detective:all-in-one
Compose:   docker-compose.yml (port 8081, bridge + publish; host-network overlay optional)
Database:  ./data/repository-detective.db (volume mount)
Config:    ./config/config.yaml + .env
Profile:   standard (deterministic; AI optional)
AI:        off by default (ENABLE_LLM_AUDITORS=false)
```

---

## Pre-deploy checklist

- [ ] `.env` copied from `.env.example`, secrets set
- [ ] `config/config.yaml` from example (beta defaults)
- [ ] Gitea token with repo + hook + issue permissions
- [ ] Public URL planned if Gitea is off-LAN
- [ ] `./scripts/release-test.sh` passed on build host (optional)

---

## Post-deploy validation

```bash
./scripts/operator-smoke-test.sh
```

Full checklist: [BETA_SMOKE_TEST.md](BETA_SMOKE_TEST.md)

---

## Known deployment issues

Host-specific notes: [DEPLOYMENT_ISSUES.md](DEPLOYMENT_ISSUES.md)

---

## Related

- [BACKUP_RESTORE.md](BACKUP_RESTORE.md)
- [TROUBLESHOOTING.md](TROUBLESHOOTING.md)
- [BETA_READINESS.md](BETA_READINESS.md)
