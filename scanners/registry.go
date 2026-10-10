package scanners

import (
	"context"
	"sync"
)

type trivyScanner struct{}

func (trivyScanner) Name() string { return "trivy" }

func (trivyScanner) Run(ctx context.Context, req RunRequest) []RunResult {
	if !req.Config.EnableTrivy {
		return []RunResult{{Scanner: "trivy", Status: StatusDisabled}}
	}
	if !req.EnableSecurity {
		return []RunResult{{Scanner: "trivy", Status: StatusDisabled, Detail: "security analysis disabled"}}
	}
	return []RunResult{RunTrivy(ctx, req.Logger, req.Workspace, req.Config)}
}

type grypeScanner struct{}

func (grypeScanner) Name() string { return "grype" }

func (grypeScanner) Run(ctx context.Context, req RunRequest) []RunResult {
	if !req.Config.EnableGrype {
		return []RunResult{{Scanner: "grype", Status: StatusDisabled}}
	}
	if !req.EnableSecurity {
		return []RunResult{{Scanner: "grype", Status: StatusDisabled, Detail: "security analysis disabled"}}
	}
	return []RunResult{RunGrype(ctx, req.Logger, req.Workspace, req.Config)}
}

type gitleaksScanner struct{}

func (gitleaksScanner) Name() string { return "gitleaks" }

func (gitleaksScanner) Run(ctx context.Context, req RunRequest) []RunResult {
	if !req.Config.EnableGitleaks {
		return []RunResult{{Scanner: "gitleaks", Status: StatusDisabled}}
	}
	if !req.EnableSecurity {
		return []RunResult{{Scanner: "gitleaks", Status: StatusDisabled, Detail: "security analysis disabled"}}
	}
	return []RunResult{RunGitleaks(ctx, req.Logger, req.Workspace, req.Config)}
}

type semgrepScanner struct{}

func (semgrepScanner) Name() string { return "semgrep" }

func (semgrepScanner) Run(ctx context.Context, req RunRequest) []RunResult {
	if !req.Config.EnableSemgrep {
		return []RunResult{{Scanner: "semgrep", Status: StatusDisabled}}
	}
	if !req.EnableSecurity {
		return []RunResult{{Scanner: "semgrep", Status: StatusDisabled, Detail: "security analysis disabled"}}
	}
	return []RunResult{RunSemgrep(ctx, req.Logger, req.Workspace, req.Config)}
}

type govulncheckScanner struct{}

func (govulncheckScanner) Name() string { return "govulncheck" }

func (govulncheckScanner) Run(ctx context.Context, req RunRequest) []RunResult {
	if !req.Config.EnableGovulncheck {
		return []RunResult{{Scanner: "govulncheck", Status: StatusDisabled}}
	}
	if !req.EnableSecurity {
		return []RunResult{{Scanner: "govulncheck", Status: StatusDisabled, Detail: "security analysis disabled"}}
	}
	if !WorkspaceHasGo(req.Workspace, req.Entries) {
		return []RunResult{{Scanner: "govulncheck", Status: StatusClean, Detail: "no Go module or files"}}
	}
	return []RunResult{RunGovulncheck(ctx, req.Logger, req.Workspace, req.Config)}
}

type gosecScanner struct{}

func (gosecScanner) Name() string { return "gosec" }

func (gosecScanner) Run(ctx context.Context, req RunRequest) []RunResult {
	if !req.Config.EnableGosec {
		return []RunResult{{Scanner: "gosec", Status: StatusDisabled}}
	}
	if !req.EnableSecurity {
		return []RunResult{{Scanner: "gosec", Status: StatusDisabled, Detail: "security analysis disabled"}}
	}
	if !WorkspaceHasGo(req.Workspace, req.Entries) {
		return []RunResult{{Scanner: "gosec", Status: StatusClean, Detail: "no Go files"}}
	}
	return []RunResult{RunGosec(ctx, req.Logger, req.Workspace, req.Config)}
}

type staticcheckScanner struct{}

func (staticcheckScanner) Name() string { return "staticcheck" }

func (staticcheckScanner) Run(ctx context.Context, req RunRequest) []RunResult {
	if !req.Config.EnableStaticcheck {
		return []RunResult{{Scanner: "staticcheck", Status: StatusDisabled}}
	}
	if !req.EnableQuality {
		return []RunResult{{Scanner: "staticcheck", Status: StatusDisabled, Detail: "quality analysis disabled"}}
	}
	if !WorkspaceHasGo(req.Workspace, req.Entries) {
		return []RunResult{{Scanner: "staticcheck", Status: StatusClean, Detail: "no Go files"}}
	}
	return []RunResult{RunStaticcheck(ctx, req.Logger, req.Workspace, req.Config)}
}

