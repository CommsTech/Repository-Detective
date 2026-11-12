# Acceptance summary — v0.1.0-beta.3

**Proof labels:** `PUBLISHED_IMAGE_CLEAN_INSTALL_E2E_PROVEN`, `PUBLISHED_IMAGE_CORE_E2E_PROVEN`  
**Generated:** 2026-09-05 (Phase 6B)  
**Gitea tested:** **1.22.3** only (do not advertise a version range)

## Release identity

| Field | Value |
|-------|-------|
| Release version / tag | `v0.1.0-beta.3` |
| Source commit (in image) | `e130bfb` |
| Canonical tree tip at close | `a4d2a9a`+ (harness/docs commits after image bake) |
| Build timestamp (image) | `2026-09-04T22:54:45Z` |
| Go builder | `1.25` |
| Gitea registry digest | `sha256:6a615548c8a1fc2494140e73f1c3bd3f78f0ed54a7b15eaa7a1025e83e308727` |
| GHCR digest | `sha256:6a615548c8a1fc2494140e73f1c3bd3f78f0ed54a7b15eaa7a1025e83e308727` |
| Digest equivalence | **MATCH** (same digest on Gitea + GHCR; same local image ID) |

## Scanner inventory (published digest)

| Scanner | Version observed |
|---------|------------------|
| gitleaks | 8.21.2 |
| trivy | 0.68.2 |
| grype | 0.84.0 |
| semgrep | 1.76.0 |
| gosec | present (dev build string) |
| govulncheck | Go 1.25.12 toolchain |
| staticcheck | 2025.1.1 (0.6.1) |
| hadolint | 2.12.0 |
| checkov | 3.2.254 |

## Clean install (exact digest)

| Field | Value |
|-------|-------|
| Artifact | `e2e/results/clean-install-20260905T001022Z/` (local) |
| Result | **PUBLISHED_IMAGE_CLEAN_INSTALL_E2E_PROVEN** |
| Health | version `v0.1.0-beta.3`, commit `e130bfb`, build_date set |
| Onboard | HTTP 200 |
| Doctor | HTTP 200 |
| Empty storage | yes |
| Documented compose | yes |

## Core Gitea E2E (exact digest)

| Field | Value |
|-------|-------|
| Run | `e2e/results/20260905T012833Z-2799518/` (local) |
| Image | `@sha256:6a615548c8a1fc2494140e73f1c3bd3f78f0ed54a7b15eaa7a1025e83e308727` |
| Result | **PUBLISHED_IMAGE_CORE_E2E_PROVEN** |
| FAIL count | **0** |
| PARTIAL | secret_resolve_lifecycle (intentional) |

### Proof levels (capability)

| Capability | Level |
|------------|-------|
| Webhook registration / delivery | E2E_PROVEN |
| FIRST_SCAN_PROVEN | E2E_PROVEN |
| Secret / SAST / deps fixtures | E2E_PROVEN |
| PR summary idempotency | E2E_PROVEN |
| POLICY_MET | E2E_PROVEN |
| ACTION_REQUIRED | E2E_PROVEN |
| OBSERVATION_ONLY | E2E_PROVEN |
| EVALUATION_INCOMPLETE (fail-closed) | E2E_PROVEN |
| LOCAL_ONLY + Doctor + restart | E2E_PROVEN |
| Secret auto-resolve after fix | **PARTIAL** (by design — see FINDING_RESOLUTION_SEMANTICS.md) |
| Upgrade E2E | **NOT_PROVEN** |
| Class-B remediation | **NOT_PROVEN** (disabled by default; RD-008B Option C) |

No credentials, webhook secrets, or synthetic secret values are included in this document.
