# Release readiness

Checklist for deploying Repository Detective after the closeout sprint. Evidence paths are relative to repo root.

## Build and test

| Check | Command | Closeout evidence |
|-------|---------|-------------------|
| Unit tests | `docker run --rm -v $PWD:/src -w /src golang:1.23-bookworm go test ./... -count=1` | Run 2026-06-02 — all packages `ok` |
| Verify script | `./scripts/verify-all.sh` | Requires local Go + staticcheck |
| Docker health | `curl -sf http://<host>:8081/health` | `status: healthy` observed on local deployment |

## Functional smoke

| Area | Verify |
|------|--------|
| Dashboard | `/ui/` loads; charts or empty states; text summary present |
| Scanner health | `/ui/health` — configured/missing/optional distinct |
| Findings | `/ui/findings` queue loads |
| Reports | `/ui/reports` executive summary |
| API | `GET /api/v1/status` with API key |

## Documentation

- [x] [PRIVACY_AND_DATA_PROTECTION.md](PRIVACY_AND_DATA_PROTECTION.md)
- [x] [ACCESSIBILITY.md](ACCESSIBILITY.md)
- [x] [SCANNER_HEALTH.md](SCANNER_HEALTH.md)
- [x] [DASHBOARD_GUIDE.md](DASHBOARD_GUIDE.md)
- [x] [ADMIN_HARDENING.md](ADMIN_HARDENING.md)
- [x] [DATA_RETENTION.md](DATA_RETENTION.md)
- [x] [KNOWN_LIMITATIONS.md](KNOWN_LIMITATIONS.md)
- [x] [CHANGELOG.md](../CHANGELOG.md)

## Wiki

- Prepared: `docs/wiki/` — **not pushed** (no wiki remote on clone)
- See [WIKI_PUBLISHING.md](WIKI_PUBLISHING.md)

## Self-scan (dogfood)

- Guide: [DOGFOODING.md](DOGFOODING.md)
- Script: [scripts/dogfood-self-scan.sh](../scripts/dogfood-self-scan.sh)
- Closeout: health endpoint verified; full analyze requires API key + repo registration

## Sign-off

| Role | Status |
|------|--------|
| Engineering | Tests pass; closeout fixes merged locally |
| Security | Privacy-aware handling documented; not compliance certified |
| Accessibility | WCAG-aligned improvements; formal audit not run |
| Operations | Admin hardening + retention docs provided |
