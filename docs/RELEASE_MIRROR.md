# Public release mirror (GitHub tags & history)

Canonical releases live on **Gitea** (git tags + container packages).  
GitHub is a **sanitized snapshot** for discovery and public Releases.

## Rules

1. **Never force-push** GitHub `v*` release tags once published.
2. Refreshing `main` via `--github-snapshot` must **not** delete historical release tags.
3. A GitHub `vX.Y.Z` tag must represent a sanitized snapshot of the **same source tree** that produced the released container (or an explicitly documented limitation).
4. Release notes must include the **immutable OCI digest**.

## Creating a public GitHub release for an existing Gitea tag

```bash
# Example: v0.1.0-beta.3 (source commit e130bfb → image digest sha256:6a615548…)
./scripts/publish-github-release-snapshot.sh --tag v0.1.0-beta.3 --source-commit e130bfb
```

The script:

1. Builds an orphan sanitized tree from `--source-commit`
2. Creates annotated tag `vX.Y.Z` on that snapshot commit (**fails if tag already exists**)
3. Pushes the tag to GitHub without rewriting `main` (optional `--also-refresh-main`)
4. Creates a GitHub Release with notes pointing at acceptance + VERIFY_RELEASE

Sanitization note: the snapshot commit is built with `git archive` then `git add`, so
**`.gitignore` rules apply** (for example most of `docs/dogfood-reports/` stays out of the
public tag). That means the GitHub tag tree ID will not equal the raw Gitea
`e130bfb^{tree}` — it is the **public-sanitized** tree of that source commit.
The release notes still bind the immutable container digest to source commit `e130bfb`.

## Validation

```bash
./scripts/validate-github-release-tags.sh
```

Ensures release tags still resolve after a `main` snapshot refresh.

## beta.3 note

Gitea tag `v0.1.0-beta.3` → commit `e130bfb` (image source).  
Public GitHub release/tag should be created from a sanitized snapshot of that commit’s tree — not from a later `main` tip that only added docs.
