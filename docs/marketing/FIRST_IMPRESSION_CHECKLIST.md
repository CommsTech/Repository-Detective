# First impression checklist

Use before any external demo or marketing.

## Install

- [ ] Clean install from beta package or Docker all-in-one — **not tested**
- [x] `.env.example` copied; no secrets in committed files — **ready**
- [x] `/health` returns healthy — **ready**
- [x] `/api/v1/about` shows product name — **ready**

## Documentation

- [ ] Quick Start completes without developer assistance — **not tested**
- [ ] Scanner gaps explained when tools missing — **partially ready**
- [x] Container scanning marked opt-in / runner-required — **ready**
- [ ] Wiki populated — **blocked** (Gitea HTTP 500)

## Product scan quality

- [x] Product repo: 0 active-present, 0 high/critical — **ready**
- [x] Reconciliation shows actionable vs informational split — **ready**
- [x] No duplicate Gitea issues from scan — **ready**

## Demo safety

- [x] Report-only mode understood — **ready**
- [x] Remediation PR disabled — **ready**
- [x] Runner delegation disabled unless demoing runner — **ready** (rolled back post-demo)
- [x] No credentials in screenshots or logs — **ready**

## Container scan demo (2026-06-10)

- [x] One controlled `alpine:3.20` registry scan on labeled runner — **ready**
- [x] Syft/Trivy/Grype coverage reported — **ready**
- [x] No issues/PRs created — **ready**
