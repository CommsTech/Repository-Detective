package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	_ "modernc.org/sqlite"
)

func isNoRows(err error) bool {
	return errors.Is(err, sql.ErrNoRows)
}

// SQLiteStore implements Store with SQLite.
type SQLiteStore struct {
	db *sql.DB

	// Short TTL cache for DashboardSummary — shared by /ui, /ui/health, /ui/reports,
	// and the API. Keeps navigating between those pages snappy without long-lived staleness.
	dashboardSummaryMu    sync.Mutex
	dashboardSummaryCache map[int]dashboardSummaryCacheEntry
}

type dashboardSummaryCacheEntry struct {
	summary   DashboardSummary
	expiresAt time.Time
}

const dashboardSummaryCacheTTL = 2 * time.Second

// OpenSQLite opens (or creates) a SQLite database and runs migrations.
func OpenSQLite(path string) (*SQLiteStore, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open sqlite database: %w", err)
	}

	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)

	if err := db.Ping(); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("ping sqlite database: %w", err)
	}

	pragmas := []string{
		"PRAGMA busy_timeout = 30000",
		"PRAGMA journal_mode = WAL",
		"PRAGMA synchronous = NORMAL",
	}
	if runningInTestBinary() {
		// Keep test runs fast and avoid host fsync stalls in CI.
		pragmas = append(pragmas,
			"PRAGMA journal_mode = MEMORY",
			"PRAGMA synchronous = OFF",
			"PRAGMA temp_store = MEMORY",
		)
	}
	for _, pragma := range pragmas {
		if _, err := db.Exec(pragma); err != nil {
			_ = db.Close()
			return nil, fmt.Errorf("sqlite pragma failed (%s): %w", pragma, err)
		}
	}

	if err := applyMigrations(db); err != nil {
		_ = db.Close()
		return nil, err
	}

	return &SQLiteStore{db: db}, nil
}

func runningInTestBinary() bool {
	return strings.HasSuffix(os.Args[0], ".test")
}

func (s *SQLiteStore) Close() error {
	if s.db == nil {
		return nil
	}
	return s.db.Close()
}

func formatTime(t time.Time) string {
	return t.UTC().Format(time.RFC3339Nano)
}

func parseTime(value string) time.Time {
	if value == "" {
		return time.Time{}
	}
	t, err := time.Parse(time.RFC3339Nano, value)
	if err != nil {
		t, _ = time.Parse(time.RFC3339, value)
	}
	return t
}

func parseTimePtr(value sql.NullString) *time.Time {
	if !value.Valid || value.String == "" {
		return nil
	}
	t := parseTime(value.String)
	return &t
}

func nullTimeString(t *time.Time) any {
	if t == nil || t.IsZero() {
		return nil
	}
	return formatTime(*t)
}

func boolToInt(v bool) int {
	if v {
		return 1
	}
	return 0
}

func intToBool(v int) bool {
	return v != 0
}

func nullBoolPtr(v sql.NullInt64) *bool {
	if !v.Valid {
		return nil
	}
	b := v.Int64 != 0
	return &b
}

func nullStringPtr(v sql.NullString) *string {
	if !v.Valid {
		return nil
	}
	s := v.String
	return &s
}

func nullIntPtr(v sql.NullInt64) *int {
	if !v.Valid {
		return nil
	}
	i := int(v.Int64)
	return &i
}

func nullFloatPtr(v sql.NullFloat64) *float64 {
	if !v.Valid {
		return nil
	}
	f := v.Float64
	return &f
}

func derefBool(v *bool, fallback bool) bool {
	if v == nil {
		return fallback
	}
	return *v
}

func derefString(v *string, fallback string) string {
	if v == nil {
		return fallback
	}
	return *v
}

func derefInt(v *int, fallback int) int {
	if v == nil {
		return fallback
	}
	return *v
}

func derefFloat(v *float64, fallback float64) float64 {
	if v == nil {
		return fallback
	}
	return *v
}

func optionalBool(v *bool) interface{} {
	if v == nil {
		return nil
	}
	return boolToInt(*v)
}

func optionalString(v *string) interface{} {
	if v == nil {
		return nil
	}
	return *v
}

func optionalInt(v *int) interface{} {
	if v == nil {
		return nil
	}
	return *v
}

func optionalFloat(v *float64) interface{} {
	if v == nil {
		return nil
	}
	return *v
}

