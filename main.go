package main

import (
	"context"
	"crypto/hmac"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"sync/atomic"
	"syscall"
	"time"

	"git.commsnet.org/commstech/repository-detective/ai"
	"git.commsnet.org/commstech/repository-detective/analyzers"
	"git.commsnet.org/commstech/repository-detective/api"
	"git.commsnet.org/commstech/repository-detective/containers"
	"git.commsnet.org/commstech/repository-detective/docsdata"
	"git.commsnet.org/commstech/repository-detective/forge"
	"git.commsnet.org/commstech/repository-detective/gitea"
	"git.commsnet.org/commstech/repository-detective/github"
	"git.commsnet.org/commstech/repository-detective/graph"
	"git.commsnet.org/commstech/repository-detective/handlers"
	"git.commsnet.org/commstech/repository-detective/health"
	"git.commsnet.org/commstech/repository-detective/internal/config/envcompat"
	"git.commsnet.org/commstech/repository-detective/internal/middleware"
	"git.commsnet.org/commstech/repository-detective/internal/privacy"
	"git.commsnet.org/commstech/repository-detective/internal/scanid"
	"git.commsnet.org/commstech/repository-detective/internal/security"
	"git.commsnet.org/commstech/repository-detective/issues"
	"git.commsnet.org/commstech/repository-detective/limiter"
	"git.commsnet.org/commstech/repository-detective/openclaw"
	"git.commsnet.org/commstech/repository-detective/operator"
	"git.commsnet.org/commstech/repository-detective/orch"
	"git.commsnet.org/commstech/repository-detective/preinstall"
	"git.commsnet.org/commstech/repository-detective/profile"
	"git.commsnet.org/commstech/repository-detective/runner"
	"git.commsnet.org/commstech/repository-detective/scanners"
	"git.commsnet.org/commstech/repository-detective/store"
	"git.commsnet.org/commstech/repository-detective/ui"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

var version = "dev"
var commit = "unknown"
var buildDate = "unknown"

var componentsReady atomic.Bool

type scanProfileOverrideKey struct{}

type reportOnlyDryRunKey struct{}

var (
	logger              *logrus.Logger
	config              *Config
	giteaClient         *gitea.Client
	githubClient        forge.RepoClient
	aiClient            *ai.Client
	analysisEngine      *analyzers.Engine
	issueManager        *issues.Manager
	statusReporter      *gitea.StatusReporter
	webhookHandler      *handlers.WebhookHandler
	onboardingHandler   *handlers.OnboardingHandler
	analysisLimiter     *limiter.ConcurrencyLimiter
	rdStore             store.QueryStore
	scanRecorder        *store.Recorder
	controlPlaneHandler *api.Handler
	preinstallHandler   *api.PreinstallHandler
	preinstallRunner    *preinstall.Runner
	operatorUI          *ui.Handler
	scanScheduler       *orch.Scheduler
	schedulerCtx        context.Context
	schedulerCancel     context.CancelFunc
	appGlobalSnapshot   store.GlobalSettingsSnapshot
	runnerCfg           runner.Config
	runnerDispatcher    *runner.Dispatcher
	runnerReceiver      *runner.Receiver
	runnerRegistry      *runner.Registry
	runnerHandler       *api.RunnerHandler
)

// Config holds the plugin configuration
type Config struct {
	Port                                     string                               `mapstructure:"port"`
	APIKey                                   string                               `mapstructure:"api_key"` // API key for manual analysis endpoints
	GiteaURL                                 string                               `mapstructure:"gitea_url"`
	GiteaToken                               string                               `mapstructure:"gitea_token"`
	GitHubURL                                string                               `mapstructure:"github_url"`
	GitHubToken                              string                               `mapstructure:"github_token"`
	WebhookSecret                            string                               `mapstructure:"webhook_secret"`
	AllowInsecureWebhooks                    bool                                 `mapstructure:"allow_insecure_webhooks"`
	AIProvider                               string                               `mapstructure:"ai_provider"`
	AIBaseURL                                string                               `mapstructure:"ai_base_url"`
	AIAPIKey                                 string                               `mapstructure:"ai_api_key"`
	AIModel                                  string                               `mapstructure:"ai_model"`
	AIInsecureSkipTLSVerify                  bool                                 `mapstructure:"ai_insecure_skip_tls_verify"`
	OpenWebUIURL                             string                               `mapstructure:"openwebui_url"`
	OpenWebUIToken                           string                               `mapstructure:"openwebui_token"`
	OpenWebUIModel                           string                               `mapstructure:"openwebui_model"`
	LogLevel                                 string                               `mapstructure:"log_level"`
	AnalysisDepth                            int                                  `mapstructure:"analysis_depth"`
	MaxFileSize                              int64                                `mapstructure:"max_file_size"`
	EnableSecurity                           bool                                 `mapstructure:"enable_security"`
	EnableQuality                            bool                                 `mapstructure:"enable_quality"`
	EnableLLMAuditors                        bool                                 `mapstructure:"enable_llm_auditors"`
	EnableTrivy                              bool                                 `mapstructure:"enable_trivy"`
	EnableGrype                              bool                                 `mapstructure:"enable_grype"`
	EnableGitleaks                           bool                                 `mapstructure:"enable_gitleaks"`
	EnableSemgrep                            bool                                 `mapstructure:"enable_semgrep"`
	EnableGovulncheck                        bool                                 `mapstructure:"enable_govulncheck"`
	EnableGosec                              bool                                 `mapstructure:"enable_gosec"`
	EnableStaticcheck                        bool                                 `mapstructure:"enable_staticcheck"`
	EnableHadolint                           bool                                 `mapstructure:"enable_hadolint"`
	EnableCheckov                            bool                                 `mapstructure:"enable_checkov"`
	EnableLinters                            bool                                 `mapstructure:"enable_linters"`
	ScanProfile                              string                               `mapstructure:"scan_profile"`
	GitleaksConfig                           string                               `mapstructure:"gitleaks_config"`
	GitleaksTimeoutSeconds                   int                                  `mapstructure:"gitleaks_timeout_seconds"`
	SecretScanGitHistoryEnabled              bool                                 `mapstructure:"secret_scan_git_history_enabled"`
	SecretScanHistoryMaxCommits              int                                  `mapstructure:"secret_scan_history_max_commits"`
	SecretScanRecentCommitsMax               int                                  `mapstructure:"secret_scan_recent_commits_max"`
	SecretScanHistoryTimeoutSeconds          int                                  `mapstructure:"secret_scan_history_timeout_seconds"`
	SecretScanHistoryReportOnlyForPreinstall bool                                 `mapstructure:"secret_scan_history_report_only_for_preinstall"`
	SecretScanRedact                         bool                                 `mapstructure:"secret_scan_redact"`
	SemgrepConfig                            string                               `mapstructure:"semgrep_config"`
	SemgrepTimeoutSeconds                    int                                  `mapstructure:"semgrep_timeout_seconds"`
	SemgrepMaxFindings                       int                                  `mapstructure:"semgrep_max_findings"`
	SemgrepSeverityThreshold                 string                               `mapstructure:"semgrep_severity_threshold"`
	GovulncheckTimeoutSeconds                int                                  `mapstructure:"govulncheck_timeout_seconds"`
	GosecTimeoutSeconds                      int                                  `mapstructure:"gosec_timeout_seconds"`
	StaticcheckTimeoutSeconds                int                                  `mapstructure:"staticcheck_timeout_seconds"`
	GoScannerMaxFindings                     int                                  `mapstructure:"go_scanner_max_findings"`
	HadolintTimeoutSeconds                   int                                  `mapstructure:"hadolint_timeout_seconds"`
	CheckovTimeoutSeconds                    int                                  `mapstructure:"checkov_timeout_seconds"`
	IACScannerMaxFindings                    int                                  `mapstructure:"iac_scanner_max_findings"`
	ScannerTimeoutSeconds                    int                                  `mapstructure:"scanner_timeout_seconds"`
	MinIssueConfidence                       float64                              `mapstructure:"min_issue_confidence"`
	AutoCreateIssues                         bool                                 `mapstructure:"auto_create_issues"`
	MaxIssuesPerRun                          int                                  `mapstructure:"max_issues_per_run"`
	SkipLowSeverity                          bool                                 `mapstructure:"skip_low_severity"`
	GroupSimilarIssues                       bool                                 `mapstructure:"group_similar_issues"`
	SkipPatterns                             []string                             `mapstructure:"-"`
	LanguageMapping                          map[string]string                    `mapstructure:"-"`
	RepositoryIncludePatterns                []string                             `mapstructure:"-"`
	RepositoryExcludePatterns                []string                             `mapstructure:"-"`
	PublicURL                                string                               `mapstructure:"public_url"`
	ListenHost                               string                               `mapstructure:"listen_host"`
	StartupCheckTimeout                      int                                  `mapstructure:"startup_check_timeout"`
	MaxConcurrentAnalyses                    int                                  `mapstructure:"max_concurrent_analyses"`
	AnalysisTimeout                          int                                  `mapstructure:"analysis_timeout"`
	ScanCooldownSeconds                      int                                  `mapstructure:"scan_cooldown_seconds"`
	RateLimitPerMinute                       int                                  `mapstructure:"rate_limit_per_minute"`
	WorkspaceMode                            string                               `mapstructure:"workspace_mode"`
	WorkspaceMaxSizeMB                       int                                  `mapstructure:"workspace_max_size_mb"`
	WorkspaceMaxFiles                        int                                  `mapstructure:"workspace_max_files"`
	WorkspaceArchiveTimeoutSeconds           int                                  `mapstructure:"workspace_archive_timeout_seconds"`
	EnableGiteaStatus                        bool                                 `mapstructure:"enable_gitea_status"`
	GiteaStatusContext                       string                               `mapstructure:"gitea_status_context"`
	GiteaStatusFailOn                        string                               `mapstructure:"gitea_status_fail_on"`
	GiteaStatusWarnOn                        string                               `mapstructure:"gitea_status_warn_on"`
	GiteaStatusIncludeScannerFailures        bool                                 `mapstructure:"gitea_status_include_scanner_failures"`
	SkipStartupChecks                        bool                                 `mapstructure:"skip_startup_checks"`
	DatabaseEnabled                          bool                                 `mapstructure:"database_enabled"`
	DatabaseDriver                           string                               `mapstructure:"database_driver"`
	DatabasePath                             string                               `mapstructure:"database_path"`
	DatabaseDSN                              string                               `mapstructure:"database_dsn"`
	UIEnabled                                bool                                 `mapstructure:"ui_enabled"`
	UIBasePath                               string                               `mapstructure:"ui_base_path"`
	SchedulerEnabled                         bool                                 `mapstructure:"scheduler_enabled"`
	SchedulerPollIntervalSeconds             int                                  `mapstructure:"scheduler_poll_interval_seconds"`
	SchedulerMaxConcurrentScans              int                                  `mapstructure:"scheduler_max_concurrent_scans"`
	PreinstallAuditEnabled                   bool                                 `mapstructure:"preinstall_audit_enabled"`
	PreinstallAllowPrivateNetworks           bool                                 `mapstructure:"preinstall_allow_private_networks"`
	PreinstallMaxRepoSizeMB                  int                                  `mapstructure:"preinstall_max_repo_size_mb"`
	PreinstallMaxFiles                       int                                  `mapstructure:"preinstall_max_files"`
	PreinstallTimeoutSeconds                 int                                  `mapstructure:"preinstall_timeout_seconds"`
	PreinstallMaxFindings                    int                                  `mapstructure:"preinstall_max_findings"`
	PreinstallAllowGitClone                  bool                                 `mapstructure:"preinstall_allow_git_clone"`
	PreinstallReportIncludeProjectLink       bool                                 `mapstructure:"preinstall_report_include_project_link"`
	PreinstallSandboxEnabled                 bool                                 `mapstructure:"preinstall_sandbox_enabled"`
	PreinstallSandboxRetainOnFailure         bool                                 `mapstructure:"preinstall_sandbox_retain_on_failure"`
	PreinstallSandboxMaxFileSizeMB           int                                  `mapstructure:"preinstall_sandbox_max_file_size_mb"`
	PreinstallSandboxAllowSubmodules         bool                                 `mapstructure:"preinstall_sandbox_allow_submodules"`
	PreinstallSandboxNetworkMode             string                               `mapstructure:"preinstall_sandbox_network_mode"`
	PreinstallSandboxReadonlyWorkspace       bool                                 `mapstructure:"preinstall_sandbox_readonly_workspace"`
	RepositoryDetectiveProjectURL            string                               `mapstructure:"repository_detective_project_url"`
	EnableHealthChecks                       bool                                 `mapstructure:"enable_health_checks"`
	EnableTechDebtChecks                     bool                                 `mapstructure:"enable_tech_debt_checks"`
	EnableReliabilityChecks                  bool                                 `mapstructure:"enable_reliability_checks"`
	EnableMaintainabilityChecks              bool                                 `mapstructure:"enable_maintainability_checks"`
	EnableTestGapChecks                      bool                                 `mapstructure:"enable_test_gap_checks"`
	EnablePerformanceChecks                  bool                                 `mapstructure:"enable_performance_checks"`
	EnableAIRiskChecks                       bool                                 `mapstructure:"enable_ai_risk_checks"`
	HealthMaxFindings                        int                                  `mapstructure:"health_max_findings"`
	HealthLargeFileLines                     int                                  `mapstructure:"health_large_file_lines"`
	HealthLargeFunctionLines                 int                                  `mapstructure:"health_large_function_lines"`
	HealthMaxNestingDepth                    int                                  `mapstructure:"health_max_nesting_depth"`
	HealthMaxFunctionParams                  int                                  `mapstructure:"health_max_function_params"`
	EnableCodeGraph                          bool                                 `mapstructure:"enable_code_graph"`
	GraphMaxNodes                            int                                  `mapstructure:"graph_max_nodes"`
	GraphMaxEdges                            int                                  `mapstructure:"graph_max_edges"`
	GraphTimeoutSeconds                      int                                  `mapstructure:"graph_timeout_seconds"`
	GraphIncludeFunctions                    bool                                 `mapstructure:"graph_include_functions"`
	GraphIncludeFindings                     bool                                 `mapstructure:"graph_include_findings"`
	RunnerDelegationEnabled                  bool                                 `mapstructure:"runner_delegation_enabled"`
	RunnerMode                               string                               `mapstructure:"runner_mode"`
	RunnerSharedSecret                       string                               `mapstructure:"runner_shared_secret"`
	RunnerJobTimeoutSeconds                  int                                  `mapstructure:"runner_job_timeout_seconds"`
	RunnerMaxConcurrentJobs                  int                                  `mapstructure:"runner_max_concurrent_jobs"`
	RunnerResultMaxSizeMB                    int                                  `mapstructure:"runner_result_max_size_mb"`
	RunnerArtifactRetentionDays              int                                  `mapstructure:"runner_artifact_retention_days"`
	RunnerCallbackBaseURL                    string                               `mapstructure:"runner_callback_base_url"`
	RunnerRequireHMAC                        bool                                 `mapstructure:"runner_require_hmac"`
	RunnerNonceTTLSeconds                    int                                  `mapstructure:"runner_nonce_ttl_seconds"`
	RunnerAllowedJobTypes                    []string                             `mapstructure:"runner_allowed_job_types"`
	GiteaActionsTestBackendEnabled           bool                                 `mapstructure:"gitea_actions_test_backend_enabled"`
	GiteaActionsWorkflowName                 string                               `mapstructure:"gitea_actions_workflow_name"`
	GiteaActionsTriggerMode                  string                               `mapstructure:"gitea_actions_trigger_mode"`
	GiteaActionsTimeoutSeconds               int                                  `mapstructure:"gitea_actions_timeout_seconds"`
	GiteaActionsRequireOperatorApproval      bool                                 `mapstructure:"gitea_actions_require_operator_approval"`
	NotificationsEnabled                     bool                                 `mapstructure:"notifications_enabled"`
	NotificationMinSeverity                  string                               `mapstructure:"notification_min_severity"`
	NotificationCooldownSeconds              int                                  `mapstructure:"notification_cooldown_seconds"`
	TelegramEnabled                          bool                                 `mapstructure:"telegram_enabled"`
	TelegramBotToken                         string                               `mapstructure:"telegram_bot_token"`
	TelegramChatID                           string                               `mapstructure:"telegram_chat_id"`
	SlackEnabled                             bool                                 `mapstructure:"slack_enabled"`
	SlackWebhookURL                          string                               `mapstructure:"slack_webhook_url"`
	DiscordEnabled                           bool                                 `mapstructure:"discord_enabled"`
	DiscordWebhookURL                        string                               `mapstructure:"discord_webhook_url"`
	WebhookNotificationsEnabled              bool                                 `mapstructure:"webhook_notifications_enabled"`
	WebhookNotificationURL                   string                               `mapstructure:"webhook_notification_url"`
	WebhookNotificationSecret                string                               `mapstructure:"webhook_notification_secret"`
	RemediationPlannerEnabled                bool                                 `mapstructure:"remediation_planner_enabled"`
	RemediationMinSeverity                   string                               `mapstructure:"remediation_min_severity"`
	RemediationMinConfidence                 float64                              `mapstructure:"remediation_min_confidence"`
	RemediationUseAI                         bool                                 `mapstructure:"remediation_use_ai"`
	RemediationCommentOnIssue                bool                                 `mapstructure:"remediation_comment_on_issue"`
	RemediationPREnabled                     bool                                 `mapstructure:"remediation_pr_enabled"`
	RemediationPRBranchPrefix                string                               `mapstructure:"remediation_pr_branch_prefix"`
	RemediationPRRequireApproval             bool                                 `mapstructure:"remediation_pr_require_approval"`
	RemediationPRMaxFilesChanged             int                                  `mapstructure:"remediation_pr_max_files_changed"`
	RemediationPRMaxDiffLines                int                                  `mapstructure:"remediation_pr_max_diff_lines"`
	RemediationPRValidationTimeoutSeconds    int                                  `mapstructure:"remediation_pr_validation_timeout_seconds"`
	RemediationPRRequireTests                bool                                 `mapstructure:"remediation_pr_require_tests"`
	RemediationPRUseRunnerVerification       bool                                 `mapstructure:"remediation_pr_use_runner_verification"`
	RemediationPRBlockHighCritical           bool                                 `mapstructure:"remediation_pr_block_high_critical_without_manual_override"`
	RemediationPRAllowedSeverities           []string                             `mapstructure:"remediation_pr_allowed_severities"`
	EvidenceClosureEnabled                   bool                                 `mapstructure:"evidence_closure_enabled"`
	EvidenceClosureCloseIssues               bool                                 `mapstructure:"evidence_closure_close_issues"`
	EvidenceClosureComment                   bool                                 `mapstructure:"evidence_closure_comment"`
	EvidenceClosureRequireScannerSuccess     bool                                 `mapstructure:"evidence_closure_require_scanner_success"`
	IssueReconciliationEnabled               bool                                 `mapstructure:"issue_reconciliation_enabled"`
	IssueReconciliationComment               bool                                 `mapstructure:"issue_reconciliation_comment"`
	IssueReconciliationCloseVerified         bool                                 `mapstructure:"issue_reconciliation_close_verified"`
	IssueReconciliationMaxCommentsPerIssue   int                                  `mapstructure:"issue_reconciliation_max_comments_per_issue"`
	IssueReconciliationCloseDuplicates       bool                                 `mapstructure:"issue_reconciliation_close_duplicates"`
	DogfoodBacklogControlEnabled             bool                                 `mapstructure:"dogfood_backlog_control_enabled"`
	DogfoodBacklogMaxOpenIssues              int                                  `mapstructure:"dogfood_backlog_max_open_issues"`
	DogfoodBacklogAllowNewIssueSeverity      []string                             `mapstructure:"dogfood_backlog_allow_new_issue_severity"`
	DogfoodBacklogAllowNewIssueConfidence    string                               `mapstructure:"dogfood_backlog_allow_new_issue_confidence"`
	DogfoodBacklogUpdateExistingOnly         bool                                 `mapstructure:"dogfood_backlog_update_existing_only"`
	AIStartupTestEnabled                     bool                                 `mapstructure:"ai_startup_test_enabled"`
	AIConnectionTestMode                     string                               `mapstructure:"ai_connection_test_mode"`
	AIConnectionTestCacheMinutes             int                                  `mapstructure:"ai_connection_test_cache_minutes"`
	AIMaxTokensPerScan                       int                                  `mapstructure:"ai_max_tokens_per_scan"`
	CalibrationEnabled                       bool                                 `mapstructure:"calibration_enabled"`
	CalibrationIntervalHours                 int                                  `mapstructure:"calibration_interval_hours"`
	CalibrationMinFindingsForRecommendation  int                                  `mapstructure:"calibration_min_findings_for_recommendation"`
	CalibrationAutoApply                     bool                                 `mapstructure:"calibration_auto_apply"`
	LLMSanityGateEnabled                     bool                                 `mapstructure:"llm_sanity_gate_enabled"`
	LLMSanityGateMaxTokensPerScan            int                                  `mapstructure:"llm_sanity_gate_max_tokens_per_scan"`
	LLMSanityGateApplyActions                bool                                 `mapstructure:"llm_sanity_gate_apply_actions"`
	LLMSanityGateLowMediumOnly               bool                                 `mapstructure:"llm_sanity_gate_low_medium_only"`
	Reporting                                profile.ReportingConfig              `mapstructure:"reporting"`
	FalsePositiveReduction                   profile.FalsePositiveReductionConfig `mapstructure:"false_positive_reduction"`
	ContainerScan                            containers.Config                    `mapstructure:",squash"`
	OpenClawAIReview                         openclaw.Config                      `mapstructure:",squash"`
	AuthMode                                 string                               `mapstructure:"auth_mode"`
	SessionCookieName                        string                               `mapstructure:"session_cookie_name"`
	SessionSecret                            string                               `mapstructure:"session_secret"`
	SessionTTLHours                          int                                  `mapstructure:"session_ttl_hours"`
	CSRFEnabled                              bool                                 `mapstructure:"csrf_enabled"`
	LocalAdminBootstrapEnabled               bool                                 `mapstructure:"local_admin_bootstrap_enabled"`
	RejectQueryStringAPIKey                  bool                                 `mapstructure:"reject_query_string_api_key"`
	WarnQueryStringAPIKey                    bool                                 `mapstructure:"warn_query_string_api_key"`
	PrivacyMode                              string                               `mapstructure:"privacy_mode"`
}

func main() {
	// Always log to stdout so Docker/systemd capture output even when file logging fails.
	logger = logrus.New()
	logger.SetOutput(os.Stdout)
	logger.SetFormatter(&logrus.TextFormatter{
		FullTimestamp: true,
	})

	maybeRunDoctorCLI()

	logger.Info("Repository Detective starting...")

	// Load configuration
	if err := loadConfig(); err != nil {
		logger.Fatalf("Failed to load configuration: %v", err)
	}

	// Set log level
	level, err := logrus.ParseLevel(config.LogLevel)
	if err != nil {
		logger.Warnf("Invalid log level %s, using info: %v", config.LogLevel, err)
		level = logrus.InfoLevel
	}
	logger.SetLevel(level)

	logger.Info("Starting Repository Detective...")
	logger.Infof("Configuration loaded: Port=%s, ListenHost=%s, GiteaURL=%s, AIProvider=%s, SkipStartupChecks=%v",
		config.Port, config.ListenHost, config.GiteaURL, config.effectiveAIProvider(), config.SkipStartupChecks)

	// Initialize Gin router and bind HTTP before blocking startup checks.
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	router.Use(middleware.RedactingAccessLogger(), gin.Recovery())
	setupRoutes(router)

	listenAddr := config.ListenHost + ":" + config.Port
	server := &http.Server{
		Addr:         listenAddr,
		Handler:      router,
		ReadTimeout:  120 * time.Second,
		WriteTimeout: 120 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	serverErr := make(chan error, 1)
	go func() {
		logger.Infof("Listening on %s (health available immediately)", listenAddr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			serverErr <- err
		}
	}()

	// Initialize components (may block on Gitea/AI checks — health still responds).
	if err := initializeComponents(); err != nil {
		logger.Fatalf("Failed to initialize components: %v", err)
	}
	registerControlPlaneRoutes(router)
	componentsReady.Store(true)
	logger.Info("Repository Detective ready — all components initialized")

	// Wait for interrupt signal or unexpected server failure.
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	select {
	case err := <-serverErr:
		logger.Fatalf("HTTP server failed: %v", err)
	case <-quit:
	}

	logger.Info("Shutting down server...")

	if scanScheduler != nil {
		scanScheduler.Stop()
	}
	if schedulerCancel != nil {
		schedulerCancel()
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		logger.Fatalf("Server forced to shutdown: %v", err)
	}

	logger.Info("Server exited")
}

func buildBacklogControlConfig(cfg Config) issues.BacklogControlConfig {
	bc := issues.DefaultBacklogControlConfig()
	bc.Enabled = cfg.DogfoodBacklogControlEnabled
	bc.MaxOpenIssues = cfg.DogfoodBacklogMaxOpenIssues
	bc.UpdateExistingOnly = cfg.DogfoodBacklogUpdateExistingOnly
	if len(cfg.DogfoodBacklogAllowNewIssueSeverity) > 0 {
		bc.AllowNewIssueSeverity = cfg.DogfoodBacklogAllowNewIssueSeverity
	}
	level := strings.TrimSpace(cfg.DogfoodBacklogAllowNewIssueConfidence)
	if level != "" {
		switch strings.ToLower(level) {
		case "critical":
			bc.AllowMinConfidence = 0.95
		case "high":
			bc.AllowMinConfidence = 0.85
		case "medium":
			bc.AllowMinConfidence = 0.70
		case "low":
			bc.AllowMinConfidence = 0.50
		default:
			if f, err := strconv.ParseFloat(level, 64); err == nil && f > 0 && f <= 1 {
				bc.AllowMinConfidence = f
			}
		}
	}
	return bc
}

func loadConfig() error {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")
	viper.AddConfigPath("./config")

	// Set defaults
	viper.SetDefault("port", "8080")
	viper.SetDefault("listen_host", "0.0.0.0")
	viper.SetDefault("log_level", "info")
	viper.SetDefault("startup_check_timeout", 10)
	viper.SetDefault("analysis_depth", 3)
	viper.SetDefault("max_file_size", 1024*1024) // 1MB
	viper.SetDefault("enable_security", true)
	viper.SetDefault("enable_quality", true)
	viper.SetDefault("enable_llm_auditors", false)
	viper.SetDefault("enable_trivy", true)
	viper.SetDefault("enable_grype", true)
	viper.SetDefault("enable_gitleaks", false)
	viper.SetDefault("enable_semgrep", false)
	viper.SetDefault("enable_govulncheck", false)
	viper.SetDefault("enable_gosec", false)
	viper.SetDefault("enable_staticcheck", false)
	viper.SetDefault("enable_hadolint", false)
	viper.SetDefault("enable_checkov", false)
	viper.SetDefault("enable_linters", true)
	viper.SetDefault("scan_profile", "custom")
	viper.SetDefault("gitleaks_config", "")
	viper.SetDefault("gitleaks_timeout_seconds", 0)
	viper.SetDefault("secret_scan_git_history_enabled", true)
	viper.SetDefault("secret_scan_history_max_commits", 0)
	viper.SetDefault("secret_scan_recent_commits_max", 50)
	viper.SetDefault("secret_scan_history_timeout_seconds", 600)
	viper.SetDefault("secret_scan_history_report_only_for_preinstall", true)
	viper.SetDefault("secret_scan_redact", true)
	viper.SetDefault("semgrep_config", "p/ci")
	viper.SetDefault("semgrep_timeout_seconds", 0)
	viper.SetDefault("semgrep_max_findings", 100)
	viper.SetDefault("semgrep_severity_threshold", "INFO")
	viper.SetDefault("govulncheck_timeout_seconds", 0)
	viper.SetDefault("gosec_timeout_seconds", 0)
	viper.SetDefault("staticcheck_timeout_seconds", 0)
	viper.SetDefault("go_scanner_max_findings", 100)
	viper.SetDefault("hadolint_timeout_seconds", 0)
	viper.SetDefault("checkov_timeout_seconds", 0)
	viper.SetDefault("iac_scanner_max_findings", 100)
	viper.SetDefault("scanner_timeout_seconds", 180)
	viper.SetDefault("min_issue_confidence", 0.5)
	viper.SetDefault("auto_create_issues", true)
	viper.SetDefault("max_issues_per_run", 50)
	viper.SetDefault("max_concurrent_analyses", 5)
	viper.SetDefault("analysis_timeout", 900)
	viper.SetDefault("scan_cooldown_seconds", 1800)
	viper.SetDefault("rate_limit_per_minute", 60)
	viper.SetDefault("openwebui_model", "default")
	viper.SetDefault("ai_provider", "")
	viper.SetDefault("ai_model", "")
	viper.SetDefault("ai_insecure_skip_tls_verify", false)
	viper.SetDefault("skip_startup_checks", false)
	viper.SetDefault("github_url", "https://api.github.com")
	viper.SetDefault("workspace_mode", "api")
	viper.SetDefault("workspace_max_size_mb", 500)
	viper.SetDefault("workspace_max_files", 5000)
	viper.SetDefault("workspace_archive_timeout_seconds", 0)
	viper.SetDefault("enable_gitea_status", false)
	viper.SetDefault("gitea_status_context", "repository-detective/security-scan")
	viper.SetDefault("gitea_status_fail_on", "high")
	viper.SetDefault("gitea_status_warn_on", "medium")
	viper.SetDefault("gitea_status_include_scanner_failures", true)
	viper.SetDefault("database_enabled", true)
	viper.SetDefault("database_driver", "sqlite")
	viper.SetDefault("database_path", "./data/repository-detective.db")
	viper.SetDefault("database_dsn", "")
	viper.SetDefault("ui_enabled", true)
	viper.SetDefault("ui_base_path", "/ui")
	viper.SetDefault("scheduler_enabled", true)
	viper.SetDefault("scheduler_poll_interval_seconds", 60)
	viper.SetDefault("scheduler_max_concurrent_scans", 1)
	viper.SetDefault("preinstall_audit_enabled", true)
	viper.SetDefault("preinstall_allow_private_networks", false)
	viper.SetDefault("preinstall_max_repo_size_mb", 500)
	viper.SetDefault("preinstall_max_files", 5000)
	viper.SetDefault("preinstall_timeout_seconds", 600)
	viper.SetDefault("preinstall_max_findings", 200)
	viper.SetDefault("preinstall_allow_git_clone", true)
	viper.SetDefault("preinstall_report_include_project_link", true)
	viper.SetDefault("preinstall_sandbox_enabled", true)
	viper.SetDefault("preinstall_sandbox_retain_on_failure", false)
	viper.SetDefault("preinstall_sandbox_max_file_size_mb", 25)
	viper.SetDefault("preinstall_sandbox_allow_submodules", false)
	viper.SetDefault("preinstall_sandbox_network_mode", "restricted")
	viper.SetDefault("preinstall_sandbox_readonly_workspace", true)
	viper.SetDefault("repository_detective_project_url", "https://git.commsnet.org/commstech/Repository-Detective")
	viper.SetDefault("enable_health_checks", true)
	viper.SetDefault("enable_tech_debt_checks", true)
	viper.SetDefault("enable_reliability_checks", true)
	viper.SetDefault("enable_maintainability_checks", true)
	viper.SetDefault("enable_test_gap_checks", true)
	viper.SetDefault("enable_performance_checks", true)
	viper.SetDefault("enable_ai_risk_checks", false)
	viper.SetDefault("health_max_findings", 100)
	viper.SetDefault("health_large_file_lines", 1000)
	viper.SetDefault("health_large_function_lines", 150)
	viper.SetDefault("health_max_nesting_depth", 5)
	viper.SetDefault("health_max_function_params", 7)
	viper.SetDefault("enable_code_graph", true)
	viper.SetDefault("graph_max_nodes", 5000)
	viper.SetDefault("graph_max_edges", 15000)
	viper.SetDefault("graph_timeout_seconds", 120)
	viper.SetDefault("graph_include_functions", true)
	viper.SetDefault("graph_include_findings", true)
	viper.SetDefault("runner_delegation_enabled", false)
	viper.SetDefault("runner_mode", "core")
	viper.SetDefault("runner_job_timeout_seconds", 900)
	viper.SetDefault("runner_max_concurrent_jobs", 2)
	viper.SetDefault("runner_result_max_size_mb", 50)
	viper.SetDefault("runner_artifact_retention_days", 14)
	viper.SetDefault("runner_require_hmac", true)
	viper.SetDefault("runner_nonce_ttl_seconds", 300)
	viper.SetDefault("runner_allowed_job_types", []string{"scan", "sbom", "graph", "preinstall_audit", "remediation_verify", "container_image_scan"})
	defContainer := containers.DefaultConfig()
	viper.SetDefault("container_scanning_enabled", defContainer.Enabled)
	viper.SetDefault("container_scan_default_policy", defContainer.DefaultPolicy)
	viper.SetDefault("container_scan_create_issues", defContainer.CreateIssues)
	viper.SetDefault("container_scan_require_runner", defContainer.RequireRunner)
	viper.SetDefault("container_scan_allow_core_docker_socket", defContainer.AllowCoreDockerSocket)
	viper.SetDefault("container_scan_pull_policy", string(defContainer.PullPolicy))
	viper.SetDefault("container_scan_timeout_seconds", defContainer.TimeoutSeconds)
	viper.SetDefault("container_scan_max_image_size_mb", defContainer.MaxImageSizeMB)
	viper.SetDefault("container_scan_generate_sbom", defContainer.GenerateSBOM)
	viper.SetDefault("container_scan_fail_on_scanner_missing", defContainer.FailOnScannerMissing)
	viper.SetDefault("container_scan_allowed_runner_labels", defContainer.AllowedRunnerLabels)
	viper.SetDefault("gitea_actions_test_backend_enabled", false)
	viper.SetDefault("gitea_actions_workflow_name", "repository-detective-verify.yml")
	viper.SetDefault("gitea_actions_trigger_mode", "workflow_dispatch")
	viper.SetDefault("gitea_actions_timeout_seconds", 1800)
	viper.SetDefault("gitea_actions_require_operator_approval", true)
	viper.SetDefault("notifications_enabled", false)
	viper.SetDefault("notification_min_severity", "high")
	viper.SetDefault("notification_cooldown_seconds", 300)
	viper.SetDefault("telegram_enabled", false)
	viper.SetDefault("slack_enabled", false)
	viper.SetDefault("discord_enabled", false)
	viper.SetDefault("webhook_notifications_enabled", false)
	viper.SetDefault("remediation_planner_enabled", true)
	viper.SetDefault("remediation_min_severity", "medium")
	viper.SetDefault("remediation_min_confidence", 0.80)
	viper.SetDefault("remediation_use_ai", false)
	viper.SetDefault("remediation_comment_on_issue", false)
	viper.SetDefault("remediation_pr_enabled", false)
	viper.SetDefault("remediation_pr_branch_prefix", "repository-detective/fix")
	viper.SetDefault("remediation_pr_require_approval", true)
	viper.SetDefault("remediation_pr_max_files_changed", 3)
	viper.SetDefault("remediation_pr_max_diff_lines", 100)
	viper.SetDefault("remediation_pr_validation_timeout_seconds", 300)
	viper.SetDefault("remediation_pr_require_tests", true)
	viper.SetDefault("remediation_pr_use_runner_verification", true)
	viper.SetDefault("remediation_pr_block_high_critical_without_manual_override", true)
	viper.SetDefault("remediation_pr_allowed_severities", []string{"low", "medium"})
	viper.SetDefault("evidence_closure_enabled", true)
	viper.SetDefault("evidence_closure_close_issues", false)
	viper.SetDefault("evidence_closure_comment", true)
	viper.SetDefault("evidence_closure_require_scanner_success", true)
	viper.SetDefault("issue_reconciliation_enabled", true)
	viper.SetDefault("issue_reconciliation_comment", true)
	viper.SetDefault("issue_reconciliation_close_verified", false)
	viper.SetDefault("issue_reconciliation_close_duplicates", false)
	viper.SetDefault("issue_reconciliation_max_comments_per_issue", 3)
	viper.SetDefault("dogfood_backlog_control_enabled", false)
	viper.SetDefault("dogfood_backlog_max_open_issues", 0)
	viper.SetDefault("dogfood_backlog_allow_new_issue_severity", []string{"high", "critical"})
	viper.SetDefault("dogfood_backlog_allow_new_issue_confidence", "high")
	viper.SetDefault("dogfood_backlog_update_existing_only", true)
	viper.SetDefault("ai_startup_test_enabled", false)
	viper.SetDefault("ai_connection_test_mode", "metadata_only")
	viper.SetDefault("ai_connection_test_cache_minutes", 60)
	viper.SetDefault("ai_max_tokens_per_scan", 0)
	viper.SetDefault("calibration_enabled", true)
	viper.SetDefault("calibration_interval_hours", 24)
	viper.SetDefault("calibration_min_findings_for_recommendation", 20)
	viper.SetDefault("calibration_auto_apply", false)
	viper.SetDefault("llm_sanity_gate_enabled", false)
	viper.SetDefault("llm_sanity_gate_max_tokens_per_scan", 0)
	viper.SetDefault("llm_sanity_gate_apply_actions", false)
	viper.SetDefault("llm_sanity_gate_low_medium_only", true)
	defOpenClaw := openclaw.DefaultConfig()
	viper.SetDefault("openclaw_ai_review_enabled", defOpenClaw.Enabled)
	viper.SetDefault("openclaw_ai_timeout_seconds", defOpenClaw.TimeoutSeconds)
	viper.SetDefault("openclaw_ai_max_findings_per_scan", defOpenClaw.MaxFindingsPerScan)
	viper.SetDefault("openclaw_ai_max_tokens_per_scan", defOpenClaw.MaxTokensPerScan)
	viper.SetDefault("openclaw_ai_send_source_snippets", defOpenClaw.SendSourceSnippets)
	viper.SetDefault("openclaw_ai_send_full_files", defOpenClaw.SendFullFiles)
	viper.SetDefault("openclaw_ai_redact_secrets", defOpenClaw.RedactSecrets)
	viper.SetDefault("openclaw_ai_redact_pii", defOpenClaw.RedactPII)
	viper.SetDefault("openclaw_ai_allow_preinstall", defOpenClaw.AllowPreinstall)
	viper.SetDefault("openclaw_ai_allow_container_scans", defOpenClaw.AllowContainerScans)
	viper.SetDefault("openclaw_ai_allow_repo_scans", defOpenClaw.AllowRepoScans)
	viper.SetDefault("openclaw_ai_require_operator_approval", defOpenClaw.RequireOperatorApproval)
	viper.SetDefault("openclaw_ai_store_prompts", defOpenClaw.StorePrompts)
	viper.SetDefault("openclaw_ai_store_responses", defOpenClaw.StoreResponses)
	viper.SetDefault("openclaw_ai_advisory_only", defOpenClaw.AdvisoryOnly)
	viper.SetDefault("ai_recommendations_enabled", defOpenClaw.Enabled)
	viper.SetDefault("ai_recommendations_provider", defOpenClaw.Provider)
	viper.SetDefault("ai_recommendations_timeout_seconds", defOpenClaw.TimeoutSeconds)
	viper.SetDefault("ai_recommendations_max_findings_per_scan", defOpenClaw.MaxFindingsPerScan)
	viper.SetDefault("ai_recommendations_max_tokens_per_scan", defOpenClaw.MaxTokensPerScan)
	viper.SetDefault("ai_recommendations_send_source_snippets", defOpenClaw.SendSourceSnippets)
	viper.SetDefault("ai_recommendations_send_full_files", defOpenClaw.SendFullFiles)
	viper.SetDefault("ai_recommendations_redact_secrets", defOpenClaw.RedactSecrets)
	viper.SetDefault("ai_recommendations_redact_pii", defOpenClaw.RedactPII)
	viper.SetDefault("ai_recommendations_advisory_only", defOpenClaw.AdvisoryOnly)
	viper.SetDefault("ai_recommendations_require_operator_approval", defOpenClaw.RequireOperatorApproval)
	viper.SetDefault("ai_recommendations_use_cah_harness", defOpenClaw.UseCAHHarness)
	viper.SetDefault("ai_recommendations_auto_after_scan", defOpenClaw.AutoAfterScan)
	defCAH := openclaw.DefaultCAHConfig()
	viper.SetDefault("ai_recommendations_cah_enabled", defCAH.Enabled)
	viper.SetDefault("ai_recommendations_cah_max_candidates", defCAH.MaxCandidates)
	viper.SetDefault("ai_recommendations_cah_min_uncertainty_score", defCAH.MinUncertaintyScore)
	viper.SetDefault("ai_recommendations_token_budget_per_scan", defCAH.TokenBudgetPerScan)
	viper.SetDefault("ai_recommendations_fail_closed_on_redaction_error", defCAH.FailClosedOnRedaction)
	viper.SetDefault("ai_recommendations_require_strict_json", defCAH.RequireStrictJSON)
	viper.SetDefault("auth_mode", "api_key_only")
	viper.SetDefault("session_cookie_name", "rd_session")
	viper.SetDefault("session_secret", "")
	viper.SetDefault("session_ttl_hours", 12)
	viper.SetDefault("csrf_enabled", true)
	viper.SetDefault("local_admin_bootstrap_enabled", true)
	viper.SetDefault("reject_query_string_api_key", false)
	viper.SetDefault("warn_query_string_api_key", true)
	viper.SetDefault("privacy_mode", "hybrid")

	reportingDefaults := profile.DefaultReportingConfig()
	viper.SetDefault("reporting.mode", reportingDefaults.Mode)
	viper.SetDefault("reporting.default_issue_min_severity", reportingDefaults.DefaultIssueMinSeverity)
	viper.SetDefault("reporting.default_issue_min_confidence", reportingDefaults.DefaultIssueMinConfidence)
	viper.SetDefault("reporting.create_issues_for_low", reportingDefaults.CreateIssuesForLow)
	viper.SetDefault("reporting.create_issues_for_tests", reportingDefaults.CreateIssuesForTests)
	viper.SetDefault("reporting.create_issues_for_docs", reportingDefaults.CreateIssuesForDocs)
	viper.SetDefault("reporting.create_issues_for_examples", reportingDefaults.CreateIssuesForExamples)
	viper.SetDefault("reporting.create_issues_for_generated", reportingDefaults.CreateIssuesForGenerated)
	viper.SetDefault("reporting.create_issues_for_vendor", reportingDefaults.CreateIssuesForVendor)
	viper.SetDefault("reporting.max_issues_per_scan", reportingDefaults.MaxIssuesPerScan)
	viper.SetDefault("reporting.group_similar_findings", reportingDefaults.GroupSimilarFindings)
	viper.SetDefault("reporting.allow_all_categories", reportingDefaults.AllowAllCategories)
	viper.SetDefault("reporting.preserve_all_findings", reportingDefaults.PreserveAllFindings)
	viper.SetDefault("reporting.manual_review_can_create_issue", reportingDefaults.ManualReviewCanCreateIssue)
	viper.SetDefault("reporting.suppressed_findings_are_auditable", reportingDefaults.SuppressedFindingsAuditable)
	fpDefaults := profile.DefaultFalsePositiveReductionConfig()
	viper.SetDefault("false_positive_reduction.enabled", fpDefaults.Enabled)
	viper.SetDefault("false_positive_reduction.suppress_generated", fpDefaults.SuppressGenerated)
	viper.SetDefault("false_positive_reduction.suppress_vendor", fpDefaults.SuppressVendor)
	viper.SetDefault("false_positive_reduction.suppress_minified", fpDefaults.SuppressMinified)
	viper.SetDefault("false_positive_reduction.suppress_test_fixtures", fpDefaults.SuppressTestFixtures)
	viper.SetDefault("false_positive_reduction.suppress_docs_examples", fpDefaults.SuppressDocsExamples)
	viper.SetDefault("false_positive_reduction.require_file_exists", fpDefaults.RequireFileExists)
	viper.SetDefault("false_positive_reduction.require_line_match", fpDefaults.RequireLineMatch)

	// Environment variables — REPOSITORY_DETECTIVE_* only.
	viper.SetEnvPrefix("REPOSITORY_DETECTIVE")
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()

	// Read config file
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return fmt.Errorf("failed to read config file: %w", err)
		}
		logger.Warn("No config file found, using defaults and environment variables")
	}

	envcompat.Apply(viper.GetViper(), logger)

	config = &Config{}
	if err := viper.Unmarshal(config); err != nil {
		return fmt.Errorf("failed to unmarshal config: %w", err)
	}
	applyReportingDefaults(config)
	applyContainerScanDefaults(config)
	applyOpenClawDefaults(config)

	if config.DatabaseEnabled && config.DatabaseDriver == "sqlite" && config.DatabasePath != "" {
		if err := os.MkdirAll(filepath.Dir(config.DatabasePath), 0o755); err != nil && !os.IsExist(err) {
			logger.Warnf("Cannot ensure database directory %s: %v", filepath.Dir(config.DatabasePath), err)
		}
	}

	if err := viper.UnmarshalKey("skip_patterns", &config.SkipPatterns); err != nil {
		return fmt.Errorf("failed to unmarshal skip_patterns: %w", err)
	}
	if raw := viper.GetStringMap("language_mapping"); len(raw) > 0 {
		config.LanguageMapping = make(map[string]string, len(raw))
		for key, value := range raw {
			config.LanguageMapping[key] = fmt.Sprint(value)
		}
	}
	if err := viper.UnmarshalKey("repository_include_patterns", &config.RepositoryIncludePatterns); err != nil {
		return fmt.Errorf("failed to unmarshal repository_include_patterns: %w", err)
	}
	if err := viper.UnmarshalKey("repository_exclude_patterns", &config.RepositoryExcludePatterns); err != nil {
		return fmt.Errorf("failed to unmarshal repository_exclude_patterns: %w", err)
	}

	runnerCfg = mainRunnerConfig()
	if err := runnerCfg.StartupValid(); err != nil {
		return fmt.Errorf("runner configuration invalid: %w", err)
	}

	// Validate required fields — at least one forge token must be configured.
	if strings.TrimSpace(config.GiteaToken) == "" && strings.TrimSpace(config.GitHubToken) == "" {
		return fmt.Errorf("configure gitea_token and/or github_token")
	}
	if strings.TrimSpace(config.GiteaToken) != "" && config.GiteaURL == "" {
		return fmt.Errorf("gitea_url is required when gitea_token is set")
	}
	if config.needsAIProvider() {
		if config.effectiveAIProvider() == "" && config.OpenWebUIURL == "" && config.AIBaseURL == "" {
			return fmt.Errorf("configure ai_provider + ai_base_url, or legacy openwebui_url")
		}
	}
	if config.AnalysisTimeout <= 0 {
		config.AnalysisTimeout = 300
	}
	if config.MaxConcurrentAnalyses <= 0 {
		config.MaxConcurrentAnalyses = 5
	}
	if config.ListenHost == "" {
		config.ListenHost = "0.0.0.0"
	}
	if config.StartupCheckTimeout <= 0 {
		config.StartupCheckTimeout = 10
	}
	if !config.DatabaseEnabled {
		config.SchedulerEnabled = false
	}
	if config.SchedulerPollIntervalSeconds <= 0 {
		config.SchedulerPollIntervalSeconds = 60
	}
	if config.SchedulerMaxConcurrentScans <= 0 {
		config.SchedulerMaxConcurrentScans = 1
	}

	if err := config.validateAuth(); err != nil {
		return err
	}
	if err := config.validatePrivacy(); err != nil {
		return err
	}

	return nil
}

