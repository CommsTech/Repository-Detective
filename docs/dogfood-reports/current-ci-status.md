# Current CI status

## Latest runs (pre-push)

| Run | Commit | Result | Blocker |
|-----|--------|--------|---------|
| 1842 | 9d875fd | **stuck in_progress** | Runner job completion lag; dependent jobs waiting |
| 1841 | cdce47c | failure | Artifact upload (Gitea unsupported) — code passed |
| 1840 | — | failure | Runner checkout flake |
| 1839 | 962839e | failure | gofmt changed files |

## Fixes in this stabilization push

### ci.yml
- Single `CI` job: format → vet → staticcheck → test → build → govulncheck → docker
- `timeout-minutes: 60`
- `workflow_dispatch` for manual rerun
- Removed multi-job `needs:` dependency

### release.yml
- Single `Release` job: test → build binaries → docker → Gitea release upload
- Removed `upload-artifact@v4` / `download-artifact@v4`

## Local verification

```bash
go test ./...
go vet ./...
# staticcheck via: go install honnef.co/go/tools/cmd/staticcheck@v0.6.1 && staticcheck ./...
```

All passed in Docker `golang:1.23` (2026-06-06).

## Batch 2 gate

**Blocked** until first green run on `main` after push.

## URL

https://git.commsnet.org/commstech/Repository-Detective/actions/runs/1842 (superseded after push)
