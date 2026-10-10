# Automate Gitea wiki sync on release tags

**Labels:** `type/docs`, `type/feature`, `priority/p2`, `status/needs-triage`  
**Milestone:** Sprint 6 - Release Readiness

## Summary

Wiki copies exist under `docs/wiki/`; push is manual per `docs/WIKI_PUBLISHING.md`.

## Acceptance criteria

- [ ] CI job copies `docs/wiki/*.md` to wiki repo on tag only
- [ ] Uses deploy key; no force-push
- [ ] Release notes mention wiki pages updated

## Evidence

- `docs/WIKI_PUBLISHING.md` — manual steps documented
- No wiki remote in default clone
