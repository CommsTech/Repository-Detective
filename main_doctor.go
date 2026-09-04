package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"git.commsnet.org/commstech/repository-detective/doctor"
	"git.commsnet.org/commstech/repository-detective/gitea"
	"git.commsnet.org/commstech/repository-detective/internal/privacy"
	"git.commsnet.org/commstech/repository-detective/operator"
	"git.commsnet.org/commstech/repository-detective/store"
	"github.com/gin-gonic/gin"
)

// maybeRunDoctorCLI exits the process when argv is `doctor` (RD-014).
func maybeRunDoctorCLI() {
	if len(os.Args) < 2 || os.Args[1] != "doctor" {
		return
	}
	jsonOut := false
	bundleOut := false
	owner, repo := "", ""
	for i := 2; i < len(os.Args); i++ {
		switch os.Args[i] {
		case "--json", "-json":
			jsonOut = true
		case "--bundle":
			bundleOut = true
		case "--repo":
			if i+1 < len(os.Args) {
				i++
				parts := strings.SplitN(os.Args[i], "/", 2)
				if len(parts) == 2 {
					owner, repo = parts[0], parts[1]
				}
			}
		}
	}
	if err := loadConfig(); err != nil {
		fmt.Fprintf(os.Stderr, "config: %v\n", err)
		os.Exit(2)
	}
	report := runDoctorReport(context.Background(), owner, repo)
	if bundleOut {
		bundle := doctor.BuildSupportBundle(report, sanitizedDoctorConfig(), nil)
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		_ = enc.Encode(bundle)
	} else if jsonOut {
		_ = doctor.FormatJSON(os.Stdout, report)
	} else {
		_ = doctor.FormatHuman(os.Stdout, report)
	}
	switch report.Overall {
	case doctor.OverallHealthy:
		os.Exit(0)
	case doctor.OverallDegraded:
		os.Exit(0)
	default:
		os.Exit(1)
	}
}

func sanitizedDoctorConfig() map[string]string {
	if config == nil {
		return map[string]string{}
	}
	return map[string]string{
		"gitea_url":                   config.GiteaURL,
		"public_url":                  config.PublicURL,
		"privacy_mode":                config.PrivacyMode,
		"auth_mode":                   config.AuthMode,
		"scan_profile":                config.ScanProfile,
		"enable_llm_auditors":         fmt.Sprintf("%v", config.EnableLLMAuditors),
		"remediation_pr_enabled":      fmt.Sprintf("%v", config.RemediationPREnabled),
		"reject_query_string_api_key": fmt.Sprintf("%v", config.RejectQueryStringAPIKey),
		"gitea_token":                 config.GiteaToken, // redacted by bundle builder
		"api_key":                     config.APIKey,
		"webhook_secret":              config.WebhookSecret,
	}
}

