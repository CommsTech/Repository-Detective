# First impression checklist

Use before any external demo or marketing.

## Install

- [ ] Clean install from beta package or Docker all-in-one
- [ ] `.env.example` copied; no secrets in committed files
- [ ] `/health` returns healthy
- [ ] `/api/v1/about` shows product name

## Documentation

- [ ] Quick Start completes without developer assistance
- [ ] Scanner gaps explained when tools missing
- [ ] Container scanning marked opt-in / runner-required
- [ ] Wiki populated (blocked on Gitea server)

## Product scan quality

- [ ] Product repo: 0 active-present, 0 high/critical
- [ ] Reconciliation shows actionable vs informational split
- [ ] No duplicate Gitea issues from scan

## Demo safety

- [ ] Report-only mode understood
- [ ] Remediation PR disabled
- [ ] Runner delegation disabled unless demoing runner
- [ ] No credentials in screenshots or logs