func (c *Config) validatePrivacy() error {
	mode := privacy.NormalizeMode(c.PrivacyMode)
	if !privacy.ValidMode(mode) {
		return fmt.Errorf("invalid privacy_mode %q (use local_only, hybrid, or external_ai_enabled)", c.PrivacyMode)
	}
	c.PrivacyMode = mode
	return nil
}

func (c *Config) validateAuth() error {
	mode := strings.TrimSpace(strings.ToLower(c.AuthMode))
	if mode == "" {
		c.AuthMode = "api_key_only"
	} else if mode != "api_key_only" && mode != "local" {
		return fmt.Errorf("invalid auth_mode %q (use api_key_only or local)", c.AuthMode)
	} else {
		c.AuthMode = mode
	}
	if c.AuthMode == "local" {
		if strings.TrimSpace(c.SessionSecret) == "" {
			return fmt.Errorf("auth_mode=local requires session_secret (set REPOSITORY_DETECTIVE_SESSION_SECRET)")
		}
		if !c.DatabaseEnabled {
			return fmt.Errorf("auth_mode=local requires database_enabled=true")
		}
	}
	if strings.TrimSpace(c.SessionCookieName) == "" {
		c.SessionCookieName = "rd_session"
	}
	if c.SessionTTLHours <= 0 {
		c.SessionTTLHours = 12
	}
	return nil
}

