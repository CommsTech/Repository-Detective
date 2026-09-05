package handlers

import (
	"context"
	"net/url"
	"strings"
	"time"

	"git.commsnet.org/commstech/repository-detective/doctor"
	"git.commsnet.org/commstech/repository-detective/gitea"
	"git.commsnet.org/commstech/repository-detective/internal/privacy"
	"git.commsnet.org/commstech/repository-detective/store"
	"github.com/gin-gonic/gin"
)

func (h *OnboardingHandler) handleVerifyPermissions(c *gin.Context) {
	var req onboardConnectionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": "Invalid request"})
		return
	}
	if req.GiteaURL == "" || req.GiteaToken == "" {
		c.JSON(400, gin.H{"error": "Gitea URL and token are required"})
		return
	}
	client := gitea.NewClient(firstNonEmptyStr(req.GiteaURL, h.giteaURL), req.GiteaToken, h.logger)
	ctx, cancel := context.WithTimeout(c.Request.Context(), 20*time.Second)
	defer cancel()

	if err := client.TestConnection(ctx); err != nil {
		c.JSON(502, gin.H{"error": "forge connection failed", "detail": err.Error()})
		return
	}

	forgeClass := privacy.ClassUnknown
	if cls, _, err := privacy.ClassifyURL(req.GiteaURL); err == nil {
		forgeClass = cls
	}

	perms := gin.H{
		"forge_locality":        forgeClass,
		"repository_read":       "PASS",
		"issues_write":          "NOT_PROBED",
		"commit_status_write":   "NOT_PROBED",
		"pr_comment_write":      "NOT_PROBED",
		"branch_pr_remediation": "NOT_PROBED",
		"note":                  "Select a repository for a full permission matrix. Remediation permission is optional for deterministic scanning.",
	}

	full := strings.TrimSpace("")
	if len(req.Repositories) > 0 {
		full = req.Repositories[0]
	}
	if full != "" {
		parts := strings.SplitN(full, "/", 2)
		if len(parts) == 2 {
			rp, err := client.GetRepositoryPermissions(ctx, parts[0], parts[1])
			if err != nil {
				perms["repository_read"] = "ERROR"
				perms["detail"] = err.Error()
			} else {
				m := gitea.BuildPermissionMatrix(rp.Permissions)
				perms["repository_read"] = m.RepositoryRead
				perms["issues_write"] = m.IssuesWrite
				perms["commit_status_write"] = m.CommitStatusWrite
				perms["pr_comment_write"] = m.PRCommentWrite
				perms["branch_pr_remediation"] = m.BranchPRRemediation
				perms["detail"] = m.Detail
				perms["language"] = rp.Language
				profile, reason := doctor.RecommendProfile(rp.Language, nil)
				perms["recommended_profile"] = profile
				perms["recommend_reason"] = reason
			}
		}
	}
	c.JSON(200, perms)
}

func (h *OnboardingHandler) handleRecommendProfile(c *gin.Context) {
	var req struct {
		Language  string   `json:"language"`
		PathHints []string `json:"path_hints"`
	}
	_ = c.ShouldBindJSON(&req)
	profile, reason := doctor.RecommendProfile(req.Language, req.PathHints)
	c.JSON(200, gin.H{"profile": profile, "reason": reason})
}

