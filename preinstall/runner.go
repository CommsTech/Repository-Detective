package preinstall

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"git.commsnet.org/commstech/repository-detective/graph"
	"git.commsnet.org/commstech/repository-detective/health"
	"git.commsnet.org/commstech/repository-detective/issues"
	"git.commsnet.org/commstech/repository-detective/scanners"
	"git.commsnet.org/commstech/repository-detective/store"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

// AuditStore persists pre-install audit data.
type AuditStore interface {
	CreateAuditRequest(ctx context.Context, req store.AuditRequest) (store.AuditRequest, error)
	UpdateAuditRequest(ctx context.Context, req store.AuditRequest) error
	GetAuditRequest(ctx context.Context, auditID string) (store.AuditRequest, error)
	ListAuditRequests(ctx context.Context, opts store.ListOptions) ([]store.AuditRequest, error)
	AddAuditFindings(ctx context.Context, findings []store.AuditFinding) error
	ListAuditFindings(ctx context.Context, auditID string) ([]store.AuditFinding, error)
	AddDisclosureReport(ctx context.Context, report store.DisclosureReport) (store.DisclosureReport, error)
	ListDisclosureReports(ctx context.Context, auditID string) ([]store.DisclosureReport, error)
	GetDisclosureReport(ctx context.Context, id int64) (store.DisclosureReport, error)
	MarkDisclosureReportReviewed(ctx context.Context, id int64) error
	SaveAuditGraph(ctx context.Context, record store.AuditGraphRecord) error
}

// Runner executes third-party pre-install audits asynchronously.
type Runner struct {
	store       AuditStore
	cfg         Config
	scannerBase scanners.Config
	logger      *logrus.Logger
	notifier    AuditNotifier
}

// NewRunner creates a pre-install audit runner.
func NewRunner(s AuditStore, cfg Config, scannerBase scanners.Config, logger *logrus.Logger) *Runner {
	if logger == nil {
		logger = logrus.New()
	}
	return &Runner{store: s, cfg: cfg, scannerBase: scannerBase, logger: logger}
}

// SetAuditNotifier attaches an optional notification hook.
func (r *Runner) SetAuditNotifier(n AuditNotifier) {
	r.notifier = n
}

// StartAudit validates the URL, creates an audit record, and runs the audit asynchronously.
func (r *Runner) StartAudit(ctx context.Context, repoURL, auditDepth string) (string, error) {
	if !r.cfg.Enabled {
		return "", fmt.Errorf("pre-install audit is disabled")
	}
	if r.store == nil {
		return "", fmt.Errorf("audit store unavailable")
	}

	parsed, err := ValidateRepoURL(repoURL, r.cfg.AllowPrivateNetworks)
	if err != nil {
		return "", err
	}
	depth := NormalizeAuditDepth(auditDepth)
	auditID := uuid.NewString()
	now := time.Now().UTC()

	req := store.AuditRequest{
		AuditID:           auditID,
		RepoURL:           parsed.Original,
		NormalizedRepoURL: parsed.Normalized,
		RepoHost:          parsed.Host,
		RepoOwner:         parsed.Owner,
		RepoName:          parsed.Name,
		AuditDepth:        depth,
		Status:            store.AuditStatusQueued,
		Recommendation:    store.AuditRecommendationUnknown,
		StartedAt:         now,
		SummaryJSON:       json.RawMessage(`{}`),
	}
	if _, err := r.store.CreateAuditRequest(ctx, req); err != nil {
		return "", err
	}

	go r.runAudit(context.WithoutCancel(ctx), auditID, parsed, depth)
	return auditID, nil
}