func setupRoutes(router *gin.Engine) {
	router.Use(security.MiddlewareHeaders(), security.MiddlewareMaxBody(security.DefaultMaxRequestBody))
	// Set body size limit for multipart uploads
	router.MaxMultipartMemory = 8 << 20 // 8 MB max

	// Health check — no auth required
	router.GET("/health/live", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":  "ok",
			"service": "repository-detective",
		})
	})
	router.GET("/health", func(c *gin.Context) {
		ready := componentsReady.Load()
		payload := healthPayload(ready)
		// Always 200 so Docker/orchestrator health probes do not flap while components initialize.
		c.JSON(http.StatusOK, payload)
	})

	router.GET("/", func(c *gin.Context) {
		c.Redirect(http.StatusFound, "/onboard/")
	})

	uiBase := strings.TrimSuffix(strings.TrimSpace(config.UIBasePath), "/")
	if uiBase == "" {
		uiBase = "/ui"
	}
	router.GET("/favicon.ico", func(c *gin.Context) {
		c.Redirect(http.StatusFound, uiBase+"/static/favicon.svg?v=2")
	})
	router.GET("/favicon.svg", func(c *gin.Context) {
		c.Redirect(http.StatusFound, uiBase+"/static/favicon.svg?v=2")
	})

	// Webhook endpoint for Gitea — rate limited and webhook secret auth
	router.POST("/webhook", requireComponentsReady(), func(c *gin.Context) {
		webhookHandler.HandleWebhook(c)
	})

	// API endpoints — require API key auth
	api := router.Group("/api/v1")
	api.Use(requireComponentsReady(), requireAPIKeyAuth())
	{
		api.POST("/analyze", handleManualAnalysis)
		api.POST("/analyze/all", handleBulkAnalysis)
		api.GET("/status", handleStatus)
		api.GET("/about", handleAbout)
		api.GET("/openapi.yaml", handleOpenAPI)
		api.POST("/config/reload", handleConfigReload)
	}

	onboardAPI := router.Group("/api/v1/onboard")
	onboardAPI.Use(requireAPIKeyAuth())

	onboardingHandler = handlers.NewOnboardingHandler(logger, handlers.OnboardingConfig{
		GiteaURL:      config.GiteaURL,
		PublicURL:     config.PublicURL,
		GiteaScanOrgs: parseCommaSeparatedOrgs(os.Getenv("GITEA_SCAN_ORGS")),
		AIConfig: ai.Config{
			Provider: ai.ProviderType(config.AIProvider),
			BaseURL:  firstNonEmpty(config.AIBaseURL, config.OpenWebUIURL),
			APIKey:   firstNonEmpty(config.AIAPIKey, config.OpenWebUIToken),
			Model:    firstNonEmpty(config.AIModel, config.OpenWebUIModel),
		},
	})
	onboardingHandler.RegisterRoutes(router, onboardAPI)

	logger.Info("Routes configured successfully")
}