type hadolintScanner struct{}

func (hadolintScanner) Name() string { return "hadolint" }

func (hadolintScanner) Run(ctx context.Context, req RunRequest) []RunResult {
	if !req.Config.EnableHadolint {
		return []RunResult{{Scanner: "hadolint", Status: StatusDisabled}}
	}
	if !req.EnableSecurity {
		return []RunResult{{Scanner: "hadolint", Status: StatusDisabled, Detail: "security analysis disabled"}}
	}
	if !WorkspaceHasDockerfiles(req.Workspace, req.Entries) {
		return []RunResult{{Scanner: "hadolint", Status: StatusClean, Detail: "no Dockerfiles"}}
	}
	return []RunResult{RunHadolint(ctx, req.Logger, req.Workspace, req.Entries, req.Config)}
}

type checkovScanner struct{}

func (checkovScanner) Name() string { return "checkov" }

func (checkovScanner) Run(ctx context.Context, req RunRequest) []RunResult {
	if !req.Config.EnableCheckov {
		return []RunResult{{Scanner: "checkov", Status: StatusDisabled}}
	}
	if !req.EnableSecurity {
		return []RunResult{{Scanner: "checkov", Status: StatusDisabled, Detail: "security analysis disabled"}}
	}
	if !WorkspaceHasIaC(req.Entries) {
		return []RunResult{{Scanner: "checkov", Status: StatusClean, Detail: "no IaC/config files"}}
	}
	return []RunResult{RunCheckov(ctx, req.Logger, req.Workspace, req.Entries, req.Config)}
}

// lintersScanner runs golangci-lint, ruff, and shellcheck in registry order.
type lintersScanner struct{}

func (lintersScanner) Name() string { return "linters" }

func (lintersScanner) Run(ctx context.Context, req RunRequest) []RunResult {
	if !req.Config.EnableLinters {
		return []RunResult{{Scanner: "linters", Status: StatusDisabled}}
	}
	return RunLinters(ctx, req.Logger, req.Workspace, req.Entries, req.EnableSecurity, req.EnableQuality, req.Config)
}

// DefaultRegistry returns scanners in execution order:
// trivy, grype, gitleaks, semgrep, govulncheck, gosec, staticcheck, hadolint, checkov, linters.
func DefaultRegistry() []Scanner {
	return []Scanner{
		trivyScanner{},
		grypeScanner{},
		gitleaksScanner{},
		semgrepScanner{},
		govulncheckScanner{},
		gosecScanner{},
		staticcheckScanner{},
		hadolintScanner{},
		checkovScanner{},
		lintersScanner{},
	}
}

// Registry owns ordered scanner execution.
type Registry struct {
	scanners []Scanner
}

// NewRegistry creates a registry from the given scanners.
func NewRegistry(scanners ...Scanner) *Registry {
	return &Registry{scanners: scanners}
}

// DefaultScannerRegistry is the production scanner registry.
func DefaultScannerRegistry() *Registry {
	return NewRegistry(DefaultRegistry()...)
}

// Names returns registry scanner names in order.
func (r *Registry) Names() []string {
	names := make([]string, 0, len(r.scanners))
	for _, scanner := range r.scanners {
		names = append(names, scanner.Name())
	}
	return names
}

// RunAll executes every registered scanner concurrently and aggregates results
// in registry order. Each scanner inherits the parent context (and its deadline).
func (r *Registry) RunAll(ctx context.Context, req RunRequest) RunSummary {
	type indexed struct {
		idx     int
		results []RunResult
	}
	outCh := make(chan indexed, len(r.scanners))
	var wg sync.WaitGroup
	for i, scanner := range r.scanners {
		wg.Add(1)
		go func(idx int, s Scanner) {
			defer wg.Done()
			outCh <- indexed{idx: idx, results: s.Run(ctx, req)}
		}(i, scanner)
	}
	go func() {
		wg.Wait()
		close(outCh)
	}()

	ordered := make([][]RunResult, len(r.scanners))
	for item := range outCh {
		ordered[item.idx] = item.results
	}
	var summary RunSummary
	for _, results := range ordered {
		summary.Results = append(summary.Results, results...)
	}
	return summary
}
