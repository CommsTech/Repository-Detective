# Project groups

A **Project Group** ties multiple repositories into one application context for reporting and shared policy hints.

## Model

```text
Project Group
  name, description
  repositories[] 
  primary_repository_id
  shared calibration policy (future — beta: isolated per repo unless operator enables)
```

## Beta API / UI

- DB: `project_groups`, `project_group_repositories` (migration 19)
- UI: `/ui/projects` — list and create groups
- API: form POST create (CSRF protected)

## Use cases

- Main app + plugins
- Frontend + backend
- App + infra repo
- Library + integrations

## Out of scope for beta

- Cross-repo automatic issue filing
- Automatic grouping without operator action
- Cross-repo scan orchestration as single job

See: `docs/beta/PROJECT_GROUPS_BETA_SCOPE.md`