func requireComponentsReady() gin.HandlerFunc {
	return func(c *gin.Context) {
		if componentsReady.Load() {
			c.Next()
			return
		}
		c.AbortWithStatusJSON(http.StatusServiceUnavailable, gin.H{
			"status": "starting",
			"error":  "Repository Detective is still initializing — retry in a few seconds",
		})
	}
}

// requireAPIKeyAuth middleware requires API key for protected endpoints.
func requireAPIKeyAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		if config.APIKey == "" {
			c.AbortWithStatusJSON(http.StatusServiceUnavailable, gin.H{"error": "API key not configured"})
			return
		}

		apiKey := c.GetHeader("X-Repository-Detective-API-Key")
		if apiKey == "" {
			if auth := c.GetHeader("Authorization"); strings.HasPrefix(auth, "Bearer ") {
				apiKey = strings.TrimSpace(strings.TrimPrefix(auth, "Bearer "))
			}
		}
		fromQuery := false
		if apiKey == "" {
			apiKey = c.Query("api_key")
			fromQuery = apiKey != ""
		}

		if apiKey == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "API key required"})
			return
		}

		if fromQuery {
			if config.RejectQueryStringAPIKey {
				c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
					"error": "Query string API keys are disabled; use X-Repository-Detective-API-Key or Authorization: Bearer",
				})
				return
			}
			if config.WarnQueryStringAPIKey {
				logger.Warn("Query string API key accepted (deprecated); prefer header authentication")
			}
		}

		// Constant-time comparison to prevent timing attacks
		if !hmac.Equal([]byte(apiKey), []byte(config.APIKey)) {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid API key"})
			return
		}

		c.Next()
	}
}

// initializeComponents initializes all the plugin components
func initializeComponents() error {
	logger.Info("Initializing components...")

	if rdStore != nil {
		if err := rdStore.Close(); err != nil {
			logger.Warnf("Failed to close existing database: %v", err)
		}
		rdStore = nil
		scanRecorder = nil
		controlPlaneHandler = nil
		preinstallHandler = nil
		preinstallRunner = nil
		operatorUI = nil
	}
	if scanScheduler != nil {
		scanScheduler.Stop()
		scanScheduler = nil
	}
	if schedulerCancel != nil {
		schedulerCancel()
		schedulerCancel = nil
	}

	if config.DatabaseEnabled {
		s, err := store.Open(store.Config{
			Enabled: true,
			Driver:  config.DatabaseDriver,
			Path:    config.DatabasePath,
			DSN:     config.DatabaseDSN,
		})
		if err != nil {
			return fmt.Errorf("failed to open database: %w", err)
		}
		rdStore = s
		scanRecorder = store.NewRecorder(s, logger)
		initSuppressionMatcher()
		logger.Infof("Local database enabled (driver=%s path=%s)", config.DatabaseDriver, config.DatabasePath)
	} else {
		scanRecorder = store.NewRecorder(nil, logger)
		logger.Info("Local database disabled — running without persistence")
	}

	globalSnapshot := api.GlobalSnapshotFromConfig(api.GlobalConfigInput{
		ScanProfile:                 config.ScanProfile,
		WorkspaceMode:               config.WorkspaceMode,
		AnalysisDepth:               config.AnalysisDepth,
		EnableLLMAuditors:           config.EnableLLMAuditors,
		EnableTrivy:                 config.EnableTrivy,
		EnableGrype:                 config.EnableGrype,
		EnableGitleaks:              config.EnableGitleaks,
		EnableSemgrep:               config.EnableSemgrep,
		EnableGovulncheck:           config.EnableGovulncheck,
		EnableGosec:                 config.EnableGosec,
		EnableStaticcheck:           config.EnableStaticcheck,
		EnableHadolint:              config.EnableHadolint,
		EnableCheckov:               config.EnableCheckov,
		EnableLinters:               config.EnableLinters,
		GiteaStatusFailOn:           config.GiteaStatusFailOn,
		MinIssueConfidence:          config.MinIssueConfidence,
		AutoCreateIssues:            config.AutoCreateIssues,
		EnableHealthChecks:          config.EnableHealthChecks,
		EnableTechDebtChecks:        config.EnableTechDebtChecks,
		EnableReliabilityChecks:     config.EnableReliabilityChecks,
		EnableMaintainabilityChecks: config.EnableMaintainabilityChecks,
		EnableTestGapChecks:         config.EnableTestGapChecks,
		EnablePerformanceChecks:     config.EnablePerformanceChecks,
		EnableAIRiskChecks:          config.EnableAIRiskChecks,
		HealthMaxFindings:           config.HealthMaxFindings,
		HealthLargeFileLines:        config.HealthLargeFileLines,
		HealthLargeFunctionLines:    config.HealthLargeFunctionLines,
		HealthMaxNestingDepth:       config.HealthMaxNestingDepth,
		HealthMaxFunctionParams:     config.HealthMaxFunctionParams,
		EnableCodeGraph:             config.EnableCodeGraph,
		GraphMaxNodes:               config.GraphMaxNodes,
		GraphMaxEdges:               config.GraphMaxEdges,
		GraphTimeoutSeconds:         config.GraphTimeoutSeconds,
		GraphIncludeFunctions:       config.GraphIncludeFunctions,
		GraphIncludeFindings:        config.GraphIncludeFindings,
		GovulncheckTimeoutSeconds:   config.GovulncheckTimeoutSeconds,
		GosecTimeoutSeconds:         config.GosecTimeoutSeconds,
		StaticcheckTimeoutSeconds:   config.StaticcheckTimeoutSeconds,
		GoScannerMaxFindings:        config.GoScannerMaxFindings,
		HadolintTimeoutSeconds:      config.HadolintTimeoutSeconds,
		CheckovTimeoutSeconds:       config.CheckovTimeoutSeconds,
		IACScannerMaxFindings:       config.IACScannerMaxFindings,
	})
	appGlobalSnapshot = globalSnapshot
	if err := loadAndApplyPlatformSettingsOverrides(); err != nil {
		logger.Warnf("platform settings overrides: %v", err)
	} else {
		globalSnapshot = appGlobalSnapshot
	}
	initNotifyManager()
	controlPlaneHandler = api.NewHandler(rdStore, globalSnapshot, logger)
	controlPlaneHandler.SetToolsProbe(func() []operator.ToolStatus {
		return operator.CheckTools(operatorScannerConfig())
	})
	if notifyManager != nil {
		controlPlaneHandler.SetNotificationGlobal(notifyManager.Config())
	}

	preinstallCfg := preinstall.Config{
		Enabled:                       config.PreinstallAuditEnabled,
		AllowPrivateNetworks:          config.PreinstallAllowPrivateNetworks,
		MaxRepoSizeMB:                 config.PreinstallMaxRepoSizeMB,
		MaxFiles:                      config.PreinstallMaxFiles,
		TimeoutSeconds:                config.PreinstallTimeoutSeconds,
		MaxFindings:                   config.PreinstallMaxFindings,
		AllowGitClone:                 config.PreinstallAllowGitClone,
		ReportIncludeProjectLink:      config.PreinstallReportIncludeProjectLink,
		RepositoryDetectiveProjectURL: config.RepositoryDetectiveProjectURL,
		SandboxEnabled:                config.PreinstallSandboxEnabled,
		SandboxRetainOnFailure:        config.PreinstallSandboxRetainOnFailure,
		SandboxMaxFileSizeMB:          config.PreinstallSandboxMaxFileSizeMB,
		SandboxAllowSubmodules:        config.PreinstallSandboxAllowSubmodules,
		SandboxNetworkMode:            config.PreinstallSandboxNetworkMode,
		SandboxReadonlyWorkspace:      config.PreinstallSandboxReadonlyWorkspace,
		Health:                        mainHealthConfig(),
		Graph:                         mainGraphConfig(),
	}
	if rdStore != nil && config.PreinstallAuditEnabled {
		preinstallRunner = preinstall.NewRunner(rdStore, preinstallCfg, mainScannerConfig(), logger)
		preinstallRunner.SetAuditNotifier(preinstallNotifyBridge{})
		preinstallHandler = api.NewPreinstallHandler(rdStore, preinstallRunner, logger)
		logger.Info("Pre-install audit mode enabled")
	}

	runnerRegistry = runner.NewRegistry()
	runnerBackend := rdStore != nil && runnerCfg.SharedSecret != "" &&
		((runnerCfg.DelegationEnabled && runnerCfg.Mode != runner.ModeCore) || config.ContainerScan.Normalized().Enabled)
	if runnerBackend {
		runnerDispatcher = runner.NewDispatcher(rdStore, runnerCfg, logger)
		runnerReceiver = runner.NewReceiver(rdStore, runnerCfg, logger, ingestRunnerResult)
		runnerReceiver.SetJobsExpiredHandler(func(ctx context.Context, count int64) {
			notifyRunnerJobsExpired(ctx, count)
		})
		runnerHandler = api.NewRunnerHandler(rdStore, runnerCfg, runnerReceiver, runnerDispatcher, runnerRegistry, logger)
		if runnerCfg.DelegationEnabled && runnerCfg.Mode != runner.ModeCore {
			logger.Infof("Runner delegation enabled (mode=%s)", runnerCfg.Mode)
		} else if config.ContainerScan.Normalized().Enabled {
			logger.Info("Runner backend enabled for container image scanning only")
		}
	}

	if config.UIEnabled {
		uiHandler, err := ui.NewHandler(rdStore, globalSnapshot, config.UIBasePath, logger, preinstallRunner, config.PreinstallAuditEnabled, config.APIKey)
		if err == nil {
			uiHandler.SetAuthConfig(ui.AuthConfig{
				Mode:                       config.AuthMode,
				SessionSecret:              config.SessionSecret,
				SessionCookieName:          config.SessionCookieName,
				SessionTTLHours:            config.SessionTTLHours,
				CSRFEnabled:                config.CSRFEnabled,
				LocalAdminBootstrapEnabled: config.LocalAdminBootstrapEnabled,
				PublicURL:                  config.PublicURL,
				RejectQueryStringAPIKey:    config.RejectQueryStringAPIKey,
				WarnQueryStringAPIKey:      config.WarnQueryStringAPIKey,
			})
			uiHandler.SetPlatformContext(ui.PlatformContext{
				GiteaURLConfigured:                 strings.TrimSpace(config.GiteaURL) != "",
				GiteaTokenConfigured:               strings.TrimSpace(config.GiteaToken) != "",
				APIKeyConfigured:                   strings.TrimSpace(config.APIKey) != "",
				WebhookSecretConfigured:            strings.TrimSpace(config.WebhookSecret) != "",
				RunnerSharedSecretSet:              strings.TrimSpace(config.RunnerSharedSecret) != "",
				RunnerCallbackBaseURL:              config.RunnerCallbackBaseURL,
				PublicURL:                          config.PublicURL,
				RemediationPRRequireApproval:       config.RemediationPRRequireApproval,
				RemediationPRMaxFiles:              config.RemediationPRMaxFilesChanged,
				RemediationPRMaxDiffLines:          config.RemediationPRMaxDiffLines,
				RemediationPRBranchPrefix:          config.RemediationPRBranchPrefix,
				LLMSanityGateEnabled:               config.LLMSanityGateEnabled,
				BacklogControlEnabled:              config.DogfoodBacklogControlEnabled,
				MaxIssuesPerScan:                   config.Reporting.MaxIssuesPerScan,
				ScanPolicyMode:                     store.DeploymentScanMode(globalSnapshot),
				NotificationsEnabled:               config.NotificationsEnabled,
				SchedulerEnabled:                   config.SchedulerEnabled,
				RunnerDelegationEnabled:            config.RunnerDelegationEnabled,
				RunnerRequireHMAC:                  config.RunnerRequireHMAC,
				RunnerMode:                         config.RunnerMode,
				RemediationPRRequireTests:          config.RemediationPRRequireTests,
				RemediationPRUseRunnerVerification: config.RemediationPRUseRunnerVerification,
				GiteaActionsTestBackendEnabled:     config.GiteaActionsTestBackendEnabled,
				OpenClawAIReviewEnabled:            config.OpenClawAIReview.Enabled,
				OpenClawEndpointConfigured:         config.OpenClawAIReview.EndpointConfigured(),
				ContainerScanningEnabled:           config.ContainerScan.Enabled,
				ContainerScanRequireRunner:         config.ContainerScan.RequireRunner,
				ContainerScanAllowCoreSocket:       config.ContainerScan.AllowCoreDockerSocket,
				ContainerScanCreateIssues:          config.ContainerScan.CreateIssues,
			})
		}
		if err != nil {
			return fmt.Errorf("failed to initialize operator UI: %w", err)
		}
		if notifyManager != nil {
			uiHandler.SetNotificationGlobal(notifyManager.Config())
		}
		if config.RemediationPlannerEnabled {
			uiHandler.SetRemediationBackend(true, remediationUIBridge{})
		}
		if config.RemediationPREnabled {
			uiHandler.SetRemediationPRBackend(true, remediationPRUIBridge{})
			if !config.RemediationPRUseRunnerVerification {
				logger.Warn("RD-008B: remediation_pr_enabled without runner verification — Class-B validation may execute allowlisted commands on the control plane (NOT_PROVEN sandbox). See docs/RD-008B_CLASS_B_EXECUTION.md")
			}
		}
		if config.EvidenceClosureEnabled {
			uiHandler.SetClosureBackend(true, closureUIBridge{})
		}
		if rdStore != nil {
			uiHandler.SetSuppressionBackend(true, suppressionUIBridge{})
			uiHandler.SetCalibrationBackend(true, calibrationUIBridge{})
		}
		uiHandler.SetReadinessFn(func() operator.Readiness { return buildReadiness("running") })
		uiHandler.SetPlatformSettingsApplier(func(settings store.PlatformSettings) error {
			applyPlatformSettingsToRuntime(settings)
			return nil
		})
		operatorUI = uiHandler
		logger.Infof("Operator UI enabled at %s", uiHandler.BasePath())
	} else {
		logger.Info("Operator UI disabled")
	}

	checkTimeout := time.Duration(config.StartupCheckTimeout) * time.Second
	ctx, cancel := context.WithTimeout(context.Background(), checkTimeout)
	defer cancel()

	// Initialize Gitea client when configured.
	if strings.TrimSpace(config.GiteaToken) != "" {
		giteaClient = gitea.NewClient(config.GiteaURL, config.GiteaToken, logger)
		logger.Infof("Testing Gitea connection (timeout %s)...", checkTimeout)
		if err := giteaClient.TestConnection(ctx); err != nil {
			if config.SkipStartupChecks {
				logger.Warnf("Gitea connection check failed (skipped): %v", err)
			} else {
				return fmt.Errorf("failed to connect to Gitea: %w", err)
			}
		} else {
			logger.Info("Gitea connection established")
		}
	} else {
		giteaClient = nil
		logger.Info("Gitea client not configured")
	}

	if strings.TrimSpace(config.GitHubToken) != "" {
		githubClient = github.NewClient(config.GitHubURL, config.GitHubToken, logger)
		logger.Infof("Testing GitHub connection (timeout %s)...", checkTimeout)
		if err := githubClient.TestConnection(ctx); err != nil {
			if config.SkipStartupChecks {
				logger.Warnf("GitHub connection check failed (skipped): %v", err)
			} else {
				return fmt.Errorf("failed to connect to GitHub: %w", err)
			}
		} else {
			logger.Info("GitHub connection established")
		}
	}

	statusReporter = gitea.NewStatusReporter(
		giteaClient,
		config.EnableGiteaStatus,
		gitea.ChecksConfig{
			Context:                config.GiteaStatusContext,
			TargetURL:              config.PublicURL,
			FailOn:                 config.GiteaStatusFailOn,
			WarnOn:                 config.GiteaStatusWarnOn,
			IncludeScannerFailures: config.GiteaStatusIncludeScannerFailures,
		},
		logger,
	)

	// Initialize AI client (multi-provider) when LLM auditors are required
	initAIStatus()
	privacyDecision := privacy.EvaluateAIEgress(
		config.PrivacyMode,
		config.effectiveAIProvider(),
		firstNonEmpty(config.AIBaseURL, config.OpenWebUIURL),
		config.EnableLLMAuditors,
	)
	logger.Infof("Privacy mode=%s AI egress: allowed=%v class=%s (%s)",
		privacy.NormalizeMode(config.PrivacyMode), privacyDecision.Allowed, privacyDecision.EndpointClass, privacyDecision.Reason)
	if config.needsAIProvider() {
		if !privacyDecision.Allowed {
			return fmt.Errorf("privacy_mode=%s blocks AI configuration: %s", privacy.NormalizeMode(config.PrivacyMode), privacyDecision.Reason)
		}
		var err error
		aiClient, err = ai.NewClient(ai.Config{
			Provider:              ai.ProviderType(config.AIProvider),
			BaseURL:               firstNonEmpty(config.AIBaseURL, config.OpenWebUIURL),
			APIKey:                firstNonEmpty(config.AIAPIKey, config.OpenWebUIToken),
			Model:                 firstNonEmpty(config.AIModel, config.OpenWebUIModel),
			InsecureSkipTLSVerify: config.AIInsecureSkipTLSVerify,
		}, ai.LegacyConfig{
			OpenWebUIURL:   config.OpenWebUIURL,
			OpenWebUIToken: config.OpenWebUIToken,
			OpenWebUIModel: config.OpenWebUIModel,
		}, logger)
		if err != nil {
			return fmt.Errorf("failed to configure AI client: %w", err)
		}

		mode := ai.ConnectionTestMode(config.AIConnectionTestMode)
		if mode == "" {
			mode = ai.TestModeMetadataOnly
		}
		if config.AIStartupTestEnabled && !config.SkipStartupChecks {
			logger.Infof("Testing AI provider (%s, timeout %s)...", mode, checkTimeout)
			testCtx, cancel := context.WithTimeout(ctx, checkTimeout)
			st, err := ai.RunConnectionTest(testCtx, aiClient, mode, false)
			cancel()
			if err != nil {
				if config.SkipStartupChecks {
					logger.Debugf("AI provider connection check failed (skipped): %v", err)
				} else {
					return fmt.Errorf("failed to connect to AI provider: %w", err)
				}
			} else if st.LastTestOK {
				logger.Infof("AI provider reachable (%s, model=%s, test=%s)", aiClient.Provider(), aiClient.Model(), st.LastTestSource)
			}
		} else {
			logger.Info("AI startup test disabled — provider configured but not tested until manual test or AI-enabled scan")
		}
	} else {
		logger.Info("AI provider not required — deterministic-only mode (no LLM auditors)")
		aiClient = nil
	}
	if err := initOpenClawReview(); err != nil {
		return err
	}
	initRemediationPlanner()
	initClosureEngine()

	// Initialize analysis engine
	analysisConfig := &analyzers.Config{
		MaxFileSize:       config.MaxFileSize,
		AnalysisDepth:     config.AnalysisDepth,
		EnableSecurity:    config.EnableSecurity,
		EnableQuality:     config.EnableQuality,
		EnableLLMAuditors: config.EnableLLMAuditors,
		SkipPatterns:      config.SkipPatterns,
		LanguageMapping:   config.LanguageMapping,
		Scanners:          mainScannerConfig(),
		Health:            mainHealthConfig(),
		Graph:             mainGraphConfig(),
		Reporting:         config.Reporting,
		FalsePositive:     config.FalsePositiveReduction,
		Workspace: scanners.WorkspaceConfig{
			Mode:                   config.WorkspaceMode,
			MaxSizeMB:              config.WorkspaceMaxSizeMB,
			MaxFiles:               config.WorkspaceMaxFiles,
			ArchiveTimeoutSeconds:  config.WorkspaceArchiveTimeoutSeconds,
			DefaultAnalysisTimeout: config.AnalysisTimeout,
		},
	}
	analysisEngine = analyzers.NewEngine(giteaClient, githubClient, aiClient, analysisConfig, logger)

	// Initialize issue manager
	issueConfig := &issues.Config{
		AutoCreateIssues:   config.AutoCreateIssues,
		Reporting:          config.Reporting,
		BacklogControl:     buildBacklogControlConfig(*config),
		GiteaBaseURL:       config.GiteaURL,
		IssueLabels:        issues.DefaultIssueBaseLabels(),
		MaxIssuesPerRun:    config.MaxIssuesPerRun,
		SkipLowSeverity:    config.SkipLowSeverity,
		GroupSimilarIssues: config.GroupSimilarIssues,
		MinIssueConfidence: config.MinIssueConfidence,
		IssueTitleTemplate: "[{{severity}}] {{title}}",
		IssueBodyTemplate:  "",
	}
	var githubIssueClient *github.Client
	if githubClient != nil {
		if gc, ok := githubClient.(*github.Client); ok {
			githubIssueClient = gc
		}
	}
	issueConfig.GitHubBaseURL = strings.TrimSpace(config.GitHubURL)
	if issueConfig.GitHubBaseURL == "" || strings.Contains(issueConfig.GitHubBaseURL, "api.github.com") {
		issueConfig.GitHubBaseURL = "https://github.com"
	}
	issueManager = issues.NewManager(giteaClient, githubIssueClient, issueConfig, logger)
	initIssueLinkBridge()
	initReconcileEngine()
	if operatorUI != nil && reconcileEngine != nil {
		operatorUI.SetIssueReconciler(config.IssueReconciliationEnabled, uiReconcileBridge{})
	}
	startCalibrationBackgroundJob()

	webhookHandler = handlers.NewWebhookHandler(logger, &handlers.Config{
		WebhookSecret:         config.WebhookSecret,
		AllowInsecureWebhooks: config.AllowInsecureWebhooks,
		IncludePatterns:       config.RepositoryIncludePatterns,
		ExcludePatterns:       config.RepositoryExcludePatterns,
	}, &webhookProcessor{})
	if rdStore != nil {
		webhookHandler.SetDeliveryRecorder(func(eventKind, repository, commitSHA, deliveryID string, prNumber int) {
			_ = rdStore.RecordWebhookDelivery(context.Background(), store.WebhookDeliveryEvidence{
				EventKind:  eventKind,
				Repository: repository,
				CommitSHA:  commitSHA,
				DeliveryID: deliveryID,
				PRNumber:   prNumber,
			})
		})
	}

	analysisLimiter = limiter.New(config.MaxConcurrentAnalyses)
	logger.Infof("Analysis concurrency limit: %d", config.MaxConcurrentAnalyses)

	tmpDir := scanners.EnsureScannerTempDir(filepath.Dir(config.DatabasePath), logger)
	if tmpDir == "" {
		tmpDir = scanners.EnsureScannerTempDir("/app/data", logger)
	}
	scanners.CleanupStaleScannerScratch(tmpDir, 0, logger)
	scanners.CleanupStaleScannerScratch("/tmp", 0, logger)
	go scanners.WarmGrypeDB(context.Background(), logger)

	if config.SchedulerEnabled && rdStore != nil {
		schedulerCtx, schedulerCancel = context.WithCancel(context.Background())
		scanScheduler = orch.NewScheduler(
			rdStore,
			runScheduledRepositoryScan,
			analysisLimiter,
			orch.Config{
				Enabled:       true,
				PollInterval:  time.Duration(config.SchedulerPollIntervalSeconds) * time.Second,
				MaxConcurrent: config.SchedulerMaxConcurrentScans,
			},
			logger,
		)
		scanScheduler.Start(schedulerCtx)
	} else {
		logger.Info("Scheduled scans disabled (scheduler_enabled=false or database disabled)")
	}

	if rdStore != nil {
		// In-memory scan workers do not survive process restart. Reap any "started"
		// rows older than a short grace window so concurrent slots are not stuck.
		staleAge := 2 * time.Minute
		if n, err := rdStore.ReapStaleScans(context.Background(), staleAge); err != nil {
			logger.Warnf("Failed to reap stale scans: %v", err)
		} else if n > 0 {
			logger.Infof("Reaped %d stale started scan(s)", n)
		}
		if n, err := rdStore.ExpireStaleRunnerJobs(context.Background(), time.Now().UTC()); err != nil {
			logger.Warnf("Failed to expire stale runner jobs: %v", err)
		} else if n > 0 {
			logger.Infof("Expired %d stale runner job(s)", n)
		}
	}

	logger.Info("All components initialized successfully")
	wireScanTrigger()
	return nil
}