func (r *Runner) runAudit(parent context.Context, auditID string, parsed ParsedRepoURL, depth string) {
	if parent == nil {
		parent = context.Background()
	}
	ctx, cancel := context.WithTimeout(parent, r.cfg.auditTimeout())
	defer cancel()

	req, err := r.store.GetAuditRequest(ctx, auditID)
	if err != nil {
		r.logger.Errorf("preinstall audit %s load: %v", auditID, err)
		return
	}
	req.Status = store.AuditStatusRunning
	if err := r.store.UpdateAuditRequest(ctx, req); err != nil {
		r.logger.Errorf("preinstall audit %s mark running: %v", auditID, err)
		return
	}

	var sandboxMeta SandboxMeta
	fail := func(msg string) {
		stage := ClassifyFailureStage(msg)
		ApplyAuditFailure(&req, stage, msg, sandboxMeta)
		if err := r.store.UpdateAuditRequest(ctx, req); err != nil {
			r.logger.Errorf("preinstall audit %s mark failed: %v", auditID, err)
		}
	}

	var retainWorkspace bool
	var clone CloneResult
	defer func() {
		if !retainWorkspace && clone.Cleanup != nil {
			clone.Cleanup()
		}
	}()

	origFail := fail
	fail = func(msg string) {
		if r.cfg.SandboxRetainOnFailure && clone.Cleanup != nil {
			retainWorkspace = true
		}
		origFail(msg)
	}

	clone, cloneErr := ShallowClone(ctx, parsed, r.cfg)
	if cloneErr != nil {
		fail(cloneErr.Error())
		return
	}
	sandboxMeta = clone.Sandbox

	req.CommitSHA = clone.CommitSHA
	req.DefaultBranch = clone.DefaultBranch

	entries, err := scanners.ListWorkspaceFiles(clone.WorkspaceDir, r.cfg.MaxFiles)
	if err != nil {
		fail(fmt.Sprintf("list workspace files: %v", err))
		return
	}

	scannerCfg := scannerConfigForDepth(depth, r.scannerBase)
	summary := scanners.RunAll(ctx, r.logger, clone.WorkspaceDir, entries, scannerCfg, true, true)
	if depth != "quick" && scannerCfg.EnableGitleaks && scannerCfg.SecretScanGitHistoryEnabled && r.cfg.AllowGitClone {
		histCfg := scannerCfg
		if histCfg.SecretScanRecentCommitsMax <= 0 {
			histCfg.SecretScanRecentCommitsMax = 20
		}
		hist := scanners.RunGitleaksGitHistory(ctx, r.logger, clone.WorkspaceDir, histCfg, scanners.SecretScopeRecentCommits, clone.WorkspaceDir)
		summary.Results = append(summary.Results, hist)
	}
	scannerResults := toScannerResults(summary)

	var findings []store.AuditFinding
	findings = append(findings, findingsFromScanners(auditID, parsed.Normalized, summary)...)
	findings = append(findings, staticFindingsWithAuditID(auditID, RunStaticChecks(clone.WorkspaceDir, parsed.Normalized, r.cfg.MaxFindings))...)

	var graphNodes, graphEdges int
	if depth != "quick" {
		healthCfg := healthConfigForAudit(depth, r.cfg.Health)
		fileInputs := health.LoadWorkspaceFiles(clone.WorkspaceDir, entries, 512*1024)
		allPaths := make([]string, 0, len(entries))
		for _, e := range entries {
			allPaths = append(allPaths, e.Path)
		}
		findings = append(findings, healthFindingsToAudit(auditID, parsed.Normalized, health.Run(health.RunInput{
			Files: fileInputs, AllPaths: allPaths,
		}, healthCfg, nil))...)

		if r.cfg.Graph.Enabled {
			graphFiles := graph.LoadWorkspaceFiles(clone.WorkspaceDir, entries, 512*1024, nil)
			g, graphFindings := graph.Build(ctx, graph.BuildInput{
				AuditID: auditID, Files: graphFiles, AllPaths: allPaths,
			}, r.cfg.Graph, nil)
			findings = append(findings, graphFindingsToAudit(auditID, parsed.Normalized, graphFindings)...)
			graphNodes = g.Metrics.NodeCount
			graphEdges = g.Metrics.EdgeCount
			finished := time.Now().UTC()
			if raw, err := json.Marshal(g); err == nil {
				_ = r.store.SaveAuditGraph(ctx, store.AuditGraphRecord{
					AuditID: auditID, GraphJSON: raw,
					NodeCount: graphNodes, EdgeCount: graphEdges,
					GeneratedAt: finished,
				})
			}
		}
	}

	if len(findings) > r.cfg.MaxFindings {
		findings = findings[:r.cfg.MaxFindings]
	}

	findings = filterPreinstallFindings(findings)

	risk := ComputeRiskScore(findings, scannerResults)
	finished := time.Now().UTC()

	summaryPayload := map[string]any{
		"audit_depth":       depth,
		"workspace_bytes":   clone.TotalBytes,
		"workspace_files":   clone.FileCount,
		"scanner_results":   scannerResults,
		"risk_explanation":  risk.Explanation,
		"finding_count":     len(findings),
		"sandbox":           clone.Sandbox,
		"issues_created":    0,
		"prs_created":       0,
	}
	if graphNodes > 0 {
		summaryPayload["graph_nodes"] = graphNodes
		summaryPayload["graph_edges"] = graphEdges
	}
	summaryJSON, _ := json.Marshal(summaryPayload)

	req.Status = store.AuditStatusCompleted
	req.RiskScore = risk.Score
	req.Recommendation = risk.Recommendation
	req.FinishedAt = &finished
	req.SummaryJSON = summaryJSON
	req.Error = ""

	if err := r.store.UpdateAuditRequest(ctx, req); err != nil {
		r.logger.Errorf("preinstall audit %s update: %v", auditID, err)
		return
	}

	if r.notifier != nil {
		r.notifier.OnAuditComplete(req, len(findings))
	}

	for i := range findings {
		findings[i].AuditID = auditID
	}
	if err := r.store.AddAuditFindings(ctx, findings); err != nil {
		r.logger.Errorf("preinstall audit %s findings: %v", auditID, err)
	}

	storedFindings, err := r.store.ListAuditFindings(ctx, auditID)
	if err != nil {
		r.logger.Errorf("preinstall audit %s reload findings: %v", auditID, err)
		storedFindings = findings
	}

	for _, report := range GenerateReports(r.cfg, req, storedFindings, scannerResults) {
		report.AuditID = auditID
		if _, err := r.store.AddDisclosureReport(ctx, report); err != nil {
			r.logger.Errorf("preinstall audit %s report: %v", auditID, err)
			continue
		}
		if r.notifier != nil {
			r.notifier.OnDisclosureReport(auditID, report.ReportType)
		}
	}
}

