# Live structured issue body deploy

**Date:** 2026-06-12  
**Image:** `repository-detective:rc-381667a`  
**Git commits:** `381667a` (SBOM + structured issue body), `d5548f1` (Dockerfile vendor fix for future builds)  
**Prior live:** `rc-e3e19ec`

## Deploy method

Full image rebuild — **no binary hot-swap**.

```bash
export RD_VERSION=rc-381667a RD_COMMIT=381667a
docker-compose -f docker-compose.yml build \
  --build-arg VERSION="$RD_VERSION" --build-arg COMMIT="$RD_COMMIT" \
  --build-arg INSTALL_EXTERNAL_TOOLS=true
docker tag repository-detective:all-in-one repository-detective:rc-381667a
docker stop repository-detective && docker rm repository-detective
docker run -d --name repository-detective --network host --restart unless-stopped \
  --env-file .env \
  -v ./config:/app/config:ro -v ./data:/app/data -v ./certs:/app/certs:ro \
  repository-detective:rc-381667a
```

## Post-deploy verification

| Check | Result |
|-------|--------|
| `/health` status | **healthy** |
| `/health` version | **`rc-381667a`** |
| `/health` ready | **true** |
| Product `active_present_open` | **0** |
| Product `forge_open_issues` | **0** |
| Gitea issue templates API | **15 templates** |
| Structured issue body (new create) | **PASS** — issue #2 on scratch repo |

### Structured body proof (scratch repo only)

Controlled filing on `commstech/rd-filing-scratch`:

- Scan `a443edea1c4db9e2` created issue **#2**
- Body contains: `## Finding`, `## Issue filing policy`, `## False-positive guidance`, `## Repository Detective metadata`
- Does **not** use legacy `## Finding Type` heading
- Issue **#2 closed**; filing disabled on scratch repo afterward

## Not run

- No product-repo scan
- No all-repo scan
- Beta tester repos remain report-only

## Acceptance

**Structured issue body live:** YES  
**Live revision:** `rc-381667a`
