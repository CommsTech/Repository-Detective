# Batch 3c — gosec medium queue

Scan baseline: `a8bb4cddd72ab80c`

| Issue | Fingerprint | File | Rule | Fix |
|------:|-------------|------|------|-----|
| #301 | rd-7ac01e367871e750 | graph/workspace.go | G304 | ValidateWorkspacePath before ReadFile |
| #302 | rd-79390126afc631c8 | health/runner.go | G304 | ValidateWorkspacePath before ReadFile |
| #303 | rd-0bfd3f45ed94cd4f | patcher/exec.go | G204 | Allowlisted fixedCommand switch |
| #304 | rd-574f04844f6b7c09 | patcher/rules_hadolint.go | G306 | WriteFile 0o600 |
| #305 | rd-c083aa29ba214d98 | patcher/rules_hadolint.go | G304 | patchWorkspaceFile helper |
| #306 | rd-3b76c829b26f7da8 | patcher/rules_hadolint.go | G306 | WriteFile 0o600 |
| #307 | rd-0b5f428d1bf01004 | patcher/rules_hadolint.go | G304 | patchWorkspaceFile helper |
| #308 | rd-1804c62e3dd3af05 | patcher/rules_staticcheck.go | G304 | patchWorkspaceFile helper |
| #309 | rd-4ab00b3648f8de39 | patcher/rules_staticcheck.go | G306 | WriteFile 0o600 |
| #310 | rd-99bce8170166fbc0 | preinstall/checks.go | G304 | ValidateWorkspacePath + join |
| #311 | rd-b33665daa3c49eab | preinstall/checks.go | G304 | ValidateWorkspacePath + join |
| #312 | rd-26db68286d6589ad | preinstall/checks.go | G304 | checkWorkflow validated read |
| #313 | rd-c0660264888f3f7a | runner/executor.go | G304 | ValidateWorkspacePath before ReadFile |
| #318 | rd-6068dc4d0ec4c3e9 | scanners/gitleaks.go | G304 | nosec on CreateTemp report path |
| #319 | rd-db7ecda12786c683 | scanners/workspace.go | G301 | MkdirAll 0o750, WriteFile 0o600 |

Tests: `go test ./...`, `go vet ./...`, operator smoke test after deploy.
