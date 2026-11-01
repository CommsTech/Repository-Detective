# Project groups — beta scope

## In scope

- Create/list project groups
- Assign multiple repository IDs + optional primary repo
- Navigation link from Configure page
- Repo-level calibration remains isolated

## Out of scope

- Merged cross-repo findings queue
- Group-level scheduled scan job
- Shared suppression policy auto-sync

## Tests

- `TestProjectGroupCRUD` in `store/projects_test.go`
- UI smoke: `/ui/projects` returns 200

## Release impact

Medium — improves multi-repo homelab/apps; not required for single-repo beta.