func runDoctorReport(ctx context.Context, owner, repo string) doctor.Report {
	in := doctor.Input{
		Version:                   version,
		Commit:                    commit,
		Edition:                   "community",
		ConfigValid:               true,
		AuthMode:                  config.AuthMode,
		SessionSecretConfigured:   strings.TrimSpace(config.SessionSecret) != "",
		RejectQueryStringAPIKey:   config.RejectQueryStringAPIKey,
		WarnQueryStringAPIKey:     config.WarnQueryStringAPIKey,
		APIKeyConfigured:          strings.TrimSpace(config.APIKey) != "",
		PrivacyMode:               config.PrivacyMode,
		AIEnabled:                 config.EnableLLMAuditors || openClawInvokable(),
		AIProvider:                config.effectiveAIProvider(),
		AIBaseURL:                 firstNonEmpty(config.AIBaseURL, config.OpenWebUIURL),
		AIModel:                   firstNonEmpty(config.AIModel, config.OpenWebUIModel),
		ForgeURL:                  config.GiteaURL,
		ForgeTokenSet:             strings.TrimSpace(config.GiteaToken) != "",
		PublicURL:                 config.PublicURL,
		WebhookSecretSet:          strings.TrimSpace(config.WebhookSecret) != "",
		ScanProfile:               config.ScanProfile,
		DatabaseEnabled:           config.DatabaseEnabled,
		SchemaVersion:             25,
		RemediationPlannerEnabled: config.RemediationPlannerEnabled,
		RemediationPREnabled:      config.RemediationPREnabled,
		RemediationUseRunner:      config.RemediationPRUseRunnerVerification,
		ClassBIsolation:           "NOT_PROVEN",
		ClassBExecutionAllowed:    config.RemediationPREnabled,
		RunnerDelegationEnabled:   config.RunnerDelegationEnabled,
	}
	in.PolicyMode = "Warn"
	in.EnforcementMode = store.EnforcementWarn
	ws := filepath.Join(filepath.Dir(firstNonEmpty(config.DatabasePath, "data/repository-detective.db")), "tmp")
	if ws == "tmp" || strings.HasPrefix(ws, "/tmp") {
		ws = "data/tmp"
	}
	in.WorkspaceDir = ws

	d := privacy.EvaluateAIEgress(config.PrivacyMode, in.AIProvider, in.AIBaseURL, config.EnableLLMAuditors)
	in.AILocality = d.EndpointClass
	in.AIEgressAllowed = d.Allowed
	in.AIEgressReason = d.Reason
	if in.ForgeURL != "" {
		if c, _, err := privacy.ClassifyURL(in.ForgeURL); err == nil {
			in.ForgeLocality = c
		}
	}

	if config.DatabaseEnabled && rdStore != nil {
		in.DatabaseOK = true
		if ev, ok, _ := rdStore.GetWebhookDeliveryEvidence(ctx); ok {
			in.WebhookDeliveryProven = true
			in.WebhookLastDelivery = ev.ReceivedAt + " event=" + ev.EventKind + " repo=" + ev.Repository
		}
		if fs, ok, _ := rdStore.GetFirstScanEvidence(ctx); ok {
			in.FirstScanProven = true
			in.FirstScanDetail = fmt.Sprintf("scan_id=%s trigger=%s via_webhook=%v required=%d/%d",
				fs.ScanID, fs.TriggerType, fs.ViaWebhook, fs.RequiredOK, fs.RequiredTotal)
			in.WebhookScanProven = fs.ViaWebhook
		}
	} else if config.DatabaseEnabled {
		if s, err := store.Open(store.Config{Enabled: true, Path: config.DatabasePath}); err == nil {
			in.DatabaseOK = true
			_ = s.Close()
		} else {
			in.DatabaseOK = false
			in.DatabaseDetail = err.Error()
		}
	}
	if err := os.MkdirAll(in.WorkspaceDir, 0o750); err == nil {
		probe := filepath.Join(in.WorkspaceDir, ".rd-doctor-write-probe")
		if werr := os.WriteFile(probe, []byte("ok"), 0o600); werr == nil {
			in.WorkspaceOK = true
			_ = os.Remove(probe)
		} else {
			in.WorkspaceDetail = werr.Error()
		}
	} else {
		in.WorkspaceDetail = err.Error()
	}

	profile := store.NormalizeScanProfile(config.ScanProfile)
	e := effectiveFromMainConfig()
	in.RequiredScanners = store.RequiredScannersForProfile(profile, e)
	tools := operator.CheckTools(scannerConfigFromMain())
	roleRequired := map[string]struct{}{}
	for _, n := range in.RequiredScanners {
		roleRequired[n] = struct{}{}
	}
	for _, t := range tools {
		role := store.ScannerRoleInformational
		if _, ok := roleRequired[strings.ToLower(t.Name)]; ok {
			role = store.ScannerRoleRequired
		} else if t.EnabledInConfig {
			role = store.ScannerRoleOptional
		}
		in.ScannerTools = append(in.ScannerTools, doctor.ScannerToolInput{
			Name: t.Name, Role: role, EnabledInConfig: t.EnabledInConfig,
			Available: t.Available, Version: t.Version, StatusState: t.StatusState,
		})
	}

	// Notification channels locality
	mode := privacy.NormalizeMode(config.PrivacyMode)
	addCh := func(name, raw string, enabled bool) {
		if !enabled && raw == "" {
			return
		}
		ch := doctor.NotificationChannel{Name: name, URL: raw, Enabled: enabled}
		if raw != "" {
			dec := privacy.EvaluateURLEgress(mode, raw)
			ch.Locality = dec.EndpointClass
			ch.Allowed = dec.Allowed
		} else if name == "telegram" && enabled {
			ch.Locality = privacy.ClassExternal
			ch.Allowed = mode != privacy.ModeLocalOnly
		}
		in.NotificationChannels = append(in.NotificationChannels, ch)
	}
	addCh("slack", config.SlackWebhookURL, config.SlackEnabled)
	addCh("discord", config.DiscordWebhookURL, config.DiscordEnabled)
	addCh("webhook", config.WebhookNotificationURL, config.WebhookNotificationsEnabled)
	addCh("telegram", "", config.TelegramEnabled)

	if giteaClient != nil || (config.GiteaURL != "" && config.GiteaToken != "") {
		client := giteaClient
		if client == nil {
			client = gitea.NewClient(config.GiteaURL, config.GiteaToken, logger)
		}
		probeCtx, cancel := context.WithTimeout(ctx, 20*time.Second)
		defer cancel()
		if err := client.TestConnection(probeCtx); err != nil {
			in.ForgeReachable = false
			in.ForgeAuthOK = false
			in.ForgeReachDetail = err.Error()
			in.ForgeAuthDetail = err.Error()
		} else {
			in.ForgeReachable = true
			in.ForgeAuthOK = true
		}
		if owner != "" && repo != "" {
			in.RepoOwner, in.RepoName = owner, repo
			if rp, err := client.GetRepositoryPermissions(probeCtx, owner, repo); err == nil {
				m := gitea.BuildPermissionMatrix(rp.Permissions)
				in.RepoPerms = &doctor.RepoPermissionResult{
					RepositoryRead: m.RepositoryRead, IssuesWrite: m.IssuesWrite,
					CommitStatusWrite: m.CommitStatusWrite, PRCommentWrite: m.PRCommentWrite,
					BranchPRRemediation: m.BranchPRRemediation, Detail: m.Detail,
				}
			} else {
				in.RepoPerms = &doctor.RepoPermissionResult{
					RepositoryRead: "ERROR", IssuesWrite: "ERROR", CommitStatusWrite: "ERROR",
					PRCommentWrite: "ERROR", BranchPRRemediation: "ERROR", Detail: err.Error(),
				}
			}
			wantHook := strings.TrimSuffix(config.PublicURL, "/") + "/webhook"
			if hooks, err := client.ListRepositoryHooks(probeCtx, owner, repo); err == nil {
				if _, ok := gitea.FindHookByURL(hooks, wantHook); ok {
					in.WebhookRegistered = true
					in.WebhookRegistrationDetail = "matching hook URL found on repository"
				}
			}
		}
	}

	return doctor.Run(in)
}

