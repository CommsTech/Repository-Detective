package reconcile

import (
	"context"
	"fmt"
	"strings"
	"time"

	"git.commsnet.org/commstech/repository-detective/calibration"
	"git.commsnet.org/commstech/repository-detective/closure"
	"git.commsnet.org/commstech/repository-detective/issues"
	"git.commsnet.org/commstech/repository-detective/profile"
	"git.commsnet.org/commstech/repository-detective/store"
)

// ForgeActions applies comments, labels, and optional close on a forge issue.
type ForgeActions interface {
	AddIssueLabels(ctx context.Context, owner, repo string, issueNumber int, labels []string) error
	CreateIssueComment(ctx context.Context, owner, repo string, issueNumber int, body string) error
	CloseIssue(ctx context.Context, owner, repo string, issueNumber int) error
	AnnotateCalibration(ctx context.Context, forgeType, owner, repo string, issueNumber int, falsePositive bool, reason string) error
}

// Config controls reconciliation behavior.
type Config struct {
	Enabled             bool
	Comment             bool
	CloseVerified       bool
	CloseDuplicates     bool
	MaxCommentsPerIssue int
	PublicBasePath      string
	BetaNoiseRuleIDs    map[string]bool
}

// Engine reconciles tracked external issues against current scan data.
type Engine struct {
	store   store.QueryStore
	matcher *calibration.Matcher
	forge   ForgeActions
	cfg     Config
}

// NewEngine creates a reconciliation engine.
func NewEngine(s store.QueryStore, matcher *calibration.Matcher, forge ForgeActions, cfg Config) *Engine {
	if cfg.MaxCommentsPerIssue <= 0 {
		cfg.MaxCommentsPerIssue = 3
	}
	if cfg.BetaNoiseRuleIDs == nil {
		cfg.BetaNoiseRuleIDs = map[string]bool{}
		for _, id := range profile.BetaNoiseRuleIDs {
			cfg.BetaNoiseRuleIDs[strings.ToUpper(id)] = true
		}
	}
	return &Engine{store: s, matcher: matcher, forge: forge, cfg: cfg}
}

type issueContext struct {
	ext        store.ExternalIssue
	finding    store.Finding
	scanID     string
	inScan     bool
	scanner    string
	scanOK     bool
	scannerRan bool
}

// Preview computes reconciliation without forge side effects.
func (e *Engine) Preview(ctx context.Context, repositoryID int64) (Result, error) {
	return e.run(ctx, repositoryID, true)
}

// Apply executes reconciliation actions on the forge.
func (e *Engine) Apply(ctx context.Context, repositoryID int64) (Result, error) {
	return e.run(ctx, repositoryID, false)
}

