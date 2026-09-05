# External clean install report

**Date:** 2026-06-11

## Status: partial PASS

Full isolated VM not available; clean-path simulation on same host.

## Test path

| Step | Result |
|------|--------|
| Fresh tree copy (exclude `data/`, `.git/`, `dist/`) | **PASS** |
| `cp .env.example .env` | **PASS** |
| `make beta-release` (2026-06-11) | **PASS** — `dist/repository-detective-beta/` |
| Package: `docker-compose.beta.yml`, `config.example.yaml`, `.env.example` | **PASS** |
| `check-beta-package-secrets.sh` (prior sprint) | **PASS** |
| `docker build --target all-in-one` on clean copy | **in progress / prior builds queued** |
| Start container on alternate port + Gitea connect | **deferred** (uses homelab Gitea creds) |

## Documented install path

1. Clone `https://git.commsnet.org/commstech/Repository-Detective.git`
2. `cp .env.example .env` — set secrets locally only
3. `docker build --target all-in-one -t repository-detective:all-in-one .`
4. `docker compose -f docker-compose.beta.yml up -d`
5. `curl http://127.0.0.1:8081/health`
6. Connect Gitea, run one report-only scan

## Acceptance

| Item | Status |
|------|--------|
| Beta package builds from clean tree | **yes** |
| Full VM cold install | **not proven** (docker build queued on shared host) |

## Next action

Run on dedicated clean VM when docker builder queue clears; follow `docs/SETUP.md`.