func (h *OnboardingHandler) handlePrivacyPreview(c *gin.Context) {
	var req struct {
		PrivacyMode string `json:"privacy_mode"`
		GiteaURL    string `json:"gitea_url"`
		AIProvider  string `json:"ai_provider"`
		AIBaseURL   string `json:"ai_base_url"`
		AIEnabled   bool   `json:"ai_enabled"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": "Invalid request"})
		return
	}
	mode := privacy.NormalizeMode(req.PrivacyMode)
	ai := privacy.EvaluateAIEgress(mode, req.AIProvider, req.AIBaseURL, req.AIEnabled)
	forgeClass := privacy.ClassUnknown
	forgeHost := ""
	if req.GiteaURL != "" {
		if u, err := url.Parse(req.GiteaURL); err == nil {
			forgeHost = u.Hostname()
		}
		if cls, _, err := privacy.ClassifyURL(req.GiteaURL); err == nil {
			forgeClass = cls
		}
	}
	result := "Repository Detective will apply privacy_mode=" + mode
	if mode == privacy.ModeLocalOnly {
		result = "Repository Detective will block external AI/notification egress"
		if forgeClass == privacy.ClassExternal {
			result += ", but findings and PR summaries may be transmitted to the configured external forge"
		}
		result += "."
	}
	c.JSON(200, gin.H{
		"privacy_mode":   mode,
		"ai_endpoint":    req.AIBaseURL,
		"ai_class":       ai.EndpointClass,
		"ai_allowed":     ai.Allowed,
		"ai_reason":      ai.Reason,
		"forge_host":     forgeHost,
		"forge_class":    forgeClass,
		"notifications":  "None configured in wizard",
		"result":         result,
		"guarantee_note": "LOCAL_ONLY is not air-gapped when the forge is EXTERNAL.",
	})
}

func (h *OnboardingHandler) handleOnboardVerify(c *gin.Context) {
	var req struct {
		onboardConnectionRequest
		PrivacyMode  string `json:"privacy_mode"`
		ScanProfile  string `json:"scan_profile"`
		PolicyMode   string `json:"policy_mode"`
		AIEnabled    bool   `json:"ai_enabled"`
		SelectedRepo string `json:"selected_repo"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": "Invalid request"})
		return
	}
	mode := privacy.NormalizeMode(req.PrivacyMode)
	profile := store.NormalizeScanProfile(req.ScanProfile)
	if profile == "" {
		profile = store.ScanProfileStandard
	}
	policy := req.PolicyMode
	if policy == "" {
		policy = "Observe"
	}

	in := doctor.Input{
		Version:            "onboard",
		Edition:            "community",
		ConfigValid:        true,
		AuthMode:           "api_key_only",
		APIKeyConfigured:   true,
		PrivacyMode:        mode,
		AIEnabled:          req.AIEnabled && req.AIProvider != "",
		AIProvider:         req.AIProvider,
		AIBaseURL:          req.AIBaseURL,
		AIModel:            req.AIModel,
		ForgeURL:           firstNonEmptyStr(req.GiteaURL, h.giteaURL),
		ForgeTokenSet:      req.GiteaToken != "",
		PublicURL:          firstNonEmptyStr(req.PublicURL, h.publicURL),
		WebhookSecretSet:   req.WebhookSecret != "",
		ScanProfile:        profile,
		PolicyMode:         policy,
		EnforcementMode:    store.EnforcementModeFromPolicyLevel(store.PolicyLevelFromEnforcementMode(policy)),
		DatabaseEnabled:    false,
		WorkspaceOK:        true,
		RequiredScanners:   store.ProfileDeclaredRequiredScanners(profile),
		ClassBIsolation:    "NOT_PROVEN",
		SkipLiveForgeProbe: false,
	}
	if in.EnforcementMode == "" {
		in.EnforcementMode = store.EnforcementObserve
	}
	ai := privacy.EvaluateAIEgress(mode, req.AIProvider, req.AIBaseURL, req.AIEnabled)
	in.AILocality = ai.EndpointClass
	in.AIEgressAllowed = ai.Allowed || !in.AIEnabled
	in.AIEgressReason = ai.Reason
	if in.ForgeURL != "" {
		if cls, _, err := privacy.ClassifyURL(in.ForgeURL); err == nil {
			in.ForgeLocality = cls
		}
	}

	for _, name := range in.RequiredScanners {
		in.ScannerTools = append(in.ScannerTools, doctor.ScannerToolInput{
			Name: name, Role: store.ScannerRoleRequired, EnabledInConfig: true,
			Available: true, Version: "assumed-image", StatusState: "onboard_assumed",
		})
	}

	url := firstNonEmptyStr(req.GiteaURL, h.giteaURL)
	if url != "" && req.GiteaToken != "" {
		client := gitea.NewClient(url, req.GiteaToken, h.logger)
		ctx, cancel := context.WithTimeout(c.Request.Context(), 25*time.Second)
		defer cancel()
		if err := client.TestConnection(ctx); err != nil {
			in.ForgeReachable = false
			in.ForgeAuthOK = false
			in.ForgeAuthDetail = err.Error()
		} else {
			in.ForgeReachable = true
			in.ForgeAuthOK = true
		}
		selected := req.SelectedRepo
		if selected == "" && len(req.Repositories) > 0 {
			selected = req.Repositories[0]
		}
		if selected != "" {
			parts := strings.SplitN(selected, "/", 2)
			if len(parts) == 2 {
				if rp, err := client.GetRepositoryPermissions(ctx, parts[0], parts[1]); err == nil {
					m := gitea.BuildPermissionMatrix(rp.Permissions)
					in.RepoPerms = &doctor.RepoPermissionResult{
						RepositoryRead: m.RepositoryRead, IssuesWrite: m.IssuesWrite,
						CommitStatusWrite: m.CommitStatusWrite, PRCommentWrite: m.PRCommentWrite,
						BranchPRRemediation: m.BranchPRRemediation, Detail: m.Detail,
					}
				}
				want := strings.TrimSuffix(in.PublicURL, "/") + "/webhook"
				if hooks, err := client.ListRepositoryHooks(ctx, parts[0], parts[1]); err == nil {
					if _, ok := gitea.FindHookByURL(hooks, want); ok {
						in.WebhookRegistered = true
						in.WebhookRegistrationDetail = "hook URL matched"
					}
				}
			}
		}
	}

	report := doctor.Run(in)
	c.JSON(200, gin.H{
		"report":           doctor.RedactReport(report),
		"onboarding_ready": report.OnboardingReady,
		"overall":          report.Overall,
		"next_action":      nextOnboardAction(report),
		"first_scan_note":  "A successful manual first scan records FIRST_SCAN_PROVEN locally; it is not Gitea webhook E2E proof (RD-017).",
	})
}

