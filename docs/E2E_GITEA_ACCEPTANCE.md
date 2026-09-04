# Gitea E2E acceptance (RD-017A / RD-018)

## Purpose

Prove the normal Repository Detective workflow against a **clean, disposable Gitea** instance — real HTTP, webhooks, repos, commits/PRs, persistence, and deterministic scanners.

Not a unit-test substitute. Remediation Class-B execution is **intentionally excluded** (RD-008B Option C).

## Topology

`docker-compose.e2e.yml`:

| Service | Image / role |
|---------|----------------|
| `gitea` | `gitea/gitea:1.22.3` (explicitly tested baseline — not a version range claim) |
| `repository-detective` | `RD_E2E_IMAGE` (default `repository-detective:all-in-one`) |

Credentials are generated/ephemeral. Nothing production is touched.

## Run

```bash
# Rebuild image with current tree (required for webhook delivery evidence code):
docker build -t repository-detective:all-in-one --target all-in-one \
  --build-arg VERSION=e2e --build-arg COMMIT=$(git rev-parse --short HEAD) .

./scripts/e2e-gitea-acceptance.sh
```

Artifacts: `e2e/results/<run-id>/acceptance.json` (+ logs, doctor JSON, comments).

Optional: `RD_E2E_KEEP_ON_FAIL=1` (default) retains containers on failure.

## Clean install (RD-018)

```bash
./scripts/e2e-clean-install.sh
```

Uses `git archive` of HEAD + `.env.example` + compose port **8081** + Doctor + scanner inventory.

Upgrade E2E: **NOT_PROVEN** until a prior public-beta baseline is selected.

## Proof IDs

| ID | Meaning |
|----|---------|
| `WEBHOOK_DELIVERY_E2E_PROVEN` | Validated forge webhook accepted and persisted in `operator_evidence` |
| `FIRST_SCAN_PROVEN` | Terminal production-path scan persisted (distinct from webhook delivery) |
| `WEBHOOK_SCAN_PROVEN` | First-scan proof where trigger was push/PR |

## Explicitly tested Gitea version

**1.22.3** only (this harness). Do not advertise a broader support range from this single baseline.
