# Findings detail live verification

**Date:** 2026-06-10  
**Live URL tested:** `/ui/findings/37361`  
**Revision:** `rc-e3e19ec`

## Finding 37361 (commstech/Repository-Detective)

| Section | Renders |
|---------|---------|
| Finding summary | YES |
| Risk and impact | YES |
| Location | YES |
| Why this was flagged | YES |
| Scanner rule details | YES (rule `REL-INTERNAL-INFRA-REF`) |
| Recommended fix | YES |
| Verification steps | YES |
| Issue filing status | YES |
| False positive / calibration | YES |
| Evidence (redacted) | YES |
| History | YES |
| Raw JSON / debug details | YES |
| Remediation plan | YES (existing) |

## Safety checks

| Check | Result |
|-------|--------|
| Raw secrets in UI | not observed |
| Calibration requires reason | form requires reason field |
| Issue link when mapped | shown when present |

## Additional findings tested

| Finding type | Status |
|--------------|--------|
| Generic health/static (37361) | verified live |
| Graph finding | route 200 (not individually captured this pass) |
| Container/SBOM finding | not selected this pass |
| Historical secret fixture | not selected (no safe test ID confirmed) |

## Operator impression

Page is **materially more actionable** than pre-RC template: engineer can see what/why/where/fix/calibration without reading raw scanner JSON first.

## Screenshots

Not captured in this automated pass — recommend manual screenshot batch before marketing.

## Acceptance

**Findings detail live:** PASS (37361)  
**Finding 37361 verified:** YES