func scanRepository(row scanner) (Repository, error) {
	var repo Repository
	var connected int
	var createdAt, updatedAt string
	err := row.Scan(
		&repo.ID,
		&repo.ForgeType,
		&repo.Owner,
		&repo.Name,
		&repo.FullName,
		&repo.CloneURL,
		&repo.DefaultBranch,
		&connected,
		&createdAt,
		&updatedAt,
	)
	if err != nil {
		return Repository{}, err
	}
	repo.ConnectedRepo = intToBool(connected)
	repo.CreatedAt = parseTime(createdAt)
	repo.UpdatedAt = parseTime(updatedAt)
	return repo, nil
}

type scanner interface {
	Scan(dest ...any) error
}

const repositoryColumns = `id, forge_type, owner, name, full_name, clone_url, default_branch, connected_repo, created_at, updated_at`

func (s *SQLiteStore) UpsertRepository(ctx context.Context, repo Repository) (Repository, error) {
	now := time.Now().UTC()
	if repo.ForgeType == "" {
		repo.ForgeType = ForgeTypeGitea
	}
	if repo.FullName == "" {
		repo.FullName = repo.Owner + "/" + repo.Name
	}
	if repo.CreatedAt.IsZero() {
		repo.CreatedAt = now
	}
	repo.UpdatedAt = now

	_, err := s.db.ExecContext(ctx, `
		INSERT INTO repositories (forge_type, owner, name, full_name, clone_url, default_branch, connected_repo, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(forge_type, full_name) DO UPDATE SET
			owner = excluded.owner,
			name = excluded.name,
			clone_url = CASE WHEN excluded.clone_url != '' THEN excluded.clone_url ELSE repositories.clone_url END,
			default_branch = CASE WHEN excluded.default_branch != '' THEN excluded.default_branch ELSE repositories.default_branch END,
			connected_repo = excluded.connected_repo,
			updated_at = excluded.updated_at
	`, repo.ForgeType, repo.Owner, repo.Name, repo.FullName, repo.CloneURL, repo.DefaultBranch, boolToInt(repo.ConnectedRepo), formatTime(repo.CreatedAt), formatTime(repo.UpdatedAt))
	if err != nil {
		return Repository{}, fmt.Errorf("upsert repository: %w", err)
	}

	return s.GetRepositoryByFullName(ctx, repo.ForgeType, repo.FullName)
}

func (s *SQLiteStore) GetRepositoryByFullName(ctx context.Context, forgeType, fullName string) (Repository, error) {
	row := s.db.QueryRowContext(ctx, `SELECT `+repositoryColumns+` FROM repositories WHERE forge_type = ? AND full_name = ?`, forgeType, fullName)
	repo, err := scanRepository(row)
	if err != nil {
		return Repository{}, fmt.Errorf("get repository: %w", err)
	}
	return repo, nil
}