func effectiveFromMainConfig() store.EffectiveSettings {
	g := store.DefaultGlobalSettings()
	g.ScanProfile = store.NormalizeScanProfile(config.ScanProfile)
	g.EnableTrivy = config.EnableTrivy
	g.EnableGrype = config.EnableGrype
	g.EnableGitleaks = config.EnableGitleaks
	g.EnableSemgrep = config.EnableSemgrep
	g.EnableGovulncheck = config.EnableGovulncheck
	g.EnableGosec = config.EnableGosec
	g.EnableStaticcheck = config.EnableStaticcheck
	g.EnableHadolint = config.EnableHadolint
	g.EnableCheckov = config.EnableCheckov
	g.EnableLinters = config.EnableLinters
	g.EnableLLMAuditors = config.EnableLLMAuditors
	return store.ResolveRepoSettings(g, store.RepoSettings{})
}

func scannerConfigFromMain() operator.ScannerConfig {
	return operator.ScannerConfig{
		EnableTrivy: config.EnableTrivy, EnableGrype: config.EnableGrype,
		EnableGitleaks: config.EnableGitleaks, EnableSemgrep: config.EnableSemgrep,
		EnableGovulncheck: config.EnableGovulncheck, EnableGosec: config.EnableGosec,
		EnableStaticcheck: config.EnableStaticcheck, EnableHadolint: config.EnableHadolint,
		EnableCheckov: config.EnableCheckov, EnableLinters: config.EnableLinters,
		RemediationGit: config.RemediationPREnabled,
	}
}

func openClawInvokable() bool {
	if config == nil {
		return false
	}
	cfg := config.OpenClawAIReview.Normalized()
	return cfg.Enabled && cfg.CanInvoke()
}

func registerDoctorAPI(r gin.IRoutes) {
	r.GET("/doctor", func(c *gin.Context) {
		owner := c.Query("owner")
		repo := c.Query("repo")
		report := runDoctorReport(c.Request.Context(), owner, repo)
		c.JSON(http.StatusOK, doctor.RedactReport(report))
	})
	r.GET("/doctor/bundle", func(c *gin.Context) {
		report := runDoctorReport(c.Request.Context(), c.Query("owner"), c.Query("repo"))
		c.JSON(http.StatusOK, doctor.BuildSupportBundle(report, sanitizedDoctorConfig(), nil))
	})
}
