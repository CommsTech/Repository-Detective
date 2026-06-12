# SBOM test failure triage

**Date:** 2026-06-12  
**Commit fixed:** (see `test(sbom): resolve release validation assertion`)  
**Symptom:** `go test ./...` failed in `sbom_test.go:19` — expected `sbom_no_supported_manifest`, got `sbom_check_clean`.

## Root cause

`GenerateAndCheck` invoked **syft** on any directory when syft was in `PATH`, **before** checking for a supported dependency manifest.

On developer/CI hosts with syft installed, an **empty** `t.TempDir()` was scanned; syft produced an SBOM and grype (if present) returned zero vulnerabilities → `sbom_check_clean`.

This did **not** affect live PCAP_Analyser behavior (no manifest → correct empty state) when the pipeline uses manifest gating upstream; the bug was ordering in `sbom.GenerateAndCheck` only.

## Fix

Reorder logic in `sbom/sbom.go`:

1. `go.mod` → Go SBOM path (unchanged)
2. **No supported manifest** → `sbom_no_supported_manifest` (before syft)
3. syft when manifest exists
4. else `sbom_tool_missing`

Regression test: `TestEmptyWorkspaceWithSyftStillNoManifest` (skips if syft absent).

## Verification

```bash
go test ./sbom/... -count=1 -timeout=120s   # PASS
go test ./... -count=1 -timeout=300s         # PASS (all packages)
```

## Live impact

| Path | Affected? |
|------|-----------|
| Repo SBOM (manifest-less Python) | **No regression** — still honest empty state |
| Repo SBOM (go.mod / lockfiles) | unchanged |
| Container SBOM | separate path; unchanged |
