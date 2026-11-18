# Source audit remediation — 2026-11-01

Tracks the external static source audit (RD-AUD-001 … RD-AUD-010) against Gitea `main`.

## Shipped in this pass

| ID | Status | Notes |
|----|--------|-------|
| RD-AUD-002 | Done (code + compose + example env) | Reject published placeholder secrets/URLs at startup; loopback host bind by default; bootstrap hardened for public clients; API rejects example keys |
| RD-AUD-003 | Partial | Compose no longer force-skips startup checks; docs prefer `/health/ready`; example forge hostnames rejected. Full “setup mode without forge” still follow-up |
| RD-AUD-001 | Done (canonical docs) | README / QUICKSTART / SETUP / TEN_MINUTE_AHA use `git.commsnet.org` + `github.com/CommsTech` |
| RD-AUD-006 | Done | Removed `?api_key=` UI guidance; bootstrap/login path documented |
| RD-AUD-007 | Done (defaults) | `.env.example` / template default `AUTO_CREATE_ISSUES=false` |
| RD-AUD-009 | Partial | COMMUNITY_EDITION + KNOWN_LIMITATIONS aligned with enforced 10-repo limit and Go 1.26.10; Forgejo called experimental |
| RD-AUD-004 | Open | Still need immutable beta.6 digest + clean published-image acceptance receipt |
| RD-AUD-005 | Partial | Compose prefers published `RD_IMAGE`; `--build` demoted to contributor path. Synthetic no-credential demo still follow-up |
| RD-AUD-008 | Open | Pin gosec/@latest and golangci install script digests |
| RD-AUD-010 | Open | Independent first-user timing / UX study |

## Acceptance hooks added

- `internal/security.ValidateInstallSecrets` + unit tests
- `TestValidateAuthRejectsPublishedPlaceholders`
- `TestBootstrapRejectsPublicIPWithoutToken`

## Operator note

Existing secure deployments keep working. Homelab LAN dogfood should set `REPOSITORY_DETECTIVE_BIND=0.0.0.0` explicitly (gitignored `.env`) when overriding the loopback Compose default.