func scannerConfigForDepth(depth string, base scanners.Config) scanners.Config {
	cfg := base
	switch depth {
	case "quick":
		cfg.EnableSemgrep = false
		cfg.EnableGrype = false
		cfg.EnableLinters = false
		cfg.EnableGovulncheck = false
		cfg.EnableGosec = false
		cfg.EnableStaticcheck = false
		cfg.EnableCheckov = false
		cfg.EnableTrivy = true
		cfg.EnableGitleaks = true
	case "deep", "standard":
		cfg.EnableTrivy = true
		cfg.EnableGrype = true
		cfg.EnableGitleaks = true
		cfg.EnableSemgrep = true
		cfg.EnableLinters = true
	}
	return cfg
}

// ScannerConfigForDepthForTest exposes depth-based scanner tuning for tests.
func ScannerConfigForDepthForTest(depth string, base scanners.Config) scanners.Config {
	return scannerConfigForDepth(depth, base)
}

func toScannerResults(summary scanners.RunSummary) []store.AuditScannerResult {
	out := make([]store.AuditScannerResult, 0, len(summary.Results))
	for _, res := range summary.Results {
		out = append(out, store.AuditScannerResult{
			Scanner:       res.Scanner,
			Status:        string(res.Status),
			FindingsCount: len(res.Findings),
			Detail:        res.Detail,
		})
	}
	return out
}