func (s *SQLiteStore) ListRepositories(ctx context.Context) ([]Repository, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT `+repositoryColumns+` FROM repositories ORDER BY full_name`)
	if err != nil {
		return nil, fmt.Errorf("list repositories: %w", err)
	}
	defer rows.Close()

	var repos []Repository
	for rows.Next() {
		repo, err := scanRepository(rows)
		if err != nil {
			return nil, fmt.Errorf("scan repository: %w", err)
		}
		repos = append(repos, repo)
	}
	return repos, rows.Err()
}

func (s *SQLiteStore) GetRepoSettings(ctx context.Context, repositoryID int64) (RepoSettings, error) {
	var settings RepoSettings
	var scanProfile sql.NullString
	var notifEnabled sql.NullInt64
	var notifMinSeverity, notifEvents sql.NullString
	var notifCooldown sql.NullInt64
	var enabled, llm, trivy, grype, gitleaks, semgrep, govulncheck, gosec, staticcheck, hadolint, checkov, linters, schedule sql.NullInt64
	var healthEnabled, techDebt, reliability, maintainability, testGap, performance, aiRisk sql.NullInt64
	var codeGraph, graphIncludeFunctions, graphIncludeFindings sql.NullInt64
	var policy, workspace, severity, issue, remediation, runner, cron, ai sql.NullString
	var depth, healthMaxFindings, healthLargeFile, healthLargeFunc, healthNesting, healthParams sql.NullInt64
	var graphMaxNodes, graphMaxEdges, graphTimeout sql.NullInt64
	var govulncheckTimeout, gosecTimeout, staticcheckTimeout, goScannerMaxFindings sql.NullInt64
	var hadolintTimeout, checkovTimeout, iacScannerMaxFindings sql.NullInt64
	var confidence sql.NullFloat64
	var updatedAt string

	err := s.db.QueryRowContext(ctx, `
		SELECT repository_id, scan_profile, enabled, policy_level, workspace_mode, analysis_depth,
			enable_llm_auditors, enable_trivy, enable_grype, enable_gitleaks, enable_semgrep,
			enable_govulncheck, enable_gosec, enable_staticcheck, enable_linters,
			severity_gate, confidence_gate, issue_policy, remediation_policy, runner_policy,
			schedule_enabled, schedule_cron, ai_policy,
			enable_health_checks, enable_tech_debt_checks, enable_reliability_checks,
			enable_maintainability_checks, enable_test_gap_checks, enable_performance_checks, enable_ai_risk_checks,
			health_max_findings, health_large_file_lines, health_large_function_lines,
			health_max_nesting_depth, health_max_function_params,
			enable_code_graph, graph_max_nodes, graph_max_edges, graph_timeout_seconds,
			graph_include_functions, graph_include_findings,
			govulncheck_timeout_seconds, gosec_timeout_seconds, staticcheck_timeout_seconds, go_scanner_max_findings,
			enable_hadolint, enable_checkov, hadolint_timeout_seconds, checkov_timeout_seconds, iac_scanner_max_findings,
			notifications_enabled, notification_min_severity, notification_events, notification_cooldown_seconds,
			updated_at
		FROM repo_settings WHERE repository_id = ?
	`, repositoryID).Scan(
		&settings.RepositoryID,
		&scanProfile,
		&enabled,
		&policy,
		&workspace,
		&depth,
		&llm,
		&trivy,
		&grype,
		&gitleaks,
		&semgrep,
		&govulncheck,
		&gosec,
		&staticcheck,
		&linters,
		&severity,
		&confidence,
		&issue,
		&remediation,
		&runner,
		&schedule,
		&cron,
		&ai,
		&healthEnabled,
		&techDebt,
		&reliability,
		&maintainability,
		&testGap,
		&performance,
		&aiRisk,
		&healthMaxFindings,
		&healthLargeFile,
		&healthLargeFunc,
		&healthNesting,
		&healthParams,
		&codeGraph,
		&graphMaxNodes,
		&graphMaxEdges,
		&graphTimeout,
		&graphIncludeFunctions,
		&graphIncludeFindings,
		&govulncheckTimeout,
		&gosecTimeout,
		&staticcheckTimeout,
		&goScannerMaxFindings,
		&hadolint,
		&checkov,
		&hadolintTimeout,
		&checkovTimeout,
		&iacScannerMaxFindings,
		&notifEnabled,
		&notifMinSeverity,
		&notifEvents,
		&notifCooldown,
		&updatedAt,
	)
	if err == sql.ErrNoRows {
		return RepoSettings{RepositoryID: repositoryID}, nil
	}
	if err != nil {
		return RepoSettings{}, fmt.Errorf("get repo settings: %w", err)
	}

	settings.ScanProfile = nullStringPtr(scanProfile)
	settings.Enabled = nullBoolPtr(enabled)
	settings.PolicyLevel = nullStringPtr(policy)
	settings.WorkspaceMode = nullStringPtr(workspace)
	settings.AnalysisDepth = nullIntPtr(depth)
	settings.EnableLLMAuditors = nullBoolPtr(llm)
	settings.EnableTrivy = nullBoolPtr(trivy)
	settings.EnableGrype = nullBoolPtr(grype)
	settings.EnableGitleaks = nullBoolPtr(gitleaks)
	settings.EnableSemgrep = nullBoolPtr(semgrep)
	settings.EnableGovulncheck = nullBoolPtr(govulncheck)
	settings.EnableGosec = nullBoolPtr(gosec)
	settings.EnableStaticcheck = nullBoolPtr(staticcheck)
	settings.EnableHadolint = nullBoolPtr(hadolint)
	settings.EnableCheckov = nullBoolPtr(checkov)
	settings.EnableLinters = nullBoolPtr(linters)
	settings.SeverityGate = nullStringPtr(severity)
	settings.ConfidenceGate = nullFloatPtr(confidence)
	settings.IssuePolicy = nullStringPtr(issue)
	settings.RemediationPolicy = nullStringPtr(remediation)
	settings.RunnerPolicy = nullStringPtr(runner)
	settings.ScheduleEnabled = nullBoolPtr(schedule)
	settings.ScheduleCron = nullStringPtr(cron)
	settings.AIPolicy = nullStringPtr(ai)
	settings.EnableHealthChecks = nullBoolPtr(healthEnabled)
	settings.EnableTechDebtChecks = nullBoolPtr(techDebt)
	settings.EnableReliabilityChecks = nullBoolPtr(reliability)
	settings.EnableMaintainabilityChecks = nullBoolPtr(maintainability)
	settings.EnableTestGapChecks = nullBoolPtr(testGap)
	settings.EnablePerformanceChecks = nullBoolPtr(performance)
	settings.EnableAIRiskChecks = nullBoolPtr(aiRisk)
	settings.HealthMaxFindings = nullIntPtr(healthMaxFindings)
	settings.HealthLargeFileLines = nullIntPtr(healthLargeFile)
	settings.HealthLargeFunctionLines = nullIntPtr(healthLargeFunc)
	settings.HealthMaxNestingDepth = nullIntPtr(healthNesting)
	settings.HealthMaxFunctionParams = nullIntPtr(healthParams)
	settings.EnableCodeGraph = nullBoolPtr(codeGraph)
	settings.GraphMaxNodes = nullIntPtr(graphMaxNodes)
	settings.GraphMaxEdges = nullIntPtr(graphMaxEdges)
	settings.GraphTimeoutSeconds = nullIntPtr(graphTimeout)
	settings.GraphIncludeFunctions = nullBoolPtr(graphIncludeFunctions)
	settings.GraphIncludeFindings = nullBoolPtr(graphIncludeFindings)
	settings.GovulncheckTimeoutSeconds = nullIntPtr(govulncheckTimeout)
	settings.GosecTimeoutSeconds = nullIntPtr(gosecTimeout)
	settings.StaticcheckTimeoutSeconds = nullIntPtr(staticcheckTimeout)
	settings.GoScannerMaxFindings = nullIntPtr(goScannerMaxFindings)
	settings.HadolintTimeoutSeconds = nullIntPtr(hadolintTimeout)
	settings.CheckovTimeoutSeconds = nullIntPtr(checkovTimeout)
	settings.IACScannerMaxFindings = nullIntPtr(iacScannerMaxFindings)
	settings.NotificationsEnabled = nullBoolPtr(notifEnabled)
	settings.NotificationMinSeverity = nullStringPtr(notifMinSeverity)
	settings.NotificationEvents = nullStringPtr(notifEvents)
	settings.NotificationCooldownSeconds = nullIntPtr(notifCooldown)
	settings.UpdatedAt = parseTime(updatedAt)
	return settings, nil
}

func (s *SQLiteStore) SaveRepoSettings(ctx context.Context, settings RepoSettings) error {
	if settings.RepositoryID == 0 {
		return fmt.Errorf("repository_id is required")
	}
	now := time.Now().UTC()
	if settings.UpdatedAt.IsZero() {
		settings.UpdatedAt = now
	}

	_, err := s.db.ExecContext(ctx, `
		INSERT INTO repo_settings (
			repository_id, scan_profile, enabled, policy_level, workspace_mode, analysis_depth,
			enable_llm_auditors, enable_trivy, enable_grype, enable_gitleaks, enable_semgrep,
			enable_govulncheck, enable_gosec, enable_staticcheck, enable_linters,
			severity_gate, confidence_gate, issue_policy, remediation_policy, runner_policy,
			schedule_enabled, schedule_cron, ai_policy,
			enable_health_checks, enable_tech_debt_checks, enable_reliability_checks,
			enable_maintainability_checks, enable_test_gap_checks, enable_performance_checks, enable_ai_risk_checks,
			health_max_findings, health_large_file_lines, health_large_function_lines,
			health_max_nesting_depth, health_max_function_params,
			enable_code_graph, graph_max_nodes, graph_max_edges, graph_timeout_seconds,
			graph_include_functions, graph_include_findings,
			govulncheck_timeout_seconds, gosec_timeout_seconds, staticcheck_timeout_seconds, go_scanner_max_findings,
			enable_hadolint, enable_checkov, hadolint_timeout_seconds, checkov_timeout_seconds, iac_scanner_max_findings,
			notifications_enabled, notification_min_severity, notification_events, notification_cooldown_seconds,
			updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(repository_id) DO UPDATE SET
			scan_profile = excluded.scan_profile,
			enabled = excluded.enabled,
			policy_level = excluded.policy_level,
			workspace_mode = excluded.workspace_mode,
			analysis_depth = excluded.analysis_depth,
			enable_llm_auditors = excluded.enable_llm_auditors,
			enable_trivy = excluded.enable_trivy,
			enable_grype = excluded.enable_grype,
			enable_gitleaks = excluded.enable_gitleaks,
			enable_semgrep = excluded.enable_semgrep,
			enable_govulncheck = excluded.enable_govulncheck,
			enable_gosec = excluded.enable_gosec,
			enable_staticcheck = excluded.enable_staticcheck,
			enable_linters = excluded.enable_linters,
			severity_gate = excluded.severity_gate,
			confidence_gate = excluded.confidence_gate,
			issue_policy = excluded.issue_policy,
			remediation_policy = excluded.remediation_policy,
			runner_policy = excluded.runner_policy,
			schedule_enabled = excluded.schedule_enabled,
			schedule_cron = excluded.schedule_cron,
			ai_policy = excluded.ai_policy,
			enable_health_checks = excluded.enable_health_checks,
			enable_tech_debt_checks = excluded.enable_tech_debt_checks,
			enable_reliability_checks = excluded.enable_reliability_checks,
			enable_maintainability_checks = excluded.enable_maintainability_checks,
			enable_test_gap_checks = excluded.enable_test_gap_checks,
			enable_performance_checks = excluded.enable_performance_checks,
			enable_ai_risk_checks = excluded.enable_ai_risk_checks,
			health_max_findings = excluded.health_max_findings,
			health_large_file_lines = excluded.health_large_file_lines,
			health_large_function_lines = excluded.health_large_function_lines,
			health_max_nesting_depth = excluded.health_max_nesting_depth,
			health_max_function_params = excluded.health_max_function_params,
			enable_code_graph = excluded.enable_code_graph,
			graph_max_nodes = excluded.graph_max_nodes,
			graph_max_edges = excluded.graph_max_edges,
			graph_timeout_seconds = excluded.graph_timeout_seconds,
			graph_include_functions = excluded.graph_include_functions,
			graph_include_findings = excluded.graph_include_findings,
			govulncheck_timeout_seconds = excluded.govulncheck_timeout_seconds,
			gosec_timeout_seconds = excluded.gosec_timeout_seconds,
			staticcheck_timeout_seconds = excluded.staticcheck_timeout_seconds,
			go_scanner_max_findings = excluded.go_scanner_max_findings,
			enable_hadolint = excluded.enable_hadolint,
			enable_checkov = excluded.enable_checkov,
			hadolint_timeout_seconds = excluded.hadolint_timeout_seconds,
			checkov_timeout_seconds = excluded.checkov_timeout_seconds,
			iac_scanner_max_findings = excluded.iac_scanner_max_findings,
			notifications_enabled = excluded.notifications_enabled,
			notification_min_severity = excluded.notification_min_severity,
			notification_events = excluded.notification_events,
			notification_cooldown_seconds = excluded.notification_cooldown_seconds,
			updated_at = excluded.updated_at
	`,
		settings.RepositoryID,
		optionalString(settings.ScanProfile),
		optionalBool(settings.Enabled),
		optionalString(settings.PolicyLevel),
		optionalString(settings.WorkspaceMode),
		optionalInt(settings.AnalysisDepth),
		optionalBool(settings.EnableLLMAuditors),
		optionalBool(settings.EnableTrivy),
		optionalBool(settings.EnableGrype),
		optionalBool(settings.EnableGitleaks),
		optionalBool(settings.EnableSemgrep),
		optionalBool(settings.EnableGovulncheck),
		optionalBool(settings.EnableGosec),
		optionalBool(settings.EnableStaticcheck),
		optionalBool(settings.EnableLinters),
		optionalString(settings.SeverityGate),
		optionalFloat(settings.ConfidenceGate),
		optionalString(settings.IssuePolicy),
		optionalString(settings.RemediationPolicy),
		optionalString(settings.RunnerPolicy),
		optionalBool(settings.ScheduleEnabled),
		optionalString(settings.ScheduleCron),
		optionalString(settings.AIPolicy),
		optionalBool(settings.EnableHealthChecks),
		optionalBool(settings.EnableTechDebtChecks),
		optionalBool(settings.EnableReliabilityChecks),
		optionalBool(settings.EnableMaintainabilityChecks),
		optionalBool(settings.EnableTestGapChecks),
		optionalBool(settings.EnablePerformanceChecks),
		optionalBool(settings.EnableAIRiskChecks),
		optionalInt(settings.HealthMaxFindings),
		optionalInt(settings.HealthLargeFileLines),
		optionalInt(settings.HealthLargeFunctionLines),
		optionalInt(settings.HealthMaxNestingDepth),
		optionalInt(settings.HealthMaxFunctionParams),
		optionalBool(settings.EnableCodeGraph),
		optionalInt(settings.GraphMaxNodes),
		optionalInt(settings.GraphMaxEdges),
		optionalInt(settings.GraphTimeoutSeconds),
		optionalBool(settings.GraphIncludeFunctions),
		optionalBool(settings.GraphIncludeFindings),
		optionalInt(settings.GovulncheckTimeoutSeconds),
		optionalInt(settings.GosecTimeoutSeconds),
		optionalInt(settings.StaticcheckTimeoutSeconds),
		optionalInt(settings.GoScannerMaxFindings),
		optionalBool(settings.EnableHadolint),
		optionalBool(settings.EnableCheckov),
		optionalInt(settings.HadolintTimeoutSeconds),
		optionalInt(settings.CheckovTimeoutSeconds),
		optionalInt(settings.IACScannerMaxFindings),
		optionalBool(settings.NotificationsEnabled),
		optionalString(settings.NotificationMinSeverity),
		optionalString(settings.NotificationEvents),
		optionalInt(settings.NotificationCooldownSeconds),
		formatTime(settings.UpdatedAt),
	)
	if err != nil {
		return fmt.Errorf("save repo settings: %w", err)
	}
	return nil
}

func (s *SQLiteStore) CreateScan(ctx context.Context, scan Scan) (Scan, error) {
	if scan.ID == "" {
		return Scan{}, fmt.Errorf("scan id is required")
	}
	if scan.RepositoryID == 0 {
		return Scan{}, fmt.Errorf("repository_id is required")
	}
	if scan.Status == "" {
		scan.Status = ScanStatusStarted
	}
	if scan.StartedAt.IsZero() {
		scan.StartedAt = time.Now().UTC()
	}
	if scan.SummaryJSON == nil {
		scan.SummaryJSON = json.RawMessage(`{}`)
	}

	_, err := s.db.ExecContext(ctx, `
		INSERT INTO scans (
			id, repository_id, trigger_type, ref, commit_sha, pr_number,
			workspace_mode_used, commit_pinned, status, started_at, finished_at, summary_json, error
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`,
		scan.ID,
		scan.RepositoryID,
		scan.TriggerType,
		scan.Ref,
		scan.CommitSHA,
		scan.PRNumber,
		scan.WorkspaceModeUsed,
		boolToInt(scan.CommitPinned),
		scan.Status,
		formatTime(scan.StartedAt),
		nil,
		string(scan.SummaryJSON),
		scan.Error,
	)
	if err != nil {
		return Scan{}, fmt.Errorf("create scan: %w", err)
	}
	return s.GetScan(ctx, scan.ID)
}

func (s *SQLiteStore) FinishScan(ctx context.Context, scanID string, result ScanResult) error {
	if scanID == "" {
		return fmt.Errorf("scan id is required")
	}
	if result.Status == "" {
		result.Status = ScanStatusCompleted
	}
	if result.FinishedAt.IsZero() {
		result.FinishedAt = time.Now().UTC()
	}
	if result.SummaryJSON == nil {
		result.SummaryJSON = json.RawMessage(`{}`)
	}

	_, err := s.db.ExecContext(ctx, `
		UPDATE scans SET
			status = ?,
			finished_at = ?,
			summary_json = ?,
			error = ?,
			workspace_mode_used = CASE WHEN ? != '' THEN ? ELSE workspace_mode_used END,
			commit_pinned = ?,
			commit_sha = CASE WHEN ? != '' THEN ? ELSE commit_sha END
		WHERE id = ?
	`,
		result.Status,
		formatTime(result.FinishedAt),
		string(result.SummaryJSON),
		result.Error,
		result.WorkspaceModeUsed, result.WorkspaceModeUsed,
		boolToInt(result.CommitPinned),
		result.CommitSHA, result.CommitSHA,
		scanID,
	)
	if err != nil {
		return fmt.Errorf("finish scan: %w", err)
	}
	return nil
}

func (s *SQLiteStore) GetScan(ctx context.Context, scanID string) (Scan, error) {
	var scan Scan
	var commitPinned int
	var startedAt string
	var finishedAt sql.NullString
	var summary string

	err := s.db.QueryRowContext(ctx, `
		SELECT id, repository_id, trigger_type, ref, commit_sha, pr_number,
			workspace_mode_used, commit_pinned, status, started_at, finished_at, summary_json, error
		FROM scans WHERE id = ?
	`, scanID).Scan(
		&scan.ID,
		&scan.RepositoryID,
		&scan.TriggerType,
		&scan.Ref,
		&scan.CommitSHA,
		&scan.PRNumber,
		&scan.WorkspaceModeUsed,
		&commitPinned,
		&scan.Status,
		&startedAt,
		&finishedAt,
		&summary,
		&scan.Error,
	)
	if err != nil {
		return Scan{}, fmt.Errorf("get scan: %w", err)
	}
	scan.CommitPinned = intToBool(commitPinned)
	scan.StartedAt = parseTime(startedAt)
	scan.FinishedAt = parseTimePtr(finishedAt)
	scan.SummaryJSON = json.RawMessage(summary)
	return scan, nil
}

func (s *SQLiteStore) AddScannerResults(ctx context.Context, results []ScannerResultRecord) error {
	for _, result := range results {
		if result.ScanID == "" {
			return fmt.Errorf("scan_id is required for scanner result")
		}
		_, err := s.db.ExecContext(ctx, `
			INSERT INTO scanner_results (scan_id, scanner_name, status, findings_count, duration_ms, detail, error)
			VALUES (?, ?, ?, ?, ?, ?, ?)
		`, result.ScanID, result.ScannerName, result.Status, result.FindingsCount, nullInt64(result.DurationMS), result.Detail, result.Error)
		if err != nil {
			return fmt.Errorf("insert scanner result %s: %w", result.ScannerName, err)
		}
	}
	return nil
}

func nullInt64(v int64) interface{} {
	if v == 0 {
		return nil
	}
	return v
}

func (s *SQLiteStore) UpsertFinding(ctx context.Context, finding Finding) (Finding, error) {
	if finding.RepositoryID == 0 || finding.Fingerprint == "" {
		return Finding{}, fmt.Errorf("repository_id and fingerprint are required")
	}
	now := time.Now().UTC()
	if finding.FirstSeenAt.IsZero() {
		finding.FirstSeenAt = now
	}
	if finding.LastSeenAt.IsZero() {
		finding.LastSeenAt = now
	}
	if finding.Status == "" {
		finding.Status = FindingStatusOpen
	}

	existing, err := s.GetFindingByFingerprint(ctx, finding.RepositoryID, finding.Fingerprint)
	if err == nil {
		finding.ID = existing.ID
		finding.FirstSeenAt = existing.FirstSeenAt
		finding.FirstSeenScanID = existing.FirstSeenScanID
	} else if !isNoRows(err) {
		return Finding{}, err
	}

	if finding.FirstSeenScanID == "" {
		finding.FirstSeenScanID = finding.LastSeenScanID
	}

	_, err = s.db.ExecContext(ctx, `
		INSERT INTO findings (
			repository_id, fingerprint, category, severity, confidence, source, rule_id,
			package_name, file_path, line, title, status,
			first_seen_scan_id, last_seen_scan_id, first_seen_at, last_seen_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(repository_id, fingerprint) DO UPDATE SET
			category = excluded.category,
			severity = excluded.severity,
			confidence = excluded.confidence,
			source = excluded.source,
			rule_id = excluded.rule_id,
			package_name = excluded.package_name,
			file_path = excluded.file_path,
			line = excluded.line,
			title = excluded.title,
			status = excluded.status,
			last_seen_scan_id = excluded.last_seen_scan_id,
			last_seen_at = excluded.last_seen_at
	`, finding.RepositoryID, finding.Fingerprint, finding.Category, finding.Severity, finding.Confidence,
		finding.Source, finding.RuleID, finding.PackageName, finding.FilePath, finding.Line, finding.Title, finding.Status,
		finding.FirstSeenScanID, finding.LastSeenScanID, formatTime(finding.FirstSeenAt), formatTime(finding.LastSeenAt))
	if err != nil {
		return Finding{}, fmt.Errorf("upsert finding: %w", err)
	}

	return s.GetFindingByFingerprint(ctx, finding.RepositoryID, finding.Fingerprint)
}

func (s *SQLiteStore) GetFindingByFingerprint(ctx context.Context, repositoryID int64, fingerprint string) (Finding, error) {
	var finding Finding
	var firstSeen, lastSeen string
	err := s.db.QueryRowContext(ctx, `
		SELECT id, repository_id, fingerprint, category, severity, confidence, source, rule_id,
			package_name, file_path, line, title, status, first_seen_scan_id, last_seen_scan_id, first_seen_at, last_seen_at
		FROM findings WHERE repository_id = ? AND fingerprint = ?
	`, repositoryID, fingerprint).Scan(
		&finding.ID,
		&finding.RepositoryID,
		&finding.Fingerprint,
		&finding.Category,
		&finding.Severity,
		&finding.Confidence,
		&finding.Source,
		&finding.RuleID,
		&finding.PackageName,
		&finding.FilePath,
		&finding.Line,
		&finding.Title,
		&finding.Status,
		&finding.FirstSeenScanID,
		&finding.LastSeenScanID,
		&firstSeen,
		&lastSeen,
	)
	if err != nil {
		return Finding{}, err
	}
	finding.FirstSeenAt = parseTime(firstSeen)
	finding.LastSeenAt = parseTime(lastSeen)
	return finding, nil
}

func (s *SQLiteStore) AddFindingInstance(ctx context.Context, instance FindingInstance) error {
	if instance.FindingID == 0 || instance.ScanID == "" {
		return fmt.Errorf("finding_id and scan_id are required")
	}
	if instance.CreatedAt.IsZero() {
		instance.CreatedAt = time.Now().UTC()
	}
	if instance.LocationJSON == nil {
		instance.LocationJSON = json.RawMessage(`{}`)
	}
	if instance.RawMetadataJSON == nil {
		instance.RawMetadataJSON = json.RawMessage(`{}`)
	}

	_, err := s.db.ExecContext(ctx, `
		INSERT INTO finding_instances (finding_id, scan_id, evidence_redacted, location_json, raw_metadata_json, created_at)
		VALUES (?, ?, ?, ?, ?, ?)
	`, instance.FindingID, instance.ScanID, instance.EvidenceRedacted, string(instance.LocationJSON), string(instance.RawMetadataJSON), formatTime(instance.CreatedAt))
	if err != nil {
		return fmt.Errorf("add finding instance: %w", err)
	}
	return nil
}

func (s *SQLiteStore) UpsertExternalIssue(ctx context.Context, issue ExternalIssue) (ExternalIssue, error) {
	if issue.FindingID == 0 || issue.IssueNumber == 0 {
		return ExternalIssue{}, fmt.Errorf("finding_id and issue_number are required")
	}
	if issue.ForgeType == "" {
		issue.ForgeType = ForgeTypeGitea
	}
	now := time.Now().UTC()
	if issue.CreatedAt.IsZero() {
		issue.CreatedAt = now
	}
	if issue.UpdatedAt.IsZero() {
		issue.UpdatedAt = now
	}
	if issue.State == "" {
		issue.State = "open"
	}

	_, err := s.db.ExecContext(ctx, `
		INSERT INTO external_issues (finding_id, forge_type, issue_number, issue_url, state, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(finding_id, forge_type, issue_number) DO UPDATE SET
			issue_url = excluded.issue_url,
			state = excluded.state,
			updated_at = excluded.updated_at
	`, issue.FindingID, issue.ForgeType, issue.IssueNumber, issue.IssueURL, issue.State, formatTime(issue.CreatedAt), formatTime(issue.UpdatedAt))
	if err != nil {
		return ExternalIssue{}, fmt.Errorf("upsert external issue: %w", err)
	}

	err = s.db.QueryRowContext(ctx, `
		SELECT id, finding_id, forge_type, issue_number, issue_url, state, created_at, updated_at
		FROM external_issues WHERE finding_id = ? AND forge_type = ? AND issue_number = ?
	`, issue.FindingID, issue.ForgeType, issue.IssueNumber).Scan(
		&issue.ID,
		&issue.FindingID,
		&issue.ForgeType,
		&issue.IssueNumber,
		&issue.IssueURL,
		&issue.State,
		&formatTimeScan{&issue.CreatedAt},
		&formatTimeScan{&issue.UpdatedAt},
	)
	if err != nil {
		return ExternalIssue{}, fmt.Errorf("load external issue: %w", err)
	}
	return issue, nil
}

type formatTimeScan struct {
	target *time.Time
}

func (f *formatTimeScan) Scan(src any) error {
	switch v := src.(type) {
	case string:
		*f.target = parseTime(v)
	case []byte:
		*f.target = parseTime(string(v))
	case nil:
		*f.target = time.Time{}
	default:
		return fmt.Errorf("unsupported time type %T", src)
	}
	return nil
}

func (s *SQLiteStore) AddLifecycleEvent(ctx context.Context, event LifecycleEvent) error {
	if event.EventType == "" {
		return fmt.Errorf("event_type is required")
	}
	if event.CreatedAt.IsZero() {
		event.CreatedAt = time.Now().UTC()
	}
	if event.MetadataJSON == nil {
		event.MetadataJSON = json.RawMessage(`{}`)
	}

	var findingID interface{}
	if event.FindingID != nil {
		findingID = *event.FindingID
	}

	_, err := s.db.ExecContext(ctx, `
		INSERT INTO lifecycle_events (finding_id, scan_id, event_type, message, metadata_json, created_at)
		VALUES (?, ?, ?, ?, ?, ?)
	`, findingID, event.ScanID, event.EventType, event.Message, string(event.MetadataJSON), formatTime(event.CreatedAt))
	if err != nil {
		return fmt.Errorf("add lifecycle event: %w", err)
	}
	return nil
}