type webhookProcessor struct{}

func (p *webhookProcessor) ProcessPush(ctx context.Context, payload *handlers.GiteaWebhookPayload) {
	owner := payload.Repository.Owner.LoginName()
	repo := payload.Repository.Name
	ref := payload.After
	changedFiles := handlers.CollectChangedFiles(payload.Commits)
	commitSHA := strings.TrimSpace(ref)

	runAnalysis(ctx, func(analysisCtx context.Context) {
		scanCtx := store.ScanContext{
			Owner:         owner,
			Repo:          repo,
			ForgeType:     store.ForgeTypeGitea,
			TriggerType:   store.TriggerPush,
			Ref:           payload.Ref,
			CommitSHA:     commitSHA,
			CloneURL:      payload.Repository.CloneURL,
			ConnectedRepo: true,
		}
		analysisCtx, repositoryID := beginPersistedScan(analysisCtx, &scanCtx)
		analysisCtx = analyzers.WithForgeType(analysisCtx, store.ForgeTypeGitea)
		analysisCtx, effective := resolveEffectiveSettingsForRepo(analysisCtx, store.ForgeTypeGitea, owner, repo)
		if !effective.Enabled {
			logger.Infof("Push scan skipped — repository %s/%s disabled in settings", owner, repo)
			return
		}

		statusReporter.ReportPending(analysisCtx, owner, repo, commitSHA)

		result, err := analysisEngine.AnalyzeChangedFiles(analysisCtx, owner, repo, ref, changedFiles)
		postCtx, postCancel := postAnalysisContext(analysisCtx)
		defer postCancel()
		finishPersistedScan(postCtx, &scanCtx, repositoryID, result, err)
		if err != nil {
			logger.Errorf("Push analysis failed: %v", err)
			statusReporter.ReportFinalWithPolicy(postCtx, owner, repo, resolveCommitSHA(result, commitSHA), nil, nil, true, effective.PolicyLevel, effective.SeverityGate)
			return
		}
		eval := statusReporter.ReportFinalWithPolicy(
			postCtx,
			owner,
			repo,
			resolveCommitSHA(result, commitSHA),
			severitiesForStatus(result, effective, repositoryID),
			scannerSummariesForPolicy(result, effective),
			false,
			effective.PolicyLevel,
			effective.SeverityGate,
		)
		notifyPRGateFailed(postCtx, repositoryID, owner, repo, scanCtx.ScanID, eval)
		createIssuesFromResult(postCtx, store.ForgeTypeGitea, owner, repo, result, fmt.Sprintf("Push to %s", payload.Ref), ref, 0, repositoryID, effective)
	})
}

func (p *webhookProcessor) ProcessPullRequest(ctx context.Context, payload *handlers.GiteaWebhookPayload) {
	owner := payload.Repository.Owner.LoginName()
	repo := payload.Repository.Name
	prNumber := payload.PullRequest.Number
	commitSHA := strings.TrimSpace(payload.PullRequest.Head.SHA)

	runAnalysis(ctx, func(analysisCtx context.Context) {
		scanCtx := store.ScanContext{
			Owner:         owner,
			Repo:          repo,
			ForgeType:     store.ForgeTypeGitea,
			TriggerType:   store.TriggerPR,
			Ref:           payload.PullRequest.Head.Ref,
			CommitSHA:     commitSHA,
			PRNumber:      prNumber,
			CloneURL:      payload.Repository.CloneURL,
			ConnectedRepo: true,
		}
		analysisCtx, repositoryID := beginPersistedScan(analysisCtx, &scanCtx)
		analysisCtx = analyzers.WithForgeType(analysisCtx, store.ForgeTypeGitea)
		analysisCtx, effective := resolveEffectiveSettingsForRepo(analysisCtx, store.ForgeTypeGitea, owner, repo)
		if !effective.Enabled {
			logger.Infof("PR scan skipped — repository %s/%s disabled in settings", owner, repo)
			return
		}

		statusReporter.ReportPending(analysisCtx, owner, repo, commitSHA)

		result, err := analysisEngine.AnalyzePullRequest(analysisCtx, owner, repo, prNumber)
		postCtx, postCancel := postAnalysisContext(analysisCtx)
		defer postCancel()
		finishPersistedScan(postCtx, &scanCtx, repositoryID, result, err)
		if err != nil {
			logger.Errorf("Pull request analysis failed: %v", err)
			statusReporter.ReportFinalWithPolicy(postCtx, owner, repo, resolveCommitSHA(result, commitSHA), nil, nil, true, effective.PolicyLevel, effective.SeverityGate)
			return
		}
		eval := statusReporter.ReportFinalWithPolicy(
			postCtx,
			owner,
			repo,
			resolveCommitSHA(result, commitSHA),
			severitiesForStatus(result, effective, repositoryID),
			scannerSummariesForPolicy(result, effective),
			false,
			effective.PolicyLevel,
			effective.SeverityGate,
		)
		notifyPRGateFailed(postCtx, repositoryID, owner, repo, scanCtx.ScanID, eval)
		createIssuesFromResult(postCtx, store.ForgeTypeGitea, owner, repo, result, fmt.Sprintf("Pull Request #%d", prNumber), "", prNumber, repositoryID, effective)
		maybePostPRPolicySummary(postCtx, owner, repo, prNumber, result, effective, eval, repositoryID)
	})
}

func resolveCommitSHA(result *analyzers.AnalysisResult, fallback string) string {
	if result != nil && result.CommitSHA != "" {
		return result.CommitSHA
	}
	fallback = strings.TrimSpace(fallback)
	if gitea.IsCommitSHA(fallback) {
		return fallback
	}
	return ""
}

func scannerSummaries(result *analyzers.AnalysisResult) []gitea.ScannerResultSummary {
	return scannerSummariesForPolicy(result, store.EffectiveFromGlobalSnapshot(store.DefaultGlobalSettings()))
}

func scannerSummariesForPolicy(result *analyzers.AnalysisResult, effective store.EffectiveSettings) []gitea.ScannerResultSummary {
	if result == nil {
		return nil
	}
	requiredSet := map[string]struct{}{}
	for _, name := range store.RequiredScannersForProfile(effective.ScanProfile, effective) {
		requiredSet[strings.ToLower(name)] = struct{}{}
	}
	seen := map[string]bool{}
	summaries := make([]gitea.ScannerResultSummary, 0, len(result.ScannerResults)+len(requiredSet))
	for _, scannerResult := range result.ScannerResults {
		name := strings.ToLower(scannerResult.Scanner)
		seen[name] = true
		_, req := requiredSet[name]
		summaries = append(summaries, gitea.ScannerResultSummary{
			Scanner:  scannerResult.Scanner,
			Status:   string(scannerResult.Status),
			Required: req,
		})
	}
	// RD-012A: required scanners with no result still participate in policy evaluation.
	for name := range requiredSet {
		if seen[name] {
			continue
		}
		status := "scanner_unavailable"
		if !store.ScannerEnabledInSettings(name, effective) {
			status = "disabled"
		}
		summaries = append(summaries, gitea.ScannerResultSummary{
			Scanner:  name,
			Status:   status,
			Required: true,
		})
	}
	// RD-012A: never allow silent 0/0 POLICY_MET for an empty required set.
	if len(requiredSet) == 0 {
		summaries = append(summaries, gitea.ScannerResultSummary{
			Scanner:  "required-analysis",
			Status:   "scanner_unavailable",
			Required: true,
		})
	}
	return summaries
}

