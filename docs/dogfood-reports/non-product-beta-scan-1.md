# Non-product beta scan 1 — report-only

**Date:** 2026-06-10  
**Revision:** `rc-e3e19ec`

## Repo

| Field | Value |
|-------|-------|
| Repository | `commstech/PCAP_Analyser` |
| Repo ID | 13 |
| Scan ID | `1a4fc7a409f6d376` |
| Mode | `report_only_dry_run: true` |
| Profile | `beta_standard` |

## Results

| Metric | Value |
|--------|-------|
| Status | completed |
| Files analyzed | 8 |
| Issues found | 12 |
| High/critical | 0 reported in summary |
| Issues created | **0** (dry-run) |
| Duration | ~4.3s |

## Noise estimate

Small repo; 12 findings on 8 files — moderate density for a tiny Python/tooling repo. Likely acceptable for beta with calibration.

## Actionability

Findings detail template now live on RC — operator can triage without raw-only pages.

## Operator impression

**Acceptable for private beta** with calibration; not marketing-demo polished yet.

## Recommended before marketing

- Spot-check 3 findings for false-positive rate
- Confirm graph/SBOM panels on this scan if applicable