func (e *Engine) run(ctx context.Context, repositoryID int64, preview bool) (Result, error) {
	repo, err := e.store.GetRepository(ctx, repositoryID)
	if err != nil {
		return Result{}, err
	}
	extIssues, err := e.store.ListExternalIssuesByRepository(ctx, repositoryID, store.ListOptions{Limit: 500})
	if err != nil {
		return Result{}, err
	}
	latestScan, err := e.store.GetLatestReconcilableScanForRepository(ctx, repositoryID)
	if err != nil {
		return Result{}, fmt.Errorf("latest reconcilable scan: %w", err)
	}
	if latestScan.ID == "" {
		return Result{}, fmt.Errorf("no fully persisted scan available for reconciliation")
	}
	pipeline := store.PipelineStateFromSummary(latestScan.SummaryJSON)
	instanceCount, _ := e.store.CountFindingInstancesForScan(ctx, latestScan.ID)
	if !pipeline.IsReconcilable(instanceCount) {
		return Result{}, fmt.Errorf("scan %s persistence incomplete (%d/%d instances) — reconciliation blocked",
			latestScan.ID, instanceCount, pipeline.IssuesFound)
	}
	scannerResults := map[string]string{}
	if latestScan.ID != "" {
		rows, err := e.store.ListScannerResultsByScan(ctx, latestScan.ID)
		if err != nil {
			return Result{}, fmt.Errorf("list scanner results for reconciliation: %w", err)
		}
		for _, r := range rows {
			scannerResults[strings.ToLower(r.ScannerName)] = r.Status
		}
	}
	fpInScan := map[string]bool{}
	if latestScan.ID != "" {
		var err error
		fpInScan, err = e.store.ListFingerprintsInScan(ctx, latestScan.ID, repositoryID)
		if err != nil {
			return Result{}, fmt.Errorf("list fingerprints in scan for reconciliation: %w", err)
		}
	}

	openIssues := make([]store.ExternalIssue, 0, len(extIssues))
	findingIDs := make([]int64, 0, len(extIssues))
	for _, ext := range extIssues {
		if ext.State != "open" {
			continue
		}
		openIssues = append(openIssues, ext)
		findingIDs = append(findingIDs, ext.FindingID)
	}
	findingsByID, err := e.store.ListFindingsByIDs(ctx, findingIDs)
	if err != nil {
		return Result{}, fmt.Errorf("load findings for reconciliation: %w", err)
	}

	byFingerprint := map[string][]store.ExternalIssue{}
	contexts := make([]issueContext, 0, len(openIssues))
	for _, ext := range openIssues {
		f, ok := findingsByID[ext.FindingID]
		if !ok {
			continue
		}
		byFingerprint[f.Fingerprint] = append(byFingerprint[f.Fingerprint], ext)
		scanner := closure.ScannerForSource(f.Source)
		scannerStatus := scannerResults[strings.ToLower(scanner)]
		contexts = append(contexts, issueContext{
			ext:        ext,
			finding:    f,
			scanID:     latestScan.ID,
			inScan:     fpInScan[f.Fingerprint],
			scanner:    scanner,
			scanOK:     latestScan.ID != "",
			scannerRan: scannerStatus != "" && !closure.ScannerMissing(scannerStatus),
		})
	}

	runID := fmt.Sprintf("reconcile-%d-%d", repositoryID, time.Now().Unix())
	result := Result{
		RunID:        runID,
		RepositoryID: repositoryID,
		Preview:      preview,
	}

	if e.matcher != nil {
		if err := e.matcher.LoadRepository(ctx, repositoryID); err != nil {
			return Result{}, fmt.Errorf("load semantic matcher for reconciliation: %w", err)
		}
	}

	for _, ic := range contexts {
		item := e.classify(ic, byFingerprint[ic.finding.Fingerprint])
		result.Items = append(result.Items, item)
	}

	if preview {
		items := make([]store.ReconciliationItemRecord, len(result.Items))
		for i, it := range result.Items {
			items[i] = store.ReconciliationItemRecord{
				RunID: runID, IssueNumber: it.IssueNumber, FindingID: it.FindingID,
				Status: it.Status, ProposedAction: it.ProposedAction, Reason: it.Reason,
			}
		}
		_ = e.store.SaveReconciliationRun(ctx, store.ReconciliationRun{
			RunID: runID, RepositoryID: repositoryID, Preview: true,
			ItemCount: len(result.Items), Applied: 0, CreatedAt: time.Now().UTC(),
		}, items)
		return result, nil
	}

	commentCounts := map[int]int{}
	for i := range result.Items {
		item := &result.Items[i]
		if err := e.applyItem(ctx, repo, item, commentCounts); err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("#%d: %v", item.IssueNumber, err))
			result.Skipped++
		} else if item.ProposedAction != ActionNone {
			result.Applied++
		} else {
			result.Skipped++
		}
	}

	items := make([]store.ReconciliationItemRecord, len(result.Items))
	for i, it := range result.Items {
		items[i] = store.ReconciliationItemRecord{
			RunID: runID, IssueNumber: it.IssueNumber, FindingID: it.FindingID,
			Status: it.Status, ProposedAction: it.ProposedAction, Reason: it.Reason,
		}
	}
	_ = e.store.SaveReconciliationRun(ctx, store.ReconciliationRun{
		RunID: runID, RepositoryID: repositoryID, Preview: false,
		ItemCount: len(result.Items), Applied: result.Applied, CreatedAt: time.Now().UTC(),
	}, items)
	return result, nil
}