func runAnalysis(parent context.Context, fn func(context.Context)) {
	if parent == nil {
		parent = context.Background()
	}
	// Preserve request-scoped flags (report-only dry run, scan profile override) while detaching cancel.
	parent = context.WithoutCancel(parent)
	// Wait for a concurrency slot without a scan timeout — queue wait must not consume analysis time.
	if err := analysisLimiter.Run(context.Background(), func() {
		analysisCtx, cancel := context.WithTimeout(parent, time.Duration(config.AnalysisTimeout)*time.Second)
		defer cancel()
		fn(analysisCtx)
	}); err != nil {
		logger.Warnf("Analysis skipped — concurrency limit reached or timed out waiting for slot: %v", err)
	}
}

func runScheduledRepositoryScan(ctx context.Context, repo store.ScheduledRepository) error {
	var runErr error
	analysisCtx, cancel := context.WithTimeout(context.Background(), time.Duration(config.AnalysisTimeout)*time.Second)
	defer cancel()

	if err := analysisLimiter.Run(analysisCtx, func() {
		ref := strings.TrimSpace(repo.DefaultBranch)
		if ref == "" {
			ref = "main"
		}
		forgeType := normalizeForgeType(repo.ForgeType)
		if client := repoClientForForge(forgeType); client != nil {
			if resolved, rerr := client.ResolveRef(analysisCtx, repo.Owner, repo.Name, ref); rerr != nil {
				logger.WithError(rerr).Warnf("Scheduled scan ref resolve failed for %s (continuing with %q)", repo.FullName, ref)
			} else if strings.TrimSpace(resolved) != "" {
				ref = strings.TrimSpace(resolved)
			}
		}

		scanCtx := store.ScanContext{
			Owner:         repo.Owner,
			Repo:          repo.Name,
			ForgeType:     repo.ForgeType,
			CloneURL:      repo.CloneURL,
			DefaultBranch: firstNonEmpty(ref, repo.DefaultBranch),
			TriggerType:   store.TriggerScheduled,
			Ref:           ref,
			ConnectedRepo: true,
		}
		scanCtxInner, repositoryID := beginPersistedScan(analysisCtx, &scanCtx)
		scanCtxInner = analyzers.WithForgeType(scanCtxInner, forgeType)
		scanCtxInner, effective := resolveEffectiveSettingsForRepo(scanCtxInner, forgeType, repo.Owner, repo.Name)
		if !effective.Enabled {
			logger.Infof("Scheduled scan skipped — repository %s disabled in settings", repo.FullName)
			return
		}

		if rdStore != nil && repositoryID > 0 {
			if _, uerr := rdStore.UpsertRepository(scanCtxInner, store.Repository{
				ForgeType:     forgeType,
				Owner:         repo.Owner,
				Name:          repo.Name,
				FullName:      repo.FullName,
				CloneURL:      repo.CloneURL,
				DefaultBranch: ref,
				ConnectedRepo: true,
			}); uerr != nil {
				logger.WithError(uerr).Debugf("Scheduled scan: could not refresh default_branch for %s", repo.FullName)
			}
			if dbRepo, rerr := rdStore.GetRepository(scanCtxInner, repositoryID); rerr == nil {
				if delegated, derr := tryDelegateScan(scanCtxInner, &scanCtx, dbRepo, effective); delegated {
					logger.WithFields(logrus.Fields{
						"scan_id": scanCtx.ScanID, "repo": repo.FullName,
					}).Info("Scheduled scan delegated to runner")
					return
				} else if derr != nil {
					finishPersistedScan(scanCtxInner, &scanCtx, repositoryID, nil, derr)
					runErr = derr
					return
				}
			}
		}

		result, err := analysisEngine.AnalyzeRepository(analyzers.WithForgeType(scanCtxInner, repo.ForgeType), repo.Owner, repo.Name, ref)
		postCtx, postCancel := postAnalysisContext(scanCtxInner)
		defer postCancel()
		finishPersistedScan(postCtx, &scanCtx, repositoryID, result, err)
		if err != nil {
			logger.WithFields(logrus.Fields{
				"scan_id":       scanCtx.ScanID,
				"repo":          repo.FullName,
				"trigger_type":  store.TriggerScheduled,
				"schedule_cron": repo.ScheduleCron,
			}).Errorf("Scheduled analysis failed: %v", err)
			runErr = err
			return
		}

		createIssuesFromResult(postCtx, forgeType, repo.Owner, repo.Name, result,
			fmt.Sprintf("Scheduled scan (%s)", repo.ScheduleCron), ref, 0, repositoryID, effective)
	}); err != nil {
		return fmt.Errorf("scheduled scan skipped: %w", err)
	}
	return runErr
}

func scanPolicyFromEffective(e store.EffectiveSettings) analyzers.ScanPolicy {
	return analyzers.ScanPolicy{
		Enabled:                     e.Enabled,
		PolicyLevel:                 e.PolicyLevel,
		WorkspaceMode:               e.WorkspaceMode,
		AnalysisDepth:               e.AnalysisDepth,
		EnableLLMAuditors:           e.EnableLLMAuditors,
		EnableTrivy:                 e.EnableTrivy,
		EnableGrype:                 e.EnableGrype,
		EnableGitleaks:              e.EnableGitleaks,
		EnableSemgrep:               e.EnableSemgrep,
		EnableGovulncheck:           e.EnableGovulncheck,
		EnableGosec:                 e.EnableGosec,
		EnableStaticcheck:           e.EnableStaticcheck,
		EnableHadolint:              e.EnableHadolint,
		EnableCheckov:               e.EnableCheckov,
		EnableLinters:               e.EnableLinters,
		SeverityGate:                e.SeverityGate,
		ConfidenceGate:              e.ConfidenceGate,
		IssuePolicy:                 e.IssuePolicy,
		RemediationPolicy:           e.RemediationPolicy,
		AIPolicy:                    e.AIPolicy,
		EnableHealthChecks:          e.EnableHealthChecks,
		EnableTechDebtChecks:        e.EnableTechDebtChecks,
		EnableReliabilityChecks:     e.EnableReliabilityChecks,
		EnableMaintainabilityChecks: e.EnableMaintainabilityChecks,
		EnableTestGapChecks:         e.EnableTestGapChecks,
		EnablePerformanceChecks:     e.EnablePerformanceChecks,
		EnableAIRiskChecks:          e.EnableAIRiskChecks,
		HealthMaxFindings:           e.HealthMaxFindings,
		HealthLargeFileLines:        e.HealthLargeFileLines,
		HealthLargeFunctionLines:    e.HealthLargeFunctionLines,
		HealthMaxNestingDepth:       e.HealthMaxNestingDepth,
		HealthMaxFunctionParams:     e.HealthMaxFunctionParams,
		EnableCodeGraph:             e.EnableCodeGraph,
		GraphMaxNodes:               e.GraphMaxNodes,
		GraphMaxEdges:               e.GraphMaxEdges,
		GraphTimeoutSeconds:         e.GraphTimeoutSeconds,
		GraphIncludeFunctions:       e.GraphIncludeFunctions,
		GraphIncludeFindings:        e.GraphIncludeFindings,
		GovulncheckTimeoutSeconds:   e.GovulncheckTimeoutSeconds,
		GosecTimeoutSeconds:         e.GosecTimeoutSeconds,
		StaticcheckTimeoutSeconds:   e.StaticcheckTimeoutSeconds,
		GoScannerMaxFindings:        e.GoScannerMaxFindings,
		HadolintTimeoutSeconds:      e.HadolintTimeoutSeconds,
		CheckovTimeoutSeconds:       e.CheckovTimeoutSeconds,
		IACScannerMaxFindings:       e.IACScannerMaxFindings,
	}
}

func withReportOnlyDryRun(ctx context.Context, enabled bool) context.Context {
	if !enabled {
		return ctx
	}
	return context.WithValue(ctx, reportOnlyDryRunKey{}, true)
}

func reportOnlyDryRunFromContext(ctx context.Context) bool {
	v, _ := ctx.Value(reportOnlyDryRunKey{}).(bool)
	return v
}

func withScanProfileOverride(ctx context.Context, profile string) context.Context {
	profile = store.NormalizeScanProfile(profile)
	if profile == "" || !store.IsValidScanProfile(profile) {
		return ctx
	}
	return context.WithValue(ctx, scanProfileOverrideKey{}, profile)
}

func scanProfileOverrideFromContext(ctx context.Context) string {
	v, _ := ctx.Value(scanProfileOverrideKey{}).(string)
	return v
}

func resolveEffectiveSettingsForRepo(ctx context.Context, forgeType, owner, repo string) (context.Context, store.EffectiveSettings) {
	forgeType = normalizeForgeType(forgeType)
	repoSettings := store.RepoSettings{}
	effective, meta := store.ResolveEffectiveSettingsFull(appGlobalSnapshot, repoSettings)
	if rdStore != nil {
		fullName := owner + "/" + repo
		dbRepo, err := rdStore.GetRepositoryByFullName(ctx, forgeType, fullName)
		if err == nil {
			settings, serr := rdStore.GetRepoSettings(ctx, dbRepo.ID)
			if serr == nil {
				repoSettings = settings
				effective, meta = store.ResolveEffectiveSettingsFull(appGlobalSnapshot, settings)
			}
		}
	}
	if override := scanProfileOverrideFromContext(ctx); override != "" && override != store.ScanProfileCustom {
		globalCfg := store.EffectiveFromGlobalSnapshot(appGlobalSnapshot)
		effective = store.MergeConfigOverProfile(store.ProfileDefaults(override), globalCfg)
		effective = store.ApplyRepoOverridesToEffective(effective, repoSettings)
		meta.ScanProfile = override
		meta.ProfileSource = "request_override"
	}
	if effective.AIPolicy == store.AIPolicyAllowed && aiClient == nil {
		effective.EnableLLMAuditors = false
	}
	if reportOnlyDryRunFromContext(ctx) {
		store.ApplyReportOnlyDryRunSettings(&effective)
	}
	policy := scanPolicyFromEffective(effective)
	policy.ScanProfile = meta.ScanProfile
	policy.ProfileModified = meta.ProfileModified
	policy.ProfileSource = meta.ProfileSource
	return analyzers.WithScanPolicy(ctx, policy), effective
}

func filterIssuesForForge(issues []ai.CodeIssue, effective store.EffectiveSettings, reporting profile.ReportingConfig) []ai.CodeIssue {
	if !store.ShouldCreateForgeIssues(effective) {
		return nil
	}
	out := make([]ai.CodeIssue, 0, len(issues))
	for _, issue := range issues {
		if issue.ReportingAction != "" {
			if profile.IsForgeAction(issue.ReportingAction, reporting) {
				out = append(out, issue)
			}
			continue
		}
		if store.PassesIssueGates(issue.Severity, issue.Confidence, effective) {
			out = append(out, issue)
		}
	}
	return out
}

func applyReportingDefaults(cfg *Config) {
	if cfg == nil {
		return
	}
	defaults := profile.DefaultReportingConfig()
	if cfg.Reporting.Mode == "" {
		cfg.Reporting.Mode = defaults.Mode
	}
	if cfg.Reporting.DefaultIssueMinSeverity == "" {
		cfg.Reporting.DefaultIssueMinSeverity = defaults.DefaultIssueMinSeverity
	}
	if cfg.Reporting.DefaultIssueMinConfidence == "" {
		cfg.Reporting.DefaultIssueMinConfidence = defaults.DefaultIssueMinConfidence
	}
	if cfg.Reporting.MaxIssuesPerScan <= 0 {
		cfg.Reporting.MaxIssuesPerScan = defaults.MaxIssuesPerScan
	}
	if cfg.Reporting.DefaultActionBySeverity == nil {
		cfg.Reporting.DefaultActionBySeverity = defaults.DefaultActionBySeverity
	}
	if cfg.Reporting.CategoryOverrides == nil {
		cfg.Reporting.CategoryOverrides = defaults.CategoryOverrides
	}
	if cfg.Reporting.SourceTypeOverrides == nil {
		cfg.Reporting.SourceTypeOverrides = defaults.SourceTypeOverrides
	}
	if cfg.Reporting.RuleOverrides == nil {
		cfg.Reporting.RuleOverrides = defaults.RuleOverrides
	}
	if cfg.MaxIssuesPerRun <= 0 && cfg.Reporting.MaxIssuesPerScan > 0 {
		cfg.MaxIssuesPerRun = cfg.Reporting.MaxIssuesPerScan
	}
}

func severitiesForStatus(result *analyzers.AnalysisResult, effective store.EffectiveSettings, repositoryID int64) []string {
	if result == nil {
		return nil
	}
	issues := filterIssuesWithSuppression(repositoryID, result.Issues)
	severities := make([]string, len(issues))
	confidences := make([]float64, len(issues))
	for i, issue := range issues {
		severities[i] = issue.Severity
		confidences[i] = issue.Confidence
	}
	return store.SeveritiesForStatus(severities, confidences, effective)
}

func beginPersistedScan(ctx context.Context, scanCtx *store.ScanContext) (context.Context, int64) {
	if scanRecorder == nil || !scanRecorder.Enabled() || scanCtx == nil {
		return ctx, 0
	}
	if scanCtx.ScanID == "" {
		scanCtx.ScanID = scanid.New()
	}
	ctx = scanid.With(ctx, scanCtx.ScanID)

	repo, err := scanRecorder.BeginScan(ctx, *scanCtx)
	if err != nil {
		logger.Warnf("Failed to begin scan persistence: %v", err)
		return ctx, 0
	}
	return ctx, repo.ID
}

func finishPersistedScan(ctx context.Context, scanCtx *store.ScanContext, repositoryID int64, result *analyzers.AnalysisResult, analysisErr error) {
	if scanRecorder == nil || !scanRecorder.Enabled() || scanCtx == nil || scanCtx.ScanID == "" {
		return
	}
	scanID := scanCtx.ScanID
	var data *store.ScanCompletion
	if result != nil {
		scanners := make([]store.ScanCompletionScanner, 0, len(result.ScannerResults))
		for _, sr := range result.ScannerResults {
			scanners = append(scanners, store.ScanCompletionScanner{
				Scanner:             sr.Scanner,
				Status:              string(sr.Status),
				FindingsCount:       len(sr.Findings),
				Detail:              sr.Detail,
				ApplicabilityReason: sr.ApplicabilityReason,
			})
		}
		data = &store.ScanCompletion{
			IssuesFound:           len(result.Issues),
			FilesAnalyzed:         result.FilesAnalyzed,
			AnalysisTime:          result.AnalysisTime,
			OverallScore:          result.OverallScore,
			ScoreComplete:         result.ScoreComplete,
			ScoreIncompleteReason: result.ScoreIncompleteReason,
			ScoreExplanation:      result.ScoreExplanation,
			CommitSHA:             result.CommitSHA,
			WorkspaceModeUsed:     result.WorkspaceModeUsed,
			PolicySnapshot:        result.PolicySnapshot,
			ScannerResults:        scanners,
			RepoProfile:           result.RepoProfile,
		}
		if result.PolicySnapshot != nil {
			if raw, err := json.Marshal(result.PolicySnapshot); err == nil {
				var snap struct {
					EnableCodeGraph bool `json:"enable_code_graph"`
					AnalysisDepth   int  `json:"analysis_depth"`
				}
				if json.Unmarshal(raw, &snap) == nil {
					data.GraphEnabled = snap.EnableCodeGraph
				}
			}
		}
		if result.Graph != nil {
			if raw, err := json.Marshal(result.Graph); err == nil {
				data.GraphJSON = raw
				data.GraphNodeCount = result.Graph.Metrics.NodeCount
				data.GraphEdgeCount = result.Graph.Metrics.EdgeCount
				data.GraphTruncated = result.Graph.Metrics.Truncated
				if data.GraphTruncated {
					data.GraphState = store.GraphStateTruncated
				} else {
					data.GraphState = store.GraphStateAvailable
				}
			} else {
				data.GraphError = err.Error()
				data.GraphState = store.GraphStateFailed
			}
		} else if data.GraphEnabled {
			data.GraphState = store.GraphStateMissing
		}
	}
	if err := scanRecorder.FinishScan(ctx, scanID, data, analysisErr); err != nil {
		logger.Warnf("Failed to finish scan persistence: %v", err)
	}
	maybeRecordFirstScanEvidence(ctx, scanCtx, repositoryID, result, analysisErr)
	persistScanSBOM(ctx, scanID, repositoryID, result)
	if data != nil && repositoryID > 0 {
		recordScannerHealthFromScan(ctx, repositoryID, scanID, data.ScannerResults)
	}
	if reportOnlyDryRunFromContext(ctx) && scanRecorder.Enabled() && analysisErr == nil {
		if bs, ok := rdStore.(interface {
			UpdateScanPipelineState(context.Context, string, string, map[string]any) error
		}); ok {
			_ = bs.UpdateScanPipelineState(ctx, scanID, store.ScanStatusCompleted, map[string]any{
				"dry_run_report_only": true,
			})
		}
		if result != nil && repositoryID > 0 {
			emitReportOnlyDryRun(ctx, repositoryID, scanID, len(result.Issues))
		}
	}
	notifyScanFinish(ctx, scanCtx, repositoryID, result, analysisErr)
	maybeEnqueueOpenClawReview(ctx, scanCtx, repositoryID, result, analysisErr)
}

