# Feature Request: Integrate prbl-scanner for AI Code Security

## Summary
Add prbl-scanner as a deterministic scanner in Repository Detective's security analysis pipeline to catch AI-specific code vulnerabilities that traditional scanners miss.

## Source
- **prbl-scanner:** https://github.com/noreplywmsplaybook-pixel/prbl-scanner
- **Requested by:** Alan Rakestraw (2026-06-16)
- **Priority:** Medium-High (security + AI coding workflow improvement)

## What prbl-scanner Detects

| Rule ID | Vulnerability | CWE | OWASP | AI Blind Spot |
|---------|---------------|-----|-------|---------------|
| PRBL-C001 | Hardcoded secrets with fallbacks | CWE-798 | A07 | `process.env.SECRET \|\| 'default'` patterns |
| PRBL-R001 | Weak randomness in security contexts | CWE-338 | A04 | `Math.random()` for tokens/session IDs |
| PRBL-R002 | Insecure equality comparison | CWE-208 | A02 | `==` vs constant-time compare for HMAC/signatures |
| PRBL-R003 | AES-GCM missing auth tag length | CWE-345 | A02 | Missing `setAuthTagLength()` calls |
| PRBL-I001 | SQL injection (multi-line patterns) | CWE-89 | A05 | Concatenated/interpolated queries |
| PRBL-I002 | Command injection | CWE-78 | A05 | `shell=True`, unsafe subprocess |
| PRBL-I003 | Code injection | CWE-94 | A05 | `eval()`, `exec()`, `new Function()` |
| PRBL-A002 | JWT bypass | CWE-347 | A02 | `jwt.decode()` without `verify()` |
| PRBL-A001 | Missing access control | CWE-862 | A01 | Route handlers without auth checks |

## Integration Approach

### Option 1: Direct Integration (Recommended)
Add prbl-scanner rules to Repository Detective's deterministic scanner registry:

**Files to modify:**
- `scanners/deterministic.go` — Register "prbl-scanner" as deterministic source
- `scanners/prbl.go` — New scanner wrapper (similar to `gosec.go`, `semgrep.go`)
- `analyzers/engine.go` — Wire prbl findings into CAH harness
- `docs/SCANNERS.md` — Document new scanner

**Scanner wrapper example:**
```go
package scanners

import (
    "context"
    "os/exec"
    "path/filepath"
)

type PrblScanner struct {
    binary string
}

func NewPrblScanner() (*PrblScanner, error) {
    binary, err := exec.LookPath("prbl-scanner")
    if err != nil {
        return nil, fmt.Errorf("prbl-scanner not found in PATH: %w", err)
    }
    return &PrblScanner{binary: binary}, nil
}

func (s *PrblScanner) Scan(ctx context.Context, workspaceRoot string) ([]Finding, error) {
    cmd := exec.CommandContext(ctx, s.binary, "scan", workspaceRoot)
    output, err := cmd.Output()
    if err != nil {
        return nil, fmt.Errorf("prbl scan failed: %w", err)
    }
    
    // Parse JSON output into Finding structs
    // Map PRBL-* rule IDs to Repository Detective finding schema
    // Return findings for CAH processing
}

func init() {
    RegisterDeterministicSource("prbl-scanner")
}
```

### Option 2: Submodule + CI Integration
Add prbl-scanner as git submodule, run in CI pipeline before Repository Detective analysis:
```bash
git submodule add https://github.com/noreplywmsplaybook-pixel/prbl-scanner.git scanners/prbl-scanner
# Run in .gitea/workflows/security.yml
- name: Run prbl-scanner
  run: cd scanners/prbl-scanner && go run ./cmd/prbl-scanner scan $WORKSPACE
```

### Option 3: Rule Porting
Port prbl-scanner regex/AST patterns directly into Repository Detective's existing deterministic rules (`scanners/deterministic.go`):
- No external dependency
- Tighter integration with Repository Detective finding schema
- More maintenance burden (sync upstream changes)

## Configuration

**New config keys:**
```yaml
prbl_scanner_enabled: true
prbl_scanner_min_severity: medium  # low, medium, high
prbl_scanner_languages:
  - javascript
  - typescript
  - python
prbl_scanner_skip_patterns:
  - node_modules
  - vendor
  - "*.test.*"
```

## Benefits

1. **Catches AI-specific blind spots** — Traditional scanners don't look for `Math.random()` tokens or env fallback patterns
2. **Deterministic + fast** — Pattern matching, no LLM latency/cost
3. **Pre-commit gate** — Can block PRs with critical AI-introduced vulns
4. **Complements existing scanners** — Trivy/Grype (deps), gosec (Go static), prbl (AI patterns)
5. **Low false positive rate** — Rules target specific anti-patterns

## Token Efficiency Impact

- Reduces LLM review workload by catching obvious issues deterministically
- CAH harness skips high-confidence prbl findings (no debate needed)
- Estimated 15-25% reduction in AI recommendation API calls for JS/TS/Python repos

## Implementation Effort

- **Option 1 (Direct):** 4-6 hours (wrapper + integration + tests)
- **Option 2 (Submodule):** 1-2 hours (CI config only)
- **Option 3 (Porting):** 8-12 hours (rule translation + testing)

## Testing Strategy

1. Clone prbl-scanner testdata into Repository Detective `testdata/prbl/`
2. Add Go tests verifying each rule detection
3. Run against known vulnerable repos (Repository Detective itself, internal projects)
4. Measure false positive rate vs prbl-scanner baseline

## Next Steps

- [ ] Review prbl-scanner LICENSE (ensure compatible with Repository Detective)
- [ ] Choose integration approach (Option 1 recommended)
- [ ] Create CMB draft for approval
- [ ] Implement scanner wrapper
- [ ] Add tests + documentation
- [ ] Deploy to staging, validate against real repos

---

**References:**
- prbl-scanner README: https://github.com/noreplywmsplaybook-pixel/prbl-scanner/blob/main/README.md
- prbl-scanner RULES.md: https://github.com/noreplywmsplaybook-pixel/prbl-scanner/blob/main/RULES.md
- Repository-Detective scanners: `/tmp/repository-detective/scanners/`
- Repository-Detective docs: `/tmp/repository-detective/docs/`