func (e *Engine) classify(ic issueContext, dupes []store.ExternalIssue) Item {
	f := ic.finding
	item := Item{
		IssueNumber:    ic.ext.IssueNumber,
		IssueURL:       ic.ext.IssueURL,
		FindingID:      f.ID,
		Title:          f.Title,
		Fingerprint:    f.Fingerprint,
		Source:         f.Source,
		RuleID:         f.RuleID,
		Severity:       f.Severity,
		Category:       f.Category,
		FindingStatus:  f.Status,
		LatestScanID:   ic.scanID,
		InLatestScan:   ic.inScan,
		ProposedAction: ActionNone,
	}

	// Duplicate detection
	if len(dupes) > 1 {
		canonical := dupes[0].IssueNumber
		for _, d := range dupes {
			if d.IssueNumber < canonical {
				canonical = d.IssueNumber
			}
		}
		if ic.ext.IssueNumber != canonical {
			item.Status = StatusDuplicate
			item.CanonicalIssue = canonical
			item.Reason = fmt.Sprintf("Duplicate fingerprint; canonical issue #%d", canonical)
			if e.cfg.CloseDuplicates {
				item.ProposedAction = ActionCloseDuplicate
				item.LabelsToAdd = issues.ExpandLifecycleLabels(issues.LifecycleDuplicate)
			} else {
				item.ProposedAction = ActionLabel
				item.LabelsToAdd = issues.ExpandLifecycleLabels(issues.LifecycleDuplicate)
			}
			return item
		}
	}

	st := strings.ToLower(f.Status)
	if st == store.FindingStatusFalsePositive {
		item.Status = StatusFalsePositive
		item.Reason = "Finding marked false positive locally"
		item.ProposedAction = ActionMarkFalsePositive
		return item
	}
	if st == store.FindingStatusSuppressed {
		item.Status = StatusSuppressed
		item.Reason = "Finding suppressed locally"
		item.ProposedAction = ActionSuppress
		return item
	}

	if e.matcher != nil {
		in := store.FindingMatchInput{
			RepositoryID: f.RepositoryID,
			Fingerprint:  f.Fingerprint,
			Source:       f.Source,
			RuleID:       f.RuleID,
			Category:     f.Category,
			Severity:     f.Severity,
		}
		if suppressed, _ := e.matcher.IsSuppressed(f.RepositoryID, in); suppressed {
			item.Status = StatusSuppressed
			item.Reason = "Active suppression rule matches this finding"
			item.ProposedAction = ActionSuppress
			return item
		}
	}

	if f.RuleID != "" && e.cfg.BetaNoiseRuleIDs[strings.ToUpper(f.RuleID)] {
		item.Status = StatusStaleRule
		item.Reason = "Rule is report-only under current profile; likely bot bloat"
		item.ProposedAction = ActionSuppress
		return item
	}

	if !ic.scanOK {
		item.Status = StatusNeedsHumanReview
		item.Reason = "No completed scan available for comparison"
		item.ProposedAction = ActionNone
		return item
	}

	if ic.inScan {
		item.Status = StatusStillPresent
		item.Reason = "Fingerprint still present in latest scan"
		if strings.EqualFold(f.Severity, "critical") || strings.EqualFold(f.Severity, "high") {
			item.Status = StatusNeedsHumanReview
			item.Reason = "Critical/high finding still present — requires remediation"
		}
		item.ProposedAction = ActionEnrich
		return item
	}

	// Absent from latest scan
	if !ic.scannerRan && ic.scanner != "" {
		item.Status = StatusScannerNotRun
		item.Reason = fmt.Sprintf("Scanner %q did not run in latest scan — cannot verify fix", ic.scanner)
		item.ProposedAction = ActionComment
		return item
	}

	item.Status = StatusAlreadyFixedVerify
	item.Reason = "Fingerprint absent from latest scan; verify before closing"
	if e.cfg.CloseVerified {
		item.ProposedAction = ActionCloseVerified
	} else {
		item.ProposedAction = ActionComment
	}
	return item
}