func postAnalysisContext(parent context.Context) (context.Context, context.CancelFunc) {
	if parent == nil {
		parent = context.Background()
	}
	return context.WithTimeout(context.WithoutCancel(parent), 30*time.Minute)
}

func repoClientForForge(forgeType string) forge.RepoClient {
	if normalizeForgeType(forgeType) == store.ForgeTypeGitHub {
		return githubClient
	}
	if giteaClient != nil {
		return giteaClient.AsForgeClient()
	}
	return nil
}

func createIssuesFromResult(ctx context.Context, forgeType, owner, repo string, result *analyzers.AnalysisResult, contextLabel, commit string, prNumber int, repositoryID int64, effective store.EffectiveSettings) {
	if result == nil {
		return
	}

	postCtx, cancel := postAnalysisContext(ctx)
	defer cancel()

	repository := fmt.Sprintf("%s/%s", owner, repo)
	issues.EnrichIssues(repository, result.ScanID, result.Issues)
	loadSuppressionPolicy(postCtx, repositoryID)
	actionIssues := filterIssuesWithSuppression(repositoryID, result.Issues)

	commitRef := commit
	if commitRef == "" {
		commitRef = result.CommitSHA
	}

	var processed []issues.ProcessedIssueRecord
	var findingIDs map[string]int64

	// Phase 1: persist findings + instances before any forge API calls.
	if scanRecorder != nil && scanRecorder.Enabled() && repositoryID > 0 && result.ScanID != "" {
		var err error
		findingIDs, err = scanRecorder.RecordFindings(postCtx, repositoryID, result.ScanID, result.Issues)
		if err != nil {
			logger.Errorf("Failed to persist findings for scan %s: %v — skipping issue filing", result.ScanID, err)
			scanRecorder.MarkPersistenceFailed(postCtx, result.ScanID, len(result.Issues), 0, err)
			return
		}
		if !scanRecorder.IsPersistenceComplete(postCtx, result.ScanID) {
			logger.Warnf("Scan %s persistence incomplete — skipping issue filing", result.ScanID)
			return
		}
	}

	// Phase 2: backfill missing mappings, then file forge issues only after persistence is complete.
	forgeType = normalizeForgeType(forgeType)
	forgeReady := (forgeType == store.ForgeTypeGitHub && githubClient != nil) ||
		(forgeType == store.ForgeTypeGitea && giteaClient != nil)
	if store.ShouldCreateForgeIssues(effective) && forgeReady && issueManager != nil {
		backfillReq := &issues.IssueCreationRequest{
			ForgeType:    forgeType,
			Owner:        owner,
			Repository:   repo,
			RepositoryID: repositoryID,
			ScanID:       result.ScanID,
		}
		if backfill, err := issueManager.BackfillMissingMappings(postCtx, backfillReq); err != nil {
			logger.Warnf("External issue mapping backfill failed: %v", err)
		} else if backfill.Backfilled > 0 {
			logger.Infof("Backfilled %d external issue mappings (examined %d)", backfill.Backfilled, backfill.Examined)
		}

		forgeIssues := filterIssuesForForge(actionIssues, effective, config.Reporting)
		if len(forgeIssues) > 0 {
			issueReq := &issues.IssueCreationRequest{
				ForgeType:    forgeType,
				Owner:        owner,
				Repository:   repo,
				RepositoryID: repositoryID,
				AnalysisResult: &ai.CodeAnalysisResult{
					Issues:                forgeIssues,
					OverallScore:          result.OverallScore,
					ScoreComplete:         result.ScoreComplete,
					ScoreIncompleteReason: result.ScoreIncompleteReason,
					ScoreExplanation:      result.ScoreExplanation,
					AnalysisTime:          result.AnalysisTime,
				},
				Context:            contextLabel,
				Commit:             commitRef,
				PullRequest:        prNumber,
				ScanID:             result.ScanID,
				MinIssueConfidence: effective.ConfidenceGate,
				ForceIssueCreation: true,
			}

			issueResult, err := issueManager.CreateIssuesFromAnalysis(postCtx, issueReq)
			if err != nil {
				logger.Errorf("Failed to create issues: %v", err)
			} else {
				if issueResult.BacklogControlActive {
					logger.Infof("%s blocked %d new issues", issues.BacklogControlNote, issueResult.BacklogControlBlocked)
				}
				logger.Infof("Created %d issues, updated %d, skipped %d", issueResult.IssuesCreated, issueResult.IssuesUpdated, issueResult.IssuesSkipped)
				processed = issueResult.ProcessedIssues
			}
		}
	} else {
		logger.Infof("Forge issue creation skipped (policy_level=%s issue_policy=%s)", effective.PolicyLevel, effective.IssuePolicy)
		if scanRecorder != nil && scanRecorder.Enabled() && result.ScanID != "" {
			scanRecorder.MarkIssueSyncSkipped(postCtx, result.ScanID)
		}
	}

	// Phase 3: link external issues in DB after forge filing.
	issueFilingEnabled := store.ShouldCreateForgeIssues(effective) && forgeReady && issueManager != nil
	if scanRecorder != nil && scanRecorder.Enabled() && repositoryID > 0 && result.ScanID != "" {
		if len(processed) > 0 {
			if err := scanRecorder.RecordExternalIssues(postCtx, result.ScanID, forgeType, processed, findingIDs); err != nil {
				logger.Warnf("Failed to link external issues: %v", err)
			}
		} else if issueFilingEnabled {
			// Filing phase ran but nothing new to link (backlog control, all skipped, or no forge candidates).
			scanRecorder.MarkIssueSyncComplete(postCtx, result.ScanID)
		}
		if !reportOnlyDryRunFromContext(ctx) {
			maybeGenerateRemediationPlans(postCtx, repositoryID, actionIssues, processed)
		}
	}
	maybeProcessEvidenceClosure(postCtx, owner, repo, repositoryID, result)
}

func mainRunnerConfig() runner.Config {
	callback := strings.TrimSpace(config.RunnerCallbackBaseURL)
	if pub := strings.TrimSpace(config.PublicURL); pub != "" {
		callback = strings.TrimSuffix(pub, "/")
	} else if callback == "" {
		host := strings.TrimSpace(config.ListenHost)
		if host == "" || host == "0.0.0.0" {
			host = "127.0.0.1"
		}
		port := strings.TrimSpace(config.Port)
		if port == "" {
			port = "8080"
		}
		callback = fmt.Sprintf("http://%s:%s", host, port)
	}
	return runner.Config{
		DelegationEnabled:     config.RunnerDelegationEnabled,
		Mode:                  config.RunnerMode,
		SharedSecret:          config.RunnerSharedSecret,
		JobTimeoutSeconds:     config.RunnerJobTimeoutSeconds,
		MaxConcurrentJobs:     config.RunnerMaxConcurrentJobs,
		ResultMaxSizeMB:       config.RunnerResultMaxSizeMB,
		ArtifactRetentionDays: config.RunnerArtifactRetentionDays,
		CallbackBaseURL:       callback,
		MaxRepoSizeMB:         config.WorkspaceMaxSizeMB,
		MaxFiles:              config.WorkspaceMaxFiles,
		RequireHMAC:           config.RunnerRequireHMAC,
		NonceTTLSeconds:       config.RunnerNonceTTLSeconds,
		AllowedJobTypes:       config.RunnerAllowedJobTypes,
	}
}

func tryDelegateScan(ctx context.Context, scanCtx *store.ScanContext, repo store.Repository, effective store.EffectiveSettings) (bool, error) {
	if runnerDispatcher == nil {
		return false, nil
	}
	if runner.ShouldDelegate(runnerCfg, effective, scanCtx.TriggerType) != runner.DecisionDelegate {
		return false, nil
	}
	policy := analyzers.SnapshotFromPolicy(scanPolicyFromEffective(effective))
	if rdStore != nil {
		settings, serr := rdStore.GetRepoSettings(ctx, repo.ID)
		if serr == nil {
			_, meta := store.ResolveEffectiveSettingsFull(appGlobalSnapshot, settings)
			sp := scanPolicyFromEffective(effective)
			sp.ScanProfile = meta.ScanProfile
			sp.ProfileModified = meta.ProfileModified
			sp.ProfileSource = meta.ProfileSource
			policy = analyzers.SnapshotFromPolicy(sp)
		}
	}
	_, err := runnerDispatcher.CreateScanJob(ctx, repo, scanCtx.ScanID, scanCtx.Ref, scanCtx.CommitSHA, policy)
	if err != nil {
		if runner.ShouldFallbackToCore(effective) {
			logger.Warnf("Runner delegation unavailable, falling back to core scan: %v", err)
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func ingestRunnerResult(ctx context.Context, job store.RunnerJob, result runner.JobResult, repo store.Repository, _ store.EffectiveSettings) error {
	ctx = scanid.With(ctx, job.ScanID)
	forgeType := normalizeForgeType(repo.ForgeType)
	ctx = analyzers.WithForgeType(ctx, forgeType)
	ctx, effective := resolveEffectiveSettingsForRepo(ctx, forgeType, repo.Owner, repo.Name)

	var policy analyzers.PolicySnapshot
	if err := json.Unmarshal(job.PolicySnapshotJSON, &policy); err != nil {
		logger.Warnf("runner result ingest: invalid policy snapshot for job %s: %v", job.JobID, err)
	}

	if result.Status == runner.JobStatusFailed || job.Status == store.RunnerJobStatusFailed {
		errMsg := job.Error
		if len(result.Errors) > 0 {
			errMsg = result.Errors[0]
		}
		if errMsg == "" {
			errMsg = "runner job failed"
		}
		runnerScanCtx := &store.ScanContext{
			Owner: repo.Owner, Repo: repo.Name, ScanID: job.ScanID,
			TriggerType: store.TriggerScheduled, Ref: job.Ref, CommitSHA: job.CommitSHA, PRNumber: job.PRNumber,
		}
		finishPersistedScan(ctx, runnerScanCtx, repo.ID, nil, fmt.Errorf("%s", errMsg))
		notifyRunnerJobFailed(ctx, repo.ID, repo.FullName, job.ScanID, errMsg)
		return nil
	}

	analysisResult := result.ToAnalysisResult(repo.FullName, job.Ref, &policy)
	runnerScanCtx := &store.ScanContext{
		Owner: repo.Owner, Repo: repo.Name, ScanID: job.ScanID,
		TriggerType: store.TriggerScheduled, Ref: job.Ref, CommitSHA: job.CommitSHA, PRNumber: job.PRNumber,
	}
	finishPersistedScan(ctx, runnerScanCtx, repo.ID, analysisResult, nil)
	ingestContainerScanResult(ctx, job, result, repo)
	if config.ContainerScan.Normalized().CreateIssues && job.JobType == store.RunnerJobTypeContainerImageScan {
		createIssuesFromResult(ctx, forgeType, repo.Owner, repo.Name, analysisResult,
			fmt.Sprintf("Container image scan (%s)", job.JobType), job.Ref, job.PRNumber, repo.ID, effective)
	} else if job.JobType != store.RunnerJobTypeContainerImageScan {
		createIssuesFromResult(ctx, forgeType, repo.Owner, repo.Name, analysisResult,
			fmt.Sprintf("Runner scan (%s)", job.JobType), job.Ref, job.PRNumber, repo.ID, effective)
	}
	return nil
}

func runnerNonceMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if runnerReceiver == nil {
			c.Next()
			return
		}
		nonce := c.GetHeader(runner.HeaderNonce)
		if err := runnerReceiver.CheckNonce(c.Request.Context(), nonce); err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "runner nonce rejected"})
			return
		}
		c.Next()
	}
}

type manualAnalysisRequest struct {
	ForgeType        string `json:"forge_type"` // gitea (default) or github
	Owner            string `json:"owner" binding:"required"`
	Repository       string `json:"repository" binding:"required"`
	Ref              string `json:"ref"`
	Type             string `json:"type"` // "repository" or "pull_request"
	PRNumber         int    `json:"pr_number"`
	ScanProfile      string `json:"scan_profile"`
	ReportOnlyDryRun bool   `json:"report_only_dry_run"`
	ScanID           string `json:"scan_id,omitempty"`
}

func normalizeForgeType(forgeType string) string {
	forgeType = strings.ToLower(strings.TrimSpace(forgeType))
	if forgeType == "" {
		return store.ForgeTypeGitea
	}
	if forgeType == store.ForgeTypeGitHub {
		return store.ForgeTypeGitHub
	}
	return store.ForgeTypeGitea
}

func enqueueManualAnalysis(parentCtx context.Context, req manualAnalysisRequest) {
	// Detach from HTTP request context so bulk /analyze/all scans are not cancelled when the handler returns.
	go func() {
		forgeType := normalizeForgeType(req.ForgeType)
		analysisCtx := withReportOnlyDryRun(context.WithoutCancel(parentCtx), req.ReportOnlyDryRun)
		runAnalysis(withScanProfileOverride(analysisCtx, req.ScanProfile), func(ctx context.Context) {
			var result *analyzers.AnalysisResult
			var err error
			var repositoryID int64

			scanCtx := store.ScanContext{
				Owner:         req.Owner,
				Repo:          req.Repository,
				ForgeType:     forgeType,
				TriggerType:   store.TriggerManual,
				Ref:           req.Ref,
				ConnectedRepo: true,
				ScanID:        strings.TrimSpace(req.ScanID),
			}
			ctx, repositoryID = beginPersistedScan(ctx, &scanCtx)
			ctx = analyzers.WithForgeType(ctx, forgeType)
			ctx, effective := resolveEffectiveSettingsForRepo(ctx, forgeType, req.Owner, req.Repository)
			if !effective.Enabled {
				postCtx, postCancel := postAnalysisContext(ctx)
				defer postCancel()
				finishPersistedScan(postCtx, &scanCtx, repositoryID, nil, fmt.Errorf("repository disabled in settings"))
				logger.Infof("Manual scan skipped — repository %s/%s disabled in settings", req.Owner, req.Repository)
				return
			}

			if req.Type != "pull_request" || req.PRNumber <= 0 {
				if rdStore != nil && repositoryID > 0 {
					if dbRepo, rerr := rdStore.GetRepository(ctx, repositoryID); rerr == nil {
						if delegated, derr := tryDelegateScan(ctx, &scanCtx, dbRepo, effective); delegated {
							logger.WithFields(logrus.Fields{
								"scan_id": scanCtx.ScanID, "repo": dbRepo.FullName,
							}).Info("Manual scan delegated to runner")
							return
						} else if derr != nil {
							finishPersistedScan(ctx, &scanCtx, repositoryID, nil, derr)
							return
						}
					}
				}
			}

			if req.Type == "pull_request" && req.PRNumber > 0 {
				if forgeType == store.ForgeTypeGitHub {
					err = fmt.Errorf("pull request analysis is not supported for GitHub yet")
					postCtx, postCancel := postAnalysisContext(ctx)
					defer postCancel()
					finishPersistedScan(postCtx, &scanCtx, repositoryID, nil, err)
					logger.Warnf("Manual analysis: %v", err)
					return
				}
				scanCtx.PRNumber = req.PRNumber
				scanCtx.TriggerType = store.TriggerPR
				result, err = analysisEngine.AnalyzePullRequest(ctx, req.Owner, req.Repository, req.PRNumber)
			} else {
				ref := strings.TrimSpace(req.Ref)
				if ref == "" {
					ref = "main"
				}
				if client := repoClientForForge(forgeType); client != nil {
					if resolved, rerr := client.ResolveRef(ctx, req.Owner, req.Repository, ref); rerr == nil {
						ref = resolved
					} else {
						logger.Warnf("Could not resolve ref for %s/%s: %v", req.Owner, req.Repository, rerr)
					}
				}
				scanCtx.Ref = ref
				result, err = analysisEngine.AnalyzeRepository(ctx, req.Owner, req.Repository, ref)
			}

			postCtx, postCancel := postAnalysisContext(ctx)
			defer postCancel()
			finishPersistedScan(postCtx, &scanCtx, repositoryID, result, err)
			if err != nil {
				logger.Errorf("Manual analysis failed: %v", err)
				return
			}

			if len(result.Issues) > 0 || scanRecorder != nil && scanRecorder.Enabled() {
				createIssuesFromResult(postCtx, forgeType, req.Owner, req.Repository, result,
					fmt.Sprintf("Manual analysis - %s", req.Type), req.Ref, req.PRNumber, repositoryID, effective)
			}
		})
	}()
}

