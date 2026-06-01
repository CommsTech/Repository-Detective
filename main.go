package main

import (
	"context"
	"encoding/json"
	"crypto/hmac"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync/atomic"
	"syscall"
	"time"

	"git.commsnet.org/commstech/bugbot/ai"
	"git.commsnet.org/commstech/bugbot/analyzers"
	"git.commsnet.org/commstech/bugbot/api"
	"git.commsnet.org/commstech/bugbot/gitea"
	"git.commsnet.org/commstech/bugbot/graph"
	"git.commsnet.org/commstech/bugbot/handlers"
	"git.commsnet.org/commstech/bugbot/health"
	"git.commsnet.org/commstech/bugbot/internal/config/envcompat"
	"git.commsnet.org/commstech/bugbot/internal/scanid"
	"git.commsnet.org/commstech/bugbot/internal/security"
	"git.commsnet.org/commstech/bugbot/issues"
	"git.commsnet.org/commstech/bugbot/limiter"
	"git.commsnet.org/commstech/bugbot/memory/qdrant"
	"git.commsnet.org/commstech/bugbot/operator"
	"git.commsnet.org/commstech/bugbot/orch"
	"git.commsnet.org/commstech/bugbot/preinstall"
	"git.commsnet.org/commstech/bugbot/runner"
	"git.commsnet.org/commstech/bugbot/scanners"
	"git.commsnet.org/commstech/bugbot/store"
	"git.commsnet.org/commstech/bugbot/ui"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

var version = "dev"

var componentsReady atomic.Bool

type scanProfileOverrideKey struct{}

var (
	logger              *logrus.Logger
	config              *Config
	giteaClient         *gitea.Client
	aiClient            *ai.Client
	analysisEngine      *analyzers.Engine
	issueManager        *issues.Manager
	statusReporter      *gitea.StatusReporter
	webhookHandler      *handlers.WebhookHandler
	onboardingHandler   *handlers.OnboardingHandler
	analysisLimiter     *limiter.ConcurrencyLimiter
	bugbotStore         store.QueryStore
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
	runnerHandler       *api.RunnerHandler
)

// Config holds the plugin configuration
type Config struct {
	Port                              string            `mapstructure:"port"`
	APIKey                            string            `mapstructure:"api_key"` // API key for manual analysis endpoints
	GiteaURL                          string            `mapstructure:"gitea_url"`
	GiteaToken                        string            `mapstructure:"gitea_token"`
	WebhookSecret                     string            `mapstructure:"webhook_secret"`
	AllowInsecureWebhooks             bool              `mapstructure:"allow_insecure_webhooks"`
	AIProvider                        string            `mapstructure:"ai_provider"`
	AIBaseURL                         string            `mapstructure:"ai_base_url"`
	AIAPIKey                          string            `mapstructure:"ai_api_key"`
	AIModel                           string            `mapstructure:"ai_model"`
	AIInsecureSkipTLSVerify           bool              `mapstructure:"ai_insecure_skip_tls_verify"`
	OpenWebUIURL                      string            `mapstructure:"openwebui_url"`
	OpenWebUIToken                    string            `mapstructure:"openwebui_token"`
	OpenWebUIModel                    string            `mapstructure:"openwebui_model"`
	LogLevel                          string            `mapstructure:"log_level"`
	AnalysisDepth                     int               `mapstructure:"analysis_depth"`
	MaxFileSize                       int64             `mapstructure:"max_file_size"`
	EnableSecurity                    bool              `mapstructure:"enable_security"`
	EnableQuality                     bool              `mapstructure:"enable_quality"`
	EnableLLMAuditors                 bool              `mapstructure:"enable_llm_auditors"`
	EnableTrivy                       bool              `mapstructure:"enable_trivy"`
	EnableGrype                       bool              `mapstructure:"enable_grype"`
	EnableGitleaks                    bool              `mapstructure:"enable_gitleaks"`
	EnableSemgrep                     bool              `mapstructure:"enable_semgrep"`
	EnableGovulncheck                 bool              `mapstructure:"enable_govulncheck"`
	EnableGosec                       bool              `mapstructure:"enable_gosec"`
	EnableStaticcheck                 bool              `mapstructure:"enable_staticcheck"`
	EnableHadolint                    bool              `mapstructure:"enable_hadolint"`
	EnableCheckov                     bool              `mapstructure:"enable_checkov"`
	EnableLinters                     bool              `mapstructure:"enable_linters"`
	ScanProfile                       string            `mapstructure:"scan_profile"`
	GitleaksConfig                    string            `mapstructure:"gitleaks_config"`
	GitleaksTimeoutSeconds            int               `mapstructure:"gitleaks_timeout_seconds"`
	SemgrepConfig                     string            `mapstructure:"semgrep_config"`
	SemgrepTimeoutSeconds             int               `mapstructure:"semgrep_timeout_seconds"`
	SemgrepMaxFindings                int               `mapstructure:"semgrep_max_findings"`
	SemgrepSeverityThreshold          string            `mapstructure:"semgrep_severity_threshold"`
	GovulncheckTimeoutSeconds         int               `mapstructure:"govulncheck_timeout_seconds"`
	GosecTimeoutSeconds               int               `mapstructure:"gosec_timeout_seconds"`
	StaticcheckTimeoutSeconds         int               `mapstructure:"staticcheck_timeout_seconds"`
	GoScannerMaxFindings              int               `mapstructure:"go_scanner_max_findings"`
	HadolintTimeoutSeconds            int               `mapstructure:"hadolint_timeout_seconds"`
	CheckovTimeoutSeconds             int               `mapstructure:"checkov_timeout_seconds"`
	IACScannerMaxFindings             int               `mapstructure:"iac_scanner_max_findings"`
	ScannerTimeoutSeconds             int               `mapstructure:"scanner_timeout_seconds"`
	MinIssueConfidence                float64           `mapstructure:"min_issue_confidence"`
	QdrantEnabled                     bool              `mapstructure:"qdrant_enabled"`
	QdrantURL                         string            `mapstructure:"qdrant_url"`
	QdrantAPIKey                      string            `mapstructure:"qdrant_api_key"`
	QdrantCollection                  string            `mapstructure:"qdrant_collection"`
	QdrantVectorSize                  int               `mapstructure:"qdrant_vector_size"`
	QdrantSimilarityThreshold         float64           `mapstructure:"qdrant_similarity_threshold"`
	EmbeddingModel                    string            `mapstructure:"embedding_model"`
	EmbeddingBaseURL                  string            `mapstructure:"embedding_base_url"`
	EmbeddingAPIKey                   string            `mapstructure:"embedding_api_key"`
	AutoCreateIssues                  bool              `mapstructure:"auto_create_issues"`
	MaxIssuesPerRun                   int               `mapstructure:"max_issues_per_run"`
	SkipLowSeverity                   bool              `mapstructure:"skip_low_severity"`
	GroupSimilarIssues                bool              `mapstructure:"group_similar_issues"`
	SkipPatterns                      []string          `mapstructure:"-"`
	LanguageMapping                   map[string]string `mapstructure:"-"`
	RepositoryIncludePatterns         []string          `mapstructure:"-"`
	RepositoryExcludePatterns         []string          `mapstructure:"-"`
	PublicURL                         string            `mapstructure:"public_url"`
	ListenHost                        string            `mapstructure:"listen_host"`
	StartupCheckTimeout               int               `mapstructure:"startup_check_timeout"`
	MaxConcurrentAnalyses             int               `mapstructure:"max_concurrent_analyses"`
	AnalysisTimeout                   int               `mapstructure:"analysis_timeout"`
	RateLimitPerMinute                int               `mapstructure:"rate_limit_per_minute"`
	WorkspaceMode                     string            `mapstructure:"workspace_mode"`
	WorkspaceMaxSizeMB                int               `mapstructure:"workspace_max_size_mb"`
	WorkspaceMaxFiles                 int               `mapstructure:"workspace_max_files"`
	WorkspaceArchiveTimeoutSeconds    int               `mapstructure:"workspace_archive_timeout_seconds"`
	EnableGiteaStatus                 bool              `mapstructure:"enable_gitea_status"`
	GiteaStatusContext                string            `mapstructure:"gitea_status_context"`
	GiteaStatusFailOn                 string            `mapstructure:"gitea_status_fail_on"`
	GiteaStatusWarnOn                 string            `mapstructure:"gitea_status_warn_on"`
	GiteaStatusIncludeScannerFailures bool              `mapstructure:"gitea_status_include_scanner_failures"`
	SkipStartupChecks                 bool              `mapstructure:"skip_startup_checks"`
	DatabaseEnabled                   bool              `mapstructure:"database_enabled"`
	DatabaseDriver                    string            `mapstructure:"database_driver"`
	DatabasePath                      string            `mapstructure:"database_path"`
	DatabaseDSN                       string            `mapstructure:"database_dsn"`
	UIEnabled                         bool              `mapstructure:"ui_enabled"`
	UIBasePath                        string            `mapstructure:"ui_base_path"`
	SchedulerEnabled                  bool              `mapstructure:"scheduler_enabled"`
	SchedulerPollIntervalSeconds      int               `mapstructure:"scheduler_poll_interval_seconds"`
	SchedulerMaxConcurrentScans       int               `mapstructure:"scheduler_max_concurrent_scans"`
	PreinstallAuditEnabled            bool              `mapstructure:"preinstall_audit_enabled"`
	PreinstallAllowPrivateNetworks    bool              `mapstructure:"preinstall_allow_private_networks"`
	PreinstallMaxRepoSizeMB           int               `mapstructure:"preinstall_max_repo_size_mb"`
	PreinstallMaxFiles                int               `mapstructure:"preinstall_max_files"`
	PreinstallTimeoutSeconds          int               `mapstructure:"preinstall_timeout_seconds"`
	PreinstallMaxFindings             int               `mapstructure:"preinstall_max_findings"`
	PreinstallAllowGitClone           bool              `mapstructure:"preinstall_allow_git_clone"`
	EnableHealthChecks                bool              `mapstructure:"enable_health_checks"`
	EnableTechDebtChecks              bool              `mapstructure:"enable_tech_debt_checks"`
	EnableReliabilityChecks           bool              `mapstructure:"enable_reliability_checks"`
	EnableMaintainabilityChecks       bool              `mapstructure:"enable_maintainability_checks"`
	EnableTestGapChecks               bool              `mapstructure:"enable_test_gap_checks"`
	EnablePerformanceChecks           bool              `mapstructure:"enable_performance_checks"`
	EnableAIRiskChecks                bool              `mapstructure:"enable_ai_risk_checks"`
	HealthMaxFindings                 int               `mapstructure:"health_max_findings"`
	HealthLargeFileLines              int               `mapstructure:"health_large_file_lines"`
	HealthLargeFunctionLines          int               `mapstructure:"health_large_function_lines"`
	HealthMaxNestingDepth             int               `mapstructure:"health_max_nesting_depth"`
	HealthMaxFunctionParams           int               `mapstructure:"health_max_function_params"`
	EnableCodeGraph                   bool              `mapstructure:"enable_code_graph"`
	GraphMaxNodes                     int               `mapstructure:"graph_max_nodes"`
	GraphMaxEdges                     int               `mapstructure:"graph_max_edges"`
	GraphTimeoutSeconds               int               `mapstructure:"graph_timeout_seconds"`
	GraphIncludeFunctions             bool              `mapstructure:"graph_include_functions"`
	GraphIncludeFindings              bool              `mapstructure:"graph_include_findings"`
	RunnerDelegationEnabled           bool              `mapstructure:"runner_delegation_enabled"`
	RunnerMode                        string            `mapstructure:"runner_mode"`
	RunnerSharedSecret                string            `mapstructure:"runner_shared_secret"`
	RunnerJobTimeoutSeconds           int               `mapstructure:"runner_job_timeout_seconds"`
	RunnerMaxConcurrentJobs           int               `mapstructure:"runner_max_concurrent_jobs"`
	RunnerResultMaxSizeMB             int               `mapstructure:"runner_result_max_size_mb"`
	RunnerArtifactRetentionDays       int               `mapstructure:"runner_artifact_retention_days"`
	RunnerCallbackBaseURL             string            `mapstructure:"runner_callback_base_url"`
	LabelCompatMode                   string            `mapstructure:"label_compat_mode"`
	NotificationsEnabled              bool              `mapstructure:"notifications_enabled"`
	NotificationMinSeverity           string            `mapstructure:"notification_min_severity"`
	NotificationCooldownSeconds       int               `mapstructure:"notification_cooldown_seconds"`
	TelegramEnabled                   bool              `mapstructure:"telegram_enabled"`
	TelegramBotToken                  string            `mapstructure:"telegram_bot_token"`
	TelegramChatID                    string            `mapstructure:"telegram_chat_id"`
	SlackEnabled                      bool              `mapstructure:"slack_enabled"`
	SlackWebhookURL                   string            `mapstructure:"slack_webhook_url"`
	DiscordEnabled                    bool              `mapstructure:"discord_enabled"`
	DiscordWebhookURL                 string            `mapstructure:"discord_webhook_url"`
	WebhookNotificationsEnabled       bool              `mapstructure:"webhook_notifications_enabled"`
	WebhookNotificationURL            string            `mapstructure:"webhook_notification_url"`
	WebhookNotificationSecret         string            `mapstructure:"webhook_notification_secret"`
	RemediationPlannerEnabled         bool              `mapstructure:"remediation_planner_enabled"`
	RemediationMinSeverity            string            `mapstructure:"remediation_min_severity"`
	RemediationMinConfidence          float64           `mapstructure:"remediation_min_confidence"`
	RemediationUseAI                  bool              `mapstructure:"remediation_use_ai"`
	RemediationCommentOnIssue         bool              `mapstructure:"remediation_comment_on_issue"`
	RemediationPREnabled              bool              `mapstructure:"remediation_pr_enabled"`
	RemediationPRBranchPrefix         string            `mapstructure:"remediation_pr_branch_prefix"`
	RemediationPRRequireApproval      bool              `mapstructure:"remediation_pr_require_approval"`
	RemediationPRMaxFilesChanged      int               `mapstructure:"remediation_pr_max_files_changed"`
	RemediationPRMaxDiffLines         int               `mapstructure:"remediation_pr_max_diff_lines"`
	RemediationPRValidationTimeoutSeconds int           `mapstructure:"remediation_pr_validation_timeout_seconds"`
	EvidenceClosureEnabled            bool              `mapstructure:"evidence_closure_enabled"`
	EvidenceClosureCloseIssues        bool              `mapstructure:"evidence_closure_close_issues"`
	EvidenceClosureComment            bool              `mapstructure:"evidence_closure_comment"`
	EvidenceClosureRequireScannerSuccess bool           `mapstructure:"evidence_closure_require_scanner_success"`
}

func main() {
	// Always log to stdout so Docker/systemd capture output even when file logging fails.
	logger = logrus.New()
	logger.SetOutput(os.Stdout)
	logger.SetFormatter(&logrus.TextFormatter{
		FullTimestamp: true,
	})

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
	router.Use(gin.Logger(), gin.Recovery())
	setupRoutes(router)

	listenAddr := config.ListenHost + ":" + config.Port
	server := &http.Server{
		Addr:         listenAddr,
		Handler:      router,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
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
	viper.SetDefault("enable_llm_auditors", true)
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
	viper.SetDefault("scanner_timeout_seconds", 120)
	viper.SetDefault("min_issue_confidence", 0.5)
	viper.SetDefault("qdrant_enabled", false)
	viper.SetDefault("qdrant_url", "http://127.0.0.1:6333")
	viper.SetDefault("qdrant_collection", "bugbot-findings")
	viper.SetDefault("qdrant_vector_size", 1536)
	viper.SetDefault("qdrant_similarity_threshold", 0.85)
	viper.SetDefault("embedding_model", "text-embedding-3-small")
	viper.SetDefault("auto_create_issues", true)
	viper.SetDefault("max_issues_per_run", 50)
	viper.SetDefault("max_concurrent_analyses", 5)
	viper.SetDefault("analysis_timeout", 300)
	viper.SetDefault("rate_limit_per_minute", 60)
	viper.SetDefault("openwebui_model", "default")
	viper.SetDefault("ai_provider", "")
	viper.SetDefault("ai_model", "")
	viper.SetDefault("ai_insecure_skip_tls_verify", false)
	viper.SetDefault("skip_startup_checks", false)
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
	viper.SetDefault("database_path", "./data/bugbot.db")
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
	viper.SetDefault("label_compat_mode", "dual")
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
	viper.SetDefault("evidence_closure_enabled", true)
	viper.SetDefault("evidence_closure_close_issues", false)
	viper.SetDefault("evidence_closure_comment", true)
	viper.SetDefault("evidence_closure_require_scanner_success", true)

	// Environment variables — legacy BUGBOT_* plus REPOSITORY_DETECTIVE_* aliases.
	viper.AutomaticEnv()
	viper.SetEnvPrefix("BUGBOT")

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

	// Validate required fields
	if config.GiteaURL == "" {
		return fmt.Errorf("gitea_url is required")
	}
	if config.GiteaToken == "" {
		return fmt.Errorf("gitea_token is required")
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

	issues.SetLabelCompatMode(config.LabelCompatMode)

	return nil
}

func setupRoutes(router *gin.Engine) {
	router.Use(security.MiddlewareHeaders(), security.MiddlewareMaxBody(security.DefaultMaxRequestBody))
	// Set body size limit for multipart uploads
	router.MaxMultipartMemory = 8 << 20 // 8 MB max

	// Health check — no auth required
	router.GET("/health", func(c *gin.Context) {
		ready := componentsReady.Load()
		payload := healthPayload(ready)
		if !ready {
			c.JSON(http.StatusServiceUnavailable, payload)
			return
		}
		c.JSON(http.StatusOK, payload)
	})

	router.GET("/", func(c *gin.Context) {
		c.Redirect(http.StatusFound, "/onboard/")
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
		api.POST("/config/reload", handleConfigReload)
	}

	onboardAPI := router.Group("/api/v1/onboard")
	onboardAPI.Use(requireAPIKeyAuth())

	onboardingHandler = handlers.NewOnboardingHandler(logger, handlers.OnboardingConfig{
		GiteaURL:  config.GiteaURL,
		PublicURL: config.PublicURL,
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
			apiKey = c.GetHeader("X-Bugbot-API-Key")
		}
		if apiKey == "" {
			if auth := c.GetHeader("Authorization"); strings.HasPrefix(auth, "Bearer ") {
				apiKey = strings.TrimSpace(strings.TrimPrefix(auth, "Bearer "))
			}
		}
		if apiKey == "" {
			apiKey = c.Query("api_key")
		}

		if apiKey == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "API key required"})
			return
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

	if bugbotStore != nil {
		if err := bugbotStore.Close(); err != nil {
			logger.Warnf("Failed to close existing database: %v", err)
		}
		bugbotStore = nil
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
		bugbotStore = s
		scanRecorder = store.NewRecorder(s, logger)
		logger.Infof("Local database enabled (driver=%s path=%s)", config.DatabaseDriver, config.DatabasePath)
	} else {
		scanRecorder = store.NewRecorder(nil, logger)
		logger.Info("Local database disabled — running without persistence")
	}

	globalSnapshot := api.GlobalSnapshotFromConfig(api.GlobalConfigInput{
		ScanProfile:        config.ScanProfile,
		WorkspaceMode:      config.WorkspaceMode,
		AnalysisDepth:      config.AnalysisDepth,
		EnableLLMAuditors:  config.EnableLLMAuditors,
		EnableTrivy:        config.EnableTrivy,
		EnableGrype:        config.EnableGrype,
		EnableGitleaks:     config.EnableGitleaks,
		EnableSemgrep:      config.EnableSemgrep,
		EnableGovulncheck:  config.EnableGovulncheck,
		EnableGosec:        config.EnableGosec,
		EnableStaticcheck:  config.EnableStaticcheck,
		EnableHadolint:     config.EnableHadolint,
		EnableCheckov:      config.EnableCheckov,
		EnableLinters:      config.EnableLinters,
		GiteaStatusFailOn:  config.GiteaStatusFailOn,
		MinIssueConfidence: config.MinIssueConfidence,
		AutoCreateIssues:   config.AutoCreateIssues,
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
	initNotifyManager()
	controlPlaneHandler = api.NewHandler(bugbotStore, globalSnapshot, logger)
	if notifyManager != nil {
		controlPlaneHandler.SetNotificationGlobal(notifyManager.Config())
	}

	preinstallCfg := preinstall.Config{
		Enabled:              config.PreinstallAuditEnabled,
		AllowPrivateNetworks: config.PreinstallAllowPrivateNetworks,
		MaxRepoSizeMB:        config.PreinstallMaxRepoSizeMB,
		MaxFiles:             config.PreinstallMaxFiles,
		TimeoutSeconds:       config.PreinstallTimeoutSeconds,
		MaxFindings:          config.PreinstallMaxFindings,
		AllowGitClone:        config.PreinstallAllowGitClone,
		Health:               mainHealthConfig(),
		Graph:                mainGraphConfig(),
	}
	if bugbotStore != nil && config.PreinstallAuditEnabled {
		preinstallRunner = preinstall.NewRunner(bugbotStore, preinstallCfg, mainScannerConfig(), logger)
		preinstallRunner.SetAuditNotifier(preinstallNotifyBridge{})
		preinstallHandler = api.NewPreinstallHandler(bugbotStore, preinstallRunner, logger)
		logger.Info("Pre-install audit mode enabled")
	}

	if bugbotStore != nil && runnerCfg.DelegationEnabled && runnerCfg.Mode != runner.ModeCore && runnerCfg.SharedSecret != "" {
		runnerDispatcher = runner.NewDispatcher(bugbotStore, runnerCfg, logger)
		runnerReceiver = runner.NewReceiver(bugbotStore, runnerCfg, logger, ingestRunnerResult)
		runnerReceiver.SetJobsExpiredHandler(func(ctx context.Context, count int64) {
			notifyRunnerJobsExpired(ctx, count)
		})
		runnerHandler = api.NewRunnerHandler(bugbotStore, runnerCfg, runnerReceiver, logger)
		logger.Infof("Runner delegation enabled (mode=%s)", runnerCfg.Mode)
	}

	if config.UIEnabled {
		uiHandler, err := ui.NewHandler(bugbotStore, globalSnapshot, config.UIBasePath, logger, preinstallRunner, config.PreinstallAuditEnabled, config.APIKey)
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
		}
		if config.EvidenceClosureEnabled {
			uiHandler.SetClosureBackend(true, closureUIBridge{})
		}
		uiHandler.SetReadinessFn(func() operator.Readiness { return buildReadiness("running") })
		operatorUI = uiHandler
		logger.Infof("Operator UI enabled at %s", uiHandler.BasePath())
	} else {
		logger.Info("Operator UI disabled")
	}

	checkTimeout := time.Duration(config.StartupCheckTimeout) * time.Second
	ctx, cancel := context.WithTimeout(context.Background(), checkTimeout)
	defer cancel()

	// Initialize Gitea client
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

	// Initialize AI client (multi-provider) when LLM or Qdrant embeddings are required
	if config.needsAIProvider() {
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

		logger.Infof("Testing AI provider connection (timeout %s)...", checkTimeout)
		if err := aiClient.TestConnection(ctx); err != nil {
			if config.SkipStartupChecks {
				logger.Warnf("AI provider connection check failed (skipped): %v", err)
			} else {
				return fmt.Errorf("failed to connect to AI provider: %w", err)
			}
		} else {
			logger.Infof("AI provider connection established (%s, model=%s)", aiClient.Provider(), aiClient.Model())
		}
	} else {
		logger.Info("AI provider not required — deterministic-only mode (no LLM auditors, Qdrant disabled)")
		aiClient = nil
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
		Scanners: mainScannerConfig(),
		Health:   mainHealthConfig(),
		Graph:    mainGraphConfig(),
		Workspace: scanners.WorkspaceConfig{
			Mode:                   config.WorkspaceMode,
			MaxSizeMB:              config.WorkspaceMaxSizeMB,
			MaxFiles:               config.WorkspaceMaxFiles,
			ArchiveTimeoutSeconds:  config.WorkspaceArchiveTimeoutSeconds,
			DefaultAnalysisTimeout: config.AnalysisTimeout,
		},
	}
	analysisEngine = analyzers.NewEngine(giteaClient, aiClient, analysisConfig, logger)

	// Initialize semantic dedup (optional Qdrant)
	qdrantCfg := qdrant.Config{
		Enabled:             config.QdrantEnabled,
		URL:                 config.QdrantURL,
		APIKey:              config.QdrantAPIKey,
		Collection:          config.QdrantCollection,
		VectorSize:          config.QdrantVectorSize,
		SimilarityThreshold: config.QdrantSimilarityThreshold,
	}
	embedder := ai.NewEmbedder(ai.EmbedderConfig{
		BaseURL:    firstNonEmpty(config.EmbeddingBaseURL, config.AIBaseURL, config.OpenWebUIURL),
		APIKey:     firstNonEmpty(config.EmbeddingAPIKey, config.AIAPIKey, config.OpenWebUIToken),
		Model:      config.EmbeddingModel,
		Dimensions: config.QdrantVectorSize,
	})
	semanticStore := issues.NewSemanticStore(qdrant.NewStore(qdrantCfg), embedder, logger)
	if semanticStore.Enabled() {
		if err := semanticStore.Prepare(ctx); err != nil {
			logger.Warnf("Qdrant prepare failed (semantic dedup disabled): %v", err)
		} else {
			logger.Infof("Qdrant semantic dedup enabled (collection=%s, threshold=%.2f)",
				config.QdrantCollection, config.QdrantSimilarityThreshold)
		}
	}

	// Initialize issue manager
	issueConfig := &issues.Config{
		AutoCreateIssues:   config.AutoCreateIssues,
		GiteaBaseURL:       config.GiteaURL,
		IssueLabels:        issues.DefaultIssueBaseLabels(),
		MaxIssuesPerRun:    config.MaxIssuesPerRun,
		SkipLowSeverity:    config.SkipLowSeverity,
		GroupSimilarIssues: config.GroupSimilarIssues,
		MinIssueConfidence: config.MinIssueConfidence,
		IssueTitleTemplate: "[{{severity}}] {{title}}",
		IssueBodyTemplate:  "",
	}
	issueManager = issues.NewManager(giteaClient, issueConfig, logger, semanticStore)

	webhookHandler = handlers.NewWebhookHandler(logger, &handlers.Config{
		WebhookSecret:         config.WebhookSecret,
		AllowInsecureWebhooks: config.AllowInsecureWebhooks,
		IncludePatterns:       config.RepositoryIncludePatterns,
		ExcludePatterns:       config.RepositoryExcludePatterns,
	}, &webhookProcessor{})

	analysisLimiter = limiter.New(config.MaxConcurrentAnalyses)
	logger.Infof("Analysis concurrency limit: %d", config.MaxConcurrentAnalyses)

	if config.SchedulerEnabled && bugbotStore != nil {
		schedulerCtx, schedulerCancel = context.WithCancel(context.Background())
		scanScheduler = orch.NewScheduler(
			bugbotStore,
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

	if bugbotStore != nil {
		staleAge := time.Duration(config.AnalysisTimeout) * time.Second * 2
		if staleAge < 30*time.Minute {
			staleAge = 30 * time.Minute
		}
		if n, err := bugbotStore.ReapStaleScans(context.Background(), staleAge); err != nil {
			logger.Warnf("Failed to reap stale scans: %v", err)
		} else if n > 0 {
			logger.Infof("Reaped %d stale started scan(s)", n)
		}
	}

	logger.Info("All components initialized successfully")
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
			TriggerType:   store.TriggerPush,
			Ref:           payload.Ref,
			CommitSHA:     commitSHA,
			CloneURL:      payload.Repository.CloneURL,
			ConnectedRepo: true,
		}
		analysisCtx, repositoryID := beginPersistedScan(analysisCtx, &scanCtx)
		analysisCtx, effective := resolveEffectiveSettingsForRepo(analysisCtx, owner, repo)
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
			severitiesForStatus(result, effective),
			scannerSummaries(result),
			false,
			effective.PolicyLevel,
			effective.SeverityGate,
		)
		notifyPRGateFailed(postCtx, repositoryID, owner, repo, scanCtx.ScanID, eval)
		createIssuesFromResult(postCtx, owner, repo, result, fmt.Sprintf("Push to %s", payload.Ref), ref, 0, repositoryID, effective)
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
			TriggerType:   store.TriggerPR,
			Ref:           payload.PullRequest.Head.Ref,
			CommitSHA:     commitSHA,
			PRNumber:      prNumber,
			CloneURL:      payload.Repository.CloneURL,
			ConnectedRepo: true,
		}
		analysisCtx, repositoryID := beginPersistedScan(analysisCtx, &scanCtx)
		analysisCtx, effective := resolveEffectiveSettingsForRepo(analysisCtx, owner, repo)
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
			severitiesForStatus(result, effective),
			scannerSummaries(result),
			false,
			effective.PolicyLevel,
			effective.SeverityGate,
		)
		notifyPRGateFailed(postCtx, repositoryID, owner, repo, scanCtx.ScanID, eval)
		createIssuesFromResult(postCtx, owner, repo, result, fmt.Sprintf("Pull Request #%d", prNumber), "", prNumber, repositoryID, effective)
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
	if result == nil {
		return nil
	}
	summaries := make([]gitea.ScannerResultSummary, 0, len(result.ScannerResults))
	for _, scannerResult := range result.ScannerResults {
		summaries = append(summaries, gitea.ScannerResultSummary{
			Scanner: scannerResult.Scanner,
			Status:  string(scannerResult.Status),
		})
	}
	return summaries
}

func runAnalysis(_ context.Context, fn func(context.Context)) {
	// Wait for a concurrency slot without a scan timeout — queue wait must not consume analysis time.
	if err := analysisLimiter.Run(context.Background(), func() {
		analysisCtx, cancel := context.WithTimeout(context.Background(), time.Duration(config.AnalysisTimeout)*time.Second)
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

		scanCtx := store.ScanContext{
			Owner:         repo.Owner,
			Repo:          repo.Name,
			ForgeType:     repo.ForgeType,
			CloneURL:      repo.CloneURL,
			DefaultBranch: repo.DefaultBranch,
			TriggerType:   store.TriggerScheduled,
			Ref:           ref,
			ConnectedRepo: true,
		}
		scanCtxInner, repositoryID := beginPersistedScan(analysisCtx, &scanCtx)
		scanCtxInner, effective := resolveEffectiveSettingsForRepo(scanCtxInner, repo.Owner, repo.Name)
		if !effective.Enabled {
			logger.Infof("Scheduled scan skipped — repository %s disabled in settings", repo.FullName)
			return
		}

		if bugbotStore != nil && repositoryID > 0 {
			if dbRepo, rerr := bugbotStore.GetRepository(scanCtxInner, repositoryID); rerr == nil {
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

		result, err := analysisEngine.AnalyzeRepository(scanCtxInner, repo.Owner, repo.Name, ref)
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

		createIssuesFromResult(postCtx, repo.Owner, repo.Name, result,
			fmt.Sprintf("Scheduled scan (%s)", repo.ScheduleCron), ref, 0, repositoryID, effective)
	}); err != nil {
		return fmt.Errorf("scheduled scan skipped: %w", err)
	}
	return runErr
}

func scanPolicyFromEffective(e store.EffectiveSettings) analyzers.ScanPolicy {
	return analyzers.ScanPolicy{
		Enabled:           e.Enabled,
		PolicyLevel:       e.PolicyLevel,
		WorkspaceMode:     e.WorkspaceMode,
		AnalysisDepth:     e.AnalysisDepth,
		EnableLLMAuditors: e.EnableLLMAuditors,
		EnableTrivy:       e.EnableTrivy,
		EnableGrype:       e.EnableGrype,
		EnableGitleaks:    e.EnableGitleaks,
		EnableSemgrep:     e.EnableSemgrep,
		EnableGovulncheck: e.EnableGovulncheck,
		EnableGosec:       e.EnableGosec,
		EnableStaticcheck: e.EnableStaticcheck,
		EnableHadolint:    e.EnableHadolint,
		EnableCheckov:     e.EnableCheckov,
		EnableLinters:     e.EnableLinters,
		SeverityGate:      e.SeverityGate,
		ConfidenceGate:    e.ConfidenceGate,
		IssuePolicy:       e.IssuePolicy,
		RemediationPolicy: e.RemediationPolicy,
		AIPolicy:          e.AIPolicy,
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

func resolveEffectiveSettingsForRepo(ctx context.Context, owner, repo string) (context.Context, store.EffectiveSettings) {
	repoSettings := store.RepoSettings{}
	effective, meta := store.ResolveEffectiveSettingsFull(appGlobalSnapshot, repoSettings)
	if bugbotStore != nil {
		fullName := owner + "/" + repo
		dbRepo, err := bugbotStore.GetRepositoryByFullName(ctx, store.ForgeTypeGitea, fullName)
		if err == nil {
			settings, serr := bugbotStore.GetRepoSettings(ctx, dbRepo.ID)
			if serr == nil {
				repoSettings = settings
				effective, meta = store.ResolveEffectiveSettingsFull(appGlobalSnapshot, settings)
			}
		}
	}
	if override := scanProfileOverrideFromContext(ctx); override != "" && override != store.ScanProfileCustom {
		effective = store.MergeConfigOverProfile(store.ProfileDefaults(override), effective)
		meta.ScanProfile = override
		meta.ProfileSource = "request_override"
	}
	if effective.AIPolicy == store.AIPolicyAllowed && aiClient == nil {
		effective.EnableLLMAuditors = false
	}
	policy := scanPolicyFromEffective(effective)
	policy.ScanProfile = meta.ScanProfile
	policy.ProfileModified = meta.ProfileModified
	policy.ProfileSource = meta.ProfileSource
	return analyzers.WithScanPolicy(ctx, policy), effective
}

func filterIssuesForForge(issues []ai.CodeIssue, effective store.EffectiveSettings) []ai.CodeIssue {
	if !store.ShouldCreateForgeIssues(effective) {
		return nil
	}
	out := make([]ai.CodeIssue, 0, len(issues))
	for _, issue := range issues {
		if store.PassesIssueGates(issue.Severity, issue.Confidence, effective) {
			out = append(out, issue)
		}
	}
	return out
}

func severitiesForStatus(result *analyzers.AnalysisResult, effective store.EffectiveSettings) []string {
	if result == nil {
		return nil
	}
	severities := make([]string, len(result.Issues))
	confidences := make([]float64, len(result.Issues))
	for i, issue := range result.Issues {
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
				Scanner:       sr.Scanner,
				Status:        string(sr.Status),
				FindingsCount: len(sr.Findings),
				Detail:        sr.Detail,
			})
		}
		data = &store.ScanCompletion{
			IssuesFound:       len(result.Issues),
			FilesAnalyzed:     result.FilesAnalyzed,
			AnalysisTime:      result.AnalysisTime,
			OverallScore:      result.OverallScore,
			CommitSHA:         result.CommitSHA,
			WorkspaceModeUsed: result.WorkspaceModeUsed,
			PolicySnapshot:    result.PolicySnapshot,
			ScannerResults:    scanners,
		}
		if result.Graph != nil {
			if raw, err := json.Marshal(result.Graph); err == nil {
				data.GraphJSON = raw
				data.GraphNodeCount = result.Graph.Metrics.NodeCount
				data.GraphEdgeCount = result.Graph.Metrics.EdgeCount
			}
		}
	}
	if err := scanRecorder.FinishScan(ctx, scanID, data, analysisErr); err != nil {
		logger.Warnf("Failed to finish scan persistence: %v", err)
	}
	notifyScanFinish(ctx, scanCtx, repositoryID, result, analysisErr)
}

func postAnalysisContext(parent context.Context) (context.Context, context.CancelFunc) {
	if parent == nil {
		parent = context.Background()
	}
	return context.WithTimeout(context.WithoutCancel(parent), 15*time.Minute)
}

func createIssuesFromResult(ctx context.Context, owner, repo string, result *analyzers.AnalysisResult, contextLabel, commit string, prNumber int, repositoryID int64, effective store.EffectiveSettings) {
	if result == nil {
		return
	}

	postCtx, cancel := postAnalysisContext(ctx)
	defer cancel()

	repository := fmt.Sprintf("%s/%s", owner, repo)
	issues.EnrichIssues(repository, result.ScanID, result.Issues)

	commitRef := commit
	if commitRef == "" {
		commitRef = result.CommitSHA
	}

	var processed []issues.ProcessedIssueRecord

	if store.ShouldCreateForgeIssues(effective) {
		forgeIssues := filterIssuesForForge(result.Issues, effective)
		if len(forgeIssues) > 0 {
			issueReq := &issues.IssueCreationRequest{
				Owner:      owner,
				Repository: repo,
				AnalysisResult: &ai.CodeAnalysisResult{
					Issues:       forgeIssues,
					OverallScore: result.OverallScore,
					AnalysisTime: result.AnalysisTime,
				},
				Context:            contextLabel,
				Commit:             commitRef,
				PullRequest:        prNumber,
				ScanID:             result.ScanID,
				UseSemanticDedup:     store.UseSemanticDedup(effective),
				MinIssueConfidence:   effective.ConfidenceGate,
				ForceIssueCreation:   true,
			}

			issueResult, err := issueManager.CreateIssuesFromAnalysis(postCtx, issueReq)
			if err != nil {
				logger.Errorf("Failed to create issues: %v", err)
			} else {
				logger.Infof("Created %d issues, updated %d, skipped %d", issueResult.IssuesCreated, issueResult.IssuesUpdated, issueResult.IssuesSkipped)
				processed = issueResult.ProcessedIssues
			}
		}
	} else {
		logger.Infof("Forge issue creation skipped (policy_level=%s issue_policy=%s)", effective.PolicyLevel, effective.IssuePolicy)
	}

	if scanRecorder != nil && scanRecorder.Enabled() && repositoryID > 0 && result.ScanID != "" {
		if err := scanRecorder.RecordIssues(postCtx, repositoryID, result.ScanID, result.Issues, processed); err != nil {
			logger.Warnf("Failed to persist findings: %v", err)
		} else {
			maybeGenerateRemediationPlans(postCtx, repositoryID, result.Issues, processed)
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
	if bugbotStore != nil {
		settings, serr := bugbotStore.GetRepoSettings(ctx, repo.ID)
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
	ctx, effective := resolveEffectiveSettingsForRepo(ctx, repo.Owner, repo.Name)

	var policy analyzers.PolicySnapshot
	_ = json.Unmarshal(job.PolicySnapshotJSON, &policy)

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
	createIssuesFromResult(ctx, repo.Owner, repo.Name, analysisResult,
		fmt.Sprintf("Runner scan (%s)", job.JobType), job.Ref, job.PRNumber, repo.ID, effective)
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
	Owner       string `json:"owner" binding:"required"`
	Repository  string `json:"repository" binding:"required"`
	Ref         string `json:"ref"`
	Type        string `json:"type"` // "repository" or "pull_request"
	PRNumber    int    `json:"pr_number"`
	ScanProfile string `json:"scan_profile"`
}

func enqueueManualAnalysis(parentCtx context.Context, req manualAnalysisRequest) {
	// Detach from HTTP request context so bulk /analyze/all scans are not cancelled when the handler returns.
	scanCtx := context.WithoutCancel(parentCtx)
	go func() {
		runAnalysis(withScanProfileOverride(scanCtx, req.ScanProfile), func(ctx context.Context) {
			var result *analyzers.AnalysisResult
			var err error
			var repositoryID int64

			scanCtx := store.ScanContext{
				Owner:         req.Owner,
				Repo:          req.Repository,
				TriggerType:   store.TriggerManual,
				Ref:           req.Ref,
				ConnectedRepo: true,
			}
			ctx, repositoryID = beginPersistedScan(ctx, &scanCtx)
			ctx, effective := resolveEffectiveSettingsForRepo(ctx, req.Owner, req.Repository)
			if !effective.Enabled {
				postCtx, postCancel := postAnalysisContext(ctx)
				defer postCancel()
				finishPersistedScan(postCtx, &scanCtx, repositoryID, nil, fmt.Errorf("repository disabled in settings"))
				logger.Infof("Manual scan skipped — repository %s/%s disabled in settings", req.Owner, req.Repository)
				return
			}

			if req.Type != "pull_request" || req.PRNumber <= 0 {
				if bugbotStore != nil && repositoryID > 0 {
					if dbRepo, rerr := bugbotStore.GetRepository(ctx, repositoryID); rerr == nil {
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
				scanCtx.PRNumber = req.PRNumber
				scanCtx.TriggerType = store.TriggerPR
				result, err = analysisEngine.AnalyzePullRequest(ctx, req.Owner, req.Repository, req.PRNumber)
			} else {
				ref := strings.TrimSpace(req.Ref)
				if ref == "" {
					ref = "main"
				}
				if giteaClient != nil {
					if resolved, rerr := giteaClient.ResolveRef(ctx, req.Owner, req.Repository, ref); rerr == nil {
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
				createIssuesFromResult(postCtx, req.Owner, req.Repository, result,
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

	enqueueManualAnalysis(c.Request.Context(), req)
	c.JSON(http.StatusOK, gin.H{"status": "analysis started"})
}

// handleBulkAnalysis queues a full-repository scan for every Gitea repo the token can see.
func handleBulkAnalysis(c *gin.Context) {
	var req struct {
		Orgs        []string `json:"orgs"`
		Ref         string   `json:"ref"`
		ScanProfile string   `json:"scan_profile"`
		DryRun      bool     `json:"dry_run"`
	}
	_ = c.ShouldBindJSON(&req)

	if giteaClient == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Gitea client not configured"})
		return
	}

	listCtx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Minute)
	defer cancel()

	reposByID := make(map[int64]gitea.RepositorySummary)
	userRepos, err := giteaClient.ListAllUserRepositories(listCtx)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": fmt.Sprintf("list user repositories: %v", err)})
		return
	}
	for _, repo := range userRepos {
		reposByID[repo.ID] = repo
	}

	orgs := req.Orgs
	if len(orgs) == 0 {
		if envOrgs := strings.TrimSpace(os.Getenv("GITEA_SCAN_ORGS")); envOrgs != "" {
			for _, org := range strings.Split(envOrgs, ",") {
				if o := strings.TrimSpace(org); o != "" {
					orgs = append(orgs, o)
				}
			}
		}
	}
	for _, org := range orgs {
		orgRepos, err := giteaClient.ListAllOrgRepositories(listCtx, org)
		if err != nil {
			c.JSON(http.StatusBadGateway, gin.H{"error": fmt.Sprintf("list org %s repositories: %v", org, err)})
			return
		}
		for _, repo := range orgRepos {
			reposByID[repo.ID] = repo
		}
	}

	var queued, skipped []string
	defaultRef := strings.TrimSpace(req.Ref)
	for _, repo := range reposByID {
		fullName := strings.TrimSpace(repo.FullName)
		if fullName == "" {
			continue
		}
		if !handlers.RepoAllowed(fullName, config.RepositoryIncludePatterns, config.RepositoryExcludePatterns) {
			skipped = append(skipped, fullName)
			continue
		}
		parts := strings.SplitN(fullName, "/", 2)
		if len(parts) != 2 {
			skipped = append(skipped, fullName)
			continue
		}
		ref := defaultRef
		if ref == "" {
			ref = strings.TrimSpace(repo.DefaultBranch)
		}
		if ref == "" {
			ref = "main"
		}
		queued = append(queued, fullName)
		if req.DryRun {
			continue
		}
		enqueueManualAnalysis(c.Request.Context(), manualAnalysisRequest{
			Owner:       parts[0],
			Repository:  parts[1],
			Ref:         ref,
			Type:        "repository",
			ScanProfile: strings.TrimSpace(req.ScanProfile),
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"status":        "bulk analysis queued",
		"queued_count":  len(queued),
		"skipped_count": len(skipped),
		"scan_profile":  strings.TrimSpace(req.ScanProfile),
		"queued":        queued,
		"skipped":       skipped,
		"dry_run":       req.DryRun,
	})
}

// handleStatus handles status requests
func handleStatus(c *gin.Context) {
	r := buildReadiness("running")
	c.JSON(http.StatusOK, r)
}

func handleAbout(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"product_name": "Repository Detective",
		"legacy_name":  "Bugbot",
		"tagline":      "Inspect. Analyze. Improve.",
		"version":      version,
		"documentation_index": "/docs/README.md",
		"compatibility": gin.H{
			"bugbot_env":          true,
			"bugbot_labels":       true,
			"bugbot_fingerprints": true,
			"label_compat_mode":   issues.LabelCompatMode(),
		},
		"safe_loop": "detect → issue → plan → approve → patch PR → merge → rescan → verified closure",
	})
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
	if c.QdrantEnabled {
		return true
	}
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
	return scanners.Config{
		EnableTrivy:              config.EnableTrivy,
		EnableGrype:              config.EnableGrype,
		EnableGitleaks:           config.EnableGitleaks,
		EnableSemgrep:            config.EnableSemgrep,
		EnableGovulncheck:        config.EnableGovulncheck,
		EnableGosec:              config.EnableGosec,
		EnableStaticcheck:        config.EnableStaticcheck,
		EnableHadolint:           config.EnableHadolint,
		EnableCheckov:            config.EnableCheckov,
		EnableLinters:            config.EnableLinters,
		GitleaksConfig:           config.GitleaksConfig,
		GitleaksTimeoutSeconds:   config.GitleaksTimeoutSeconds,
		SemgrepConfig:            config.SemgrepConfig,
		SemgrepTimeoutSeconds:    config.SemgrepTimeoutSeconds,
		SemgrepMaxFindings:       config.SemgrepMaxFindings,
		SemgrepSeverityThreshold: config.SemgrepSeverityThreshold,
		GovulncheckTimeoutSeconds: config.GovulncheckTimeoutSeconds,
		GosecTimeoutSeconds:       config.GosecTimeoutSeconds,
		StaticcheckTimeoutSeconds: config.StaticcheckTimeoutSeconds,
		GoScannerMaxFindings:      config.GoScannerMaxFindings,
		HadolintTimeoutSeconds:    config.HadolintTimeoutSeconds,
		CheckovTimeoutSeconds:     config.CheckovTimeoutSeconds,
		IACScannerMaxFindings:     config.IACScannerMaxFindings,
		TrivySeverity:            "HIGH,CRITICAL",
		GrypeFailOn:              "high",
		LinterMinSeverity:        "warning",
		TimeoutSeconds:           config.ScannerTimeoutSeconds,
	}
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
		api.NewRemediationHandler(bugbotStore, remediationBridge{}, config.RemediationPREnabled).RegisterRoutes(cp)
	}
	if config.EvidenceClosureEnabled {
		api.NewClosureHandler(bugbotStore, closureBridge{}).RegisterRoutes(cp)
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
	uiGroup.Use(requireComponentsReady(), requireAPIKeyAuth())
	operatorUI.RegisterRoutes(uiGroup)
	logger.Infof("Control plane API and UI routes registered")
}
