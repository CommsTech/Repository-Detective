# Calibration beta policy

## Beta requirements

1. Repo A calibration must not suppress repo B findings (`calibration/matcher.go` loads by `repositoryID`).
2. High-confidence security categories bypass graph-only downgrades unless operator approves suppression.
3. Expired or disabled suppressions stop applying immediately.
4. Global rules require evidence in calibration recommendations — no silent global suppress from single-repo dry-run.

## Not allowed in beta

- Overfitting netmapper/commsnet_optimizer dry-run noise into global defaults
- Hiding grype/trivy CVEs via calibration
- Auto-apply without `calibration_auto_apply` (stays false)

## Verification tests

- `go test ./calibration/...`
- `go test ./graph/...` (homelab calibration)
- Manual: two-repo DB with repo-specific suppression — only matching repo filtered