// handleManualAnalysis handles manual analysis requests
func handleManualAnalysis(c *gin.Context) {
	var req manualAnalysisRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}
	if strings.TrimSpace(req.ScanID) == "" {
		req.ScanID = scanid.New()
	}

	forgeType := normalizeForgeType(req.ForgeType)
	if skip, reason := checkScanAdmission(c.Request.Context(), forgeType, req.Owner, req.Repository, store.TriggerManual); skip {
		c.JSON(http.StatusConflict, gin.H{
			"error":   reason,
			"status":  "skipped",
			"scan_id": req.ScanID,
		})
		return
	}

	enqueueManualAnalysis(c.Request.Context(), req)
	c.JSON(http.StatusOK, gin.H{
		"status":              "analysis started",
		"scan_id":             req.ScanID,
		"report_only_dry_run": req.ReportOnlyDryRun,
		"trigger_type":        store.TriggerManual,
	})
}

type bulkForgeResult struct {
	Queued  []string `json:"queued"`
	Skipped []string `json:"skipped"`
	Error   string   `json:"error,omitempty"`
}

// collectBulkRepos lists user + org repos from a forge client, deduped by full name.
func collectBulkRepos(ctx context.Context, client forge.RepoClient, orgs []string, envOrgKey string) (map[string]forge.RepositorySummary, error) {
	if client == nil {
		return nil, nil
	}
	reposByName := make(map[string]forge.RepositorySummary)
	userRepos, err := client.ListAllUserRepositories(ctx)
	if err != nil {
		return nil, fmt.Errorf("list user repositories: %w", err)
	}
	for _, repo := range userRepos {
		if name := strings.TrimSpace(repo.FullName); name != "" {
			reposByName[name] = repo
		}
	}
	if len(orgs) == 0 {
		if envOrgs := strings.TrimSpace(os.Getenv(envOrgKey)); envOrgs != "" {
			for _, org := range strings.Split(envOrgs, ",") {
				if o := strings.TrimSpace(org); o != "" {
					orgs = append(orgs, o)
				}
			}
		}
	}
	for _, org := range orgs {
		orgRepos, err := client.ListAllOrgRepositories(ctx, org)
		if err != nil {
			return nil, fmt.Errorf("list org %s repositories: %w", org, err)
		}
		for _, repo := range orgRepos {
			if name := strings.TrimSpace(repo.FullName); name != "" {
				reposByName[name] = repo
			}
		}
	}
	return reposByName, nil
}

func queueBulkForgeRepos(parentCtx context.Context, forgeType string, repos map[string]forge.RepositorySummary, defaultRef, scanProfile string, dryRun bool) (queued, skipped []string) {
	prefix := normalizeForgeType(forgeType) + ":"
	for _, repo := range repos {
		fullName := strings.TrimSpace(repo.FullName)
		if fullName == "" {
			continue
		}
		if !handlers.RepoAllowed(fullName, config.RepositoryIncludePatterns, config.RepositoryExcludePatterns) {
			skipped = append(skipped, prefix+fullName)
			continue
		}
		parts := strings.SplitN(fullName, "/", 2)
		if len(parts) != 2 {
			skipped = append(skipped, prefix+fullName)
			continue
		}
		ref := strings.TrimSpace(defaultRef)
		if ref == "" {
			ref = strings.TrimSpace(repo.DefaultBranch)
		}
		if ref == "" {
			ref = "main"
		}
		queued = append(queued, prefix+fullName)
		if dryRun {
			continue
		}
		enqueueManualAnalysis(parentCtx, manualAnalysisRequest{
			ForgeType:   forgeType,
			Owner:       parts[0],
			Repository:  parts[1],
			Ref:         ref,
			Type:        "repository",
			ScanProfile: strings.TrimSpace(scanProfile),
		})
	}
	return queued, skipped
}

// handleBulkAnalysis queues full-repository scans for every repo visible to configured forge tokens.
func handleBulkAnalysis(c *gin.Context) {
	var req struct {
		Orgs        []string `json:"orgs"`
		Ref         string   `json:"ref"`
		ScanProfile string   `json:"scan_profile"`
		DryRun      bool     `json:"dry_run"`
		Forge       string   `json:"forge"` // gitea, github, or all (default)
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	if giteaClient == nil && githubClient == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "no forge client configured (set gitea_token and/or github_token)"})
		return
	}

	listCtx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Minute)
	defer cancel()

	forgeFilter := strings.ToLower(strings.TrimSpace(req.Forge))
	if forgeFilter == "" {
		forgeFilter = "all"
	}

	var giteaResult, githubResult bulkForgeResult
	defaultRef := strings.TrimSpace(req.Ref)
	scanProfile := strings.TrimSpace(req.ScanProfile)

	if forgeFilter == "all" || forgeFilter == "gitea" {
		if giteaClient != nil {
			repos, err := collectBulkRepos(listCtx, giteaClient.AsForgeClient(), req.Orgs, "GITEA_SCAN_ORGS")
			if err != nil {
				giteaResult.Error = err.Error()
			} else {
				giteaResult.Queued, giteaResult.Skipped = queueBulkForgeRepos(c.Request.Context(), store.ForgeTypeGitea, repos, defaultRef, scanProfile, req.DryRun)
			}
		} else if forgeFilter == "gitea" {
			giteaResult.Error = "Gitea client not configured"
		}
	}

	if forgeFilter == "all" || forgeFilter == "github" {
		if githubClient != nil {
			repos, err := collectBulkRepos(listCtx, githubClient, req.Orgs, "GITHUB_SCAN_ORGS")
			if err != nil {
				githubResult.Error = err.Error()
			} else {
				githubResult.Queued, githubResult.Skipped = queueBulkForgeRepos(c.Request.Context(), store.ForgeTypeGitHub, repos, defaultRef, scanProfile, req.DryRun)
			}
		} else if forgeFilter == "github" {
			githubResult.Error = "GitHub client not configured"
		}
	}

	allQueued := append(append([]string{}, giteaResult.Queued...), githubResult.Queued...)
	allSkipped := append(append([]string{}, giteaResult.Skipped...), githubResult.Skipped...)

	c.JSON(http.StatusOK, gin.H{
		"status":        "bulk analysis queued",
		"forge":         forgeFilter,
		"queued_count":  len(allQueued),
		"skipped_count": len(allSkipped),
		"scan_profile":  scanProfile,
		"queued":        allQueued,
		"skipped":       allSkipped,
		"dry_run":       req.DryRun,
		"gitea":         giteaResult,
		"github":        githubResult,
	})
}

// handleStatus handles status requests
func handleStatus(c *gin.Context) {
	r := buildReadiness("running")
	c.JSON(http.StatusOK, r)
}

func handleAbout(c *gin.Context) {
	projectURL := "https://git.commsnet.org/commstech/Repository-Detective"
	c.JSON(http.StatusOK, gin.H{
		"product_name":        "Repository Detective",
		"tagline":             "Inspect. Analyze. Improve.",
		"version":             version,
		"commit":              commit,
		"build_date":          buildDate,
		"api_base_path":       "/api/v1",
		"openapi_url":         "/api/v1/openapi.yaml",
		"documentation_index": projectURL + "/src/branch/main/docs/README.md",
		"agent_docs_url":      projectURL + "/src/branch/main/docs/AGENT_QUICKSTART.md",
		"mcp_docs_url":        projectURL + "/src/branch/main/docs/MCP.md",
		"openclaw_docs_url":   projectURL + "/src/branch/main/docs/OPENCLAW_INTEGRATION.md",
		"project_url":         projectURL,
		"auth_headers": []string{
			"X-Repository-Detective-API-Key",
			"Authorization: Bearer <key>",
		},
		"mcp_command": "go build -o repository-detective-mcp ./cmd/repository-detective-mcp",
		"safe_loop":   "detect → issue → plan → approve → patch PR → merge → rescan → verified closure",
	})
}

func handleOpenAPI(c *gin.Context) {
	c.Header("Content-Type", "application/yaml; charset=utf-8")
	c.Header("Cache-Control", "public, max-age=300")
	c.Data(http.StatusOK, "application/yaml; charset=utf-8", docsdata.OpenAPIYAML)
}

// handleConfigReload handles configuration reload requests
func handleConfigReload(c *gin.Context) {
	if err := loadConfig(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Reinitialize components with new config
	if err := initializeComponents(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to reinitialize components"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "configuration reloaded"})
}

func (c *Config) effectiveAIProvider() string {
	if c.AIProvider != "" {
		return c.AIProvider
	}
	if c.OpenWebUIURL != "" {
		return string(ai.ProviderOpenWebUI)
	}
	return ""
}

// needsAIProvider reports whether an AI backend must be configured at startup.
func (c *Config) needsAIProvider() bool {
	depth := c.AnalysisDepth
	if depth <= 0 {
		depth = 3
	}
	return depth >= 3 && c.EnableLLMAuditors
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

func parseCommaSeparatedOrgs(raw string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	var out []string
	for _, part := range strings.Split(raw, ",") {
		if org := strings.TrimSpace(part); org != "" {
			out = append(out, org)
		}
	}
	return out
}

func mainHealthConfig() health.Config {
	return health.Config{
		Enabled:               config.EnableHealthChecks,
		EnableTechDebt:        config.EnableTechDebtChecks,
		EnableReliability:     config.EnableReliabilityChecks,
		EnableMaintainability: config.EnableMaintainabilityChecks,
		EnableTestGap:         config.EnableTestGapChecks,
		EnablePerformance:     config.EnablePerformanceChecks,
		EnableAIRisk:          config.EnableAIRiskChecks,
		MaxFindings:           config.HealthMaxFindings,
		LargeFileLines:        config.HealthLargeFileLines,
		LargeFunctionLines:    config.HealthLargeFunctionLines,
		MaxNestingDepth:       config.HealthMaxNestingDepth,
		MaxFunctionParams:     config.HealthMaxFunctionParams,
	}
}

func mainGraphConfig() graph.Config {
	return graph.Config{
		Enabled:          config.EnableCodeGraph,
		MaxNodes:         config.GraphMaxNodes,
		MaxEdges:         config.GraphMaxEdges,
		TimeoutSeconds:   config.GraphTimeoutSeconds,
		IncludeFunctions: config.GraphIncludeFunctions,
		IncludeFindings:  config.GraphIncludeFindings,
	}
}

func mainScannerConfig() scanners.Config {
	cfg := scanners.Config{
		EnableTrivy:                     config.EnableTrivy,
		EnableGrype:                     config.EnableGrype,
		EnableGitleaks:                  config.EnableGitleaks,
		EnableSemgrep:                   config.EnableSemgrep,
		EnableGovulncheck:               config.EnableGovulncheck,
		EnableGosec:                     config.EnableGosec,
		EnableStaticcheck:               config.EnableStaticcheck,
		EnableHadolint:                  config.EnableHadolint,
		EnableCheckov:                   config.EnableCheckov,
		EnableLinters:                   config.EnableLinters,
		GitleaksConfig:                  config.GitleaksConfig,
		GitleaksTimeoutSeconds:          config.GitleaksTimeoutSeconds,
		SecretScanGitHistoryEnabled:     config.SecretScanGitHistoryEnabled,
		SecretScanHistoryMaxCommits:     config.SecretScanHistoryMaxCommits,
		SecretScanRecentCommitsMax:      config.SecretScanRecentCommitsMax,
		SecretScanHistoryTimeoutSeconds: config.SecretScanHistoryTimeoutSeconds,
		SecretScanHistoryReportOnly:     config.SecretScanHistoryReportOnlyForPreinstall,
		SecretScanRedact:                config.SecretScanRedact,
		SemgrepConfig:                   config.SemgrepConfig,
		SemgrepTimeoutSeconds:           config.SemgrepTimeoutSeconds,
		SemgrepMaxFindings:              config.SemgrepMaxFindings,
		SemgrepSeverityThreshold:        config.SemgrepSeverityThreshold,
		GovulncheckTimeoutSeconds:       config.GovulncheckTimeoutSeconds,
		GosecTimeoutSeconds:             config.GosecTimeoutSeconds,
		StaticcheckTimeoutSeconds:       config.StaticcheckTimeoutSeconds,
		GoScannerMaxFindings:            config.GoScannerMaxFindings,
		HadolintTimeoutSeconds:          config.HadolintTimeoutSeconds,
		CheckovTimeoutSeconds:           config.CheckovTimeoutSeconds,
		IACScannerMaxFindings:           config.IACScannerMaxFindings,
		TrivySeverity:                   "HIGH,CRITICAL",
		GrypeFailOn:                     "high",
		LinterMinSeverity:               "warning",
		TimeoutSeconds:                  config.ScannerTimeoutSeconds,
	}
	return scanners.ApplyRuntimeAvailability(cfg, logger)
}

func registerControlPlaneRoutes(router *gin.Engine) {
	if controlPlaneHandler == nil {
		return
	}
	cp := router.Group("/api/v1")
	cp.Use(requireComponentsReady(), requireAPIKeyAuth())
	controlPlaneHandler.RegisterRoutes(cp)
	if runnerHandler != nil {
		runnerHandler.RegisterOperatorRoutes(cp)
	}
	if preinstallHandler != nil {
		preinstallHandler.RegisterRoutes(cp)
	}
	if notifyManager != nil {
		api.NewNotificationHandler(notifyManager).RegisterRoutes(cp)
	}
	if config.RemediationPlannerEnabled {
		api.NewRemediationHandler(rdStore, remediationBridge{}, config.RemediationPREnabled).RegisterRoutes(cp)
	}
	if config.EvidenceClosureEnabled {
		api.NewClosureHandler(rdStore, closureBridge{}).RegisterRoutes(cp)
	}
	if rdStore != nil {
		api.NewSuppressionsHandler(rdStore, suppressionBridge{}).RegisterRoutes(cp)
	}
	if config.IssueReconciliationEnabled && rdStore != nil {
		api.NewReconcileHandler(rdStore, reconcileBridge{}).RegisterRoutes(cp)
	}
	if config.CalibrationEnabled && rdStore != nil {
		api.NewCalibrationHandler(rdStore, calibrationBridge{}).RegisterRoutes(cp)
	}
	if rdStore != nil {
		api.NewContainerHandler(rdStore, containerScanBridge{}).RegisterRoutes(cp)
	}
	api.NewAIHandler(aiStatusBridge{}).RegisterRoutes(cp)
	registerDoctorAPI(cp)
	if rdStore != nil {
		api.NewOpenClawReviewHandler(openclawReviewBridge{}).RegisterRoutes(cp)
	}

	if runnerHandler != nil && runnerCfg.SharedSecret != "" {
		rg := router.Group("/api/v1/runner")
		rg.Use(requireComponentsReady(), runnerNonceMiddleware(), api.RequireRunnerHMAC(runnerCfg.SharedSecret))
		runnerHandler.RegisterRunnerRoutes(rg)
	}

	if !config.UIEnabled || operatorUI == nil {
		return
	}
	uiGroup := router.Group(operatorUI.BasePath())
	uiGroup.Use(requireComponentsReady())
	uiGroup.Use(operatorUI.UIAPIKeyCookieMiddleware())
	operatorUI.RegisterPublicRoutes(uiGroup)
	if config.AuthMode == "local" {
		operatorUI.RegisterAuthRoutes(uiGroup)
		uiAuth := uiGroup.Group("")
		uiAuth.Use(operatorUI.SessionAuthMiddleware())
		operatorUI.RegisterRoutes(uiAuth)
	} else {
		uiAuth := uiGroup.Group("")
		uiAuth.Use(operatorUI.APIKeyAuthMiddleware())
		operatorUI.RegisterRoutes(uiAuth)
	}
	logger.Infof("Control plane API and UI routes registered")
}