func nextOnboardAction(r doctor.Report) string {
	switch r.OnboardingReady {
	case doctor.Ready, doctor.ReadyWithLimits:
		return "Run first scan or open/push a test PR"
	default:
		return "Resolve required ERROR checks before scanning"
	}
}

func (h *OnboardingHandler) handleDefaultsExtended(c *gin.Context) {
	webhookURL := strings.TrimSuffix(h.publicURL, "/") + "/webhook"
	authMode := h.authMode
	if authMode == "" {
		authMode = "api_key_only"
	}
	fresh := h.isFreshInstall(c.Request.Context())
	recommendLocal := fresh || authMode == "local"
	c.JSON(200, gin.H{
		"gitea_url":            h.giteaURL,
		"public_url":           h.publicURL,
		"webhook_url":          webhookURL,
		"gitea_scan_orgs":      h.giteaScanOrgs,
		"ai_provider":          h.aiConfig.Provider,
		"ai_model":             h.aiConfig.Model,
		"ai_base_url":          h.aiConfig.BaseURL,
		"privacy_mode":         "hybrid",
		"scan_profile":         "standard",
		"policy_mode":          "Observe",
		"auth_mode":            authMode,
		"fresh_install":        fresh,
		"recommend_local_auth": recommendLocal,
		"auth_recommendation": gin.H{
			"mode":        "local",
			"summary":     "New installs should use operator login (local session). Keep the API key for scripts/MCP/OpenClaw only.",
			"bootstrap":   "/ui/bootstrap",
			"login":       "/ui/login",
			"password":    "Minimum 12 characters with letters and numbers. No default username/password.",
			"compat_note": "Existing api_key_only installs are unchanged until you set AUTH_MODE=local.",
		},
		"stages": []string{"connect", "select", "protect", "verify", "ready"},
		"policy_modes": []gin.H{
			{"id": "Observe", "summary": "Monitor only — no forge issue filing from policy gates"},
			{"id": "Warn", "summary": "File issues when policy requires (Community default-safe)"},
			{"id": "Enforce", "summary": "Gate PRs via commit status — not auto-merge"},
		},
		"privacy_modes": []gin.H{
			{"id": "local_only", "summary": "Block EXTERNAL AI and notification egress"},
			{"id": "hybrid", "summary": "External integrations allowed when configured and disclosed"},
			{"id": "external_ai_enabled", "summary": "External AI intentionally enabled"},
		},
	})
}

func (h *OnboardingHandler) isFreshInstall(ctx context.Context) bool {
	if h.userCounter == nil {
		return true // soft recommend when we cannot observe state yet
	}
	users, err := h.userCounter(ctx)
	if err != nil {
		return true
	}
	if users > 0 {
		return false
	}
	if h.repoCounter == nil {
		return true
	}
	repos, err := h.repoCounter(ctx)
	if err != nil {
		return users == 0
	}
	return users == 0 && repos == 0
}

// SetInstallCounters wires optional DB counters used for fresh-install detection (RD-032).
func (h *OnboardingHandler) SetInstallCounters(users, repos func(context.Context) (int, error)) {
	if h == nil {
		return
	}
	h.userCounter = users
	h.repoCounter = repos
}

// Register extended onboarding routes (called from RegisterRoutes).
func (h *OnboardingHandler) registerPhase4Routes(onboardAPI *gin.RouterGroup) {
	onboardAPI.POST("/permissions", h.handleVerifyPermissions)
	onboardAPI.POST("/recommend-profile", h.handleRecommendProfile)
	onboardAPI.POST("/privacy-preview", h.handlePrivacyPreview)
	onboardAPI.POST("/verify", h.handleOnboardVerify)
}