func findingsFromScanners(auditID, repoRef string, summary scanners.RunSummary) []store.AuditFinding {
	var out []store.AuditFinding
	for _, res := range summary.Results {
		for _, f := range res.Findings {
			evidence := issues.SanitizeSecretEvidence(firstNonEmpty(f.Code, f.Description))
			fp := issues.ComputeFingerprint(issues.FingerprintInput{
				Repository:   repoRef,
				Category:     f.Category,
				Source:       f.Source,
				RuleID:       f.Reference,
				File:         f.File,
				Line:         f.Line,
				EvidenceHash: issues.SanitizedEvidenceHash(evidence),
			})
			conf := f.Confidence
			if conf <= 0 {
				conf = 0.95
			}
			out = append(out, store.AuditFinding{
				AuditID:          auditID,
				Fingerprint:      fp,
				Category:         normalizeCategory(f),
				Severity:         strings.ToLower(f.Severity),
				Confidence:       conf,
				Source:           f.Source,
				RuleID:           firstNonEmpty(f.Reference, f.ID),
				FilePath:         f.File,
				Line:             f.Line,
				Title:            f.Title,
				EvidenceRedacted: firstNonEmpty(f.Title, evidence),
			})
		}
	}
	return out
}

func normalizeCategory(f scanners.Finding) string {
	cat := strings.ToLower(strings.TrimSpace(f.Category))
	if cat != "" {
		return cat
	}
	if f.Source == "gitleaks" {
		return "secret"
	}
	return "security"
}

func healthConfigForAudit(depth string, base health.Config) health.Config {
	cfg := base
	if cfg.MaxFindings <= 0 {
		cfg.MaxFindings = 100
	}
	cfg.EnableAIRisk = false
	if depth == "deep" && base.EnableAIRisk {
		cfg.EnableAIRisk = true
	}
	return cfg
}

func healthFindingsToAudit(auditID, repoRef string, findings []health.Finding) []store.AuditFinding {
	var out []store.AuditFinding
	for _, f := range findings {
		evidence := issues.SanitizeSecretEvidence(firstNonEmpty(f.Evidence, f.Description))
		fp := issues.ComputeFingerprint(issues.FingerprintInput{
			Repository:   repoRef,
			Category:     f.Category,
			Source:       f.Source,
			RuleID:       f.RuleID,
			File:         f.File,
			Line:         f.Line,
			EvidenceHash: issues.SanitizedEvidenceHash(evidence),
		})
		out = append(out, store.AuditFinding{
			AuditID:          auditID,
			Fingerprint:      fp,
			Category:         f.Category,
			Severity:         strings.ToLower(f.Severity),
			Confidence:       f.Confidence,
			Source:           f.Source,
			RuleID:           f.RuleID,
			FilePath:         f.File,
			Line:             f.Line,
			Title:            f.Title,
			EvidenceRedacted: firstNonEmpty(f.Description, evidence),
		})
	}
	return out
}

func graphFindingsToAudit(auditID, repoRef string, findings []graph.GraphFinding) []store.AuditFinding {
	var out []store.AuditFinding
	for _, f := range findings {
		evidence := issues.SanitizeSecretEvidence(firstNonEmpty(f.Evidence, f.Description))
		fp := issues.ComputeFingerprint(issues.FingerprintInput{
			Repository:   repoRef,
			Category:     f.Category,
			Source:       f.Source,
			RuleID:       f.RuleID,
			File:         f.File,
			Line:         f.Line,
			EvidenceHash: issues.SanitizedEvidenceHash(evidence),
		})
		out = append(out, store.AuditFinding{
			AuditID:          auditID,
			Fingerprint:      fp,
			Category:         f.Category,
			Severity:         strings.ToLower(f.Severity),
			Confidence:       f.Confidence,
			Source:           f.Source,
			RuleID:           f.RuleID,
			FilePath:         f.File,
			Line:             f.Line,
			Title:            f.Title,
			EvidenceRedacted: firstNonEmpty(f.Description, evidence),
		})
	}
	return out
}

func staticFindingsWithAuditID(auditID string, findings []store.AuditFinding) []store.AuditFinding {
	for i := range findings {
		findings[i].AuditID = auditID
	}
	return findings
}