func (e *Engine) applyItem(ctx context.Context, repo store.Repository, item *Item, commentCounts map[int]int) error {
	if e.forge == nil || item.ProposedAction == ActionNone {
		return nil
	}
	owner, name := repo.Owner, repo.Name
	forgeType := repo.ForgeType
	if forgeType == "" {
		forgeType = "gitea"
	}

	switch item.ProposedAction {
	case ActionMarkFalsePositive:
		return e.forge.AnnotateCalibration(ctx, forgeType, owner, name, item.IssueNumber, true, item.Reason)
	case ActionSuppress:
		return e.forge.AnnotateCalibration(ctx, forgeType, owner, name, item.IssueNumber, false, item.Reason)
	case ActionLabel:
		if len(item.LabelsToAdd) > 0 {
			return e.forge.AddIssueLabels(ctx, owner, name, item.IssueNumber, item.LabelsToAdd)
		}
	case ActionEnrich, ActionComment:
		if !e.cfg.Comment {
			return nil
		}
		if commentCounts[item.IssueNumber] >= e.cfg.MaxCommentsPerIssue {
			return nil
		}
		finding, err := e.store.GetFindingDetail(ctx, item.FindingID)
		if err != nil {
			return err
		}
		body := EnrichmentComment(finding.Finding, item.LatestScanID, e.cfg.PublicBasePath, repo.ID, "")
		if err := e.forge.CreateIssueComment(ctx, owner, name, item.IssueNumber, body); err != nil {
			return err
		}
		commentCounts[item.IssueNumber]++
		_ = e.store.AddLifecycleEvent(ctx, store.LifecycleEvent{
			FindingID: &item.FindingID,
			ScanID:    item.LatestScanID,
			EventType: store.LifecycleEventReconciled,
			Message:   item.Reason,
		})
	case ActionCloseVerified:
		if !e.cfg.CloseVerified {
			return nil
		}
		if item.Status != StatusAlreadyFixedVerify {
			return nil
		}
		if e.cfg.Comment {
			if err := e.forge.CreateIssueComment(ctx, owner, name, item.IssueNumber,
				"Repository Detective verified this finding is absent from the latest scan. Closing with evidence."); err != nil {
				return fmt.Errorf("comment on verified close #%d: %w", item.IssueNumber, err)
			}
		}
		if err := e.forge.CloseIssue(ctx, owner, name, item.IssueNumber); err != nil {
			return err
		}
		if _, err := e.store.UpsertExternalIssue(ctx, store.ExternalIssue{
			FindingID: item.FindingID, ForgeType: forgeType,
			IssueNumber: item.IssueNumber, IssueURL: item.IssueURL, State: "closed",
		}); err != nil {
			return fmt.Errorf("upsert closed external issue #%d: %w", item.IssueNumber, err)
		}
		if err := e.store.UpdateFindingStatus(ctx, item.FindingID, store.FindingStatusResolvedVerified); err != nil {
			return fmt.Errorf("mark finding verified for #%d: %w", item.IssueNumber, err)
		}
	case ActionCloseDuplicate:
		if !e.cfg.CloseDuplicates {
			return nil
		}
		if item.Status != StatusDuplicate {
			return nil
		}
		if len(item.LabelsToAdd) > 0 {
			if err := e.forge.AddIssueLabels(ctx, owner, name, item.IssueNumber, item.LabelsToAdd); err != nil {
				return fmt.Errorf("add duplicate labels on #%d: %w", item.IssueNumber, err)
			}
		}
		comment := fmt.Sprintf(
			"Repository Detective closed this issue as a **duplicate** of #%d.\n\n"+
				"- Same fingerprint: `%s`\n"+
				"- Latest scan: `%s`\n"+
				"- Canonical issue remains open for active tracking.",
			item.CanonicalIssue, item.Fingerprint, item.LatestScanID,
		)
		if e.cfg.Comment {
			if err := e.forge.CreateIssueComment(ctx, owner, name, item.IssueNumber, comment); err != nil {
				return fmt.Errorf("comment on duplicate #%d: %w", item.IssueNumber, err)
			}
		}
		if err := e.forge.CloseIssue(ctx, owner, name, item.IssueNumber); err != nil {
			return err
		}
		_, _ = e.store.UpsertExternalIssue(ctx, store.ExternalIssue{
			FindingID: item.FindingID, ForgeType: forgeType,
			IssueNumber: item.IssueNumber, IssueURL: item.IssueURL, State: "closed",
		})
		_ = e.store.AddLifecycleEvent(ctx, store.LifecycleEvent{
			FindingID: &item.FindingID,
			ScanID:    item.LatestScanID,
			EventType: "external_issue_closed_duplicate",
			Message:   item.Reason,
		})
	}
	return nil
}
