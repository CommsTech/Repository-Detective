package containers

import (
	"strings"
	"time"
)

// TargetType identifies how an image scan target was resolved.
type TargetType string

const (
	TargetRegistryImage       TargetType = "registry_image"
	TargetLocalDockerImage    TargetType = "local_docker_image"
	TargetComposeFile         TargetType = "compose_file"
	TargetKubernetesManifest  TargetType = "kubernetes_manifest"
	TargetRunnerHostInventory TargetType = "runner_host_inventory"
)

// PullPolicy controls image pull behavior on the runner.
type PullPolicy string

const (
	PullNever    PullPolicy = "never"
	PullIfMissing PullPolicy = "if_missing"
	PullAlways   PullPolicy = "always"
)

// ScanTargets lists explicit scan targets from configuration.
type ScanTargets struct {
	Registries           []string `mapstructure:"registries" json:"registries"`
	Images               []string `mapstructure:"images" json:"images"`
	ComposeFiles         []string `mapstructure:"compose_files" json:"compose_files"`
	KubernetesManifests  []string `mapstructure:"kubernetes_manifests" json:"kubernetes_manifests"`
}

// ScanTools toggles which scanners run against an image.
type ScanTools struct {
	Trivy bool `mapstructure:"trivy" json:"trivy"`
	Grype bool `mapstructure:"grype" json:"grype"`
	Syft  bool `mapstructure:"syft" json:"syft"`
}

// Config controls container image scanning (opt-in, runner-first).
type Config struct {
	Enabled                    bool        `mapstructure:"container_scanning_enabled"`
	Targets                    ScanTargets `mapstructure:"container_scan_targets"`
	DefaultPolicy              string      `mapstructure:"container_scan_default_policy"`
	CreateIssues               bool        `mapstructure:"container_scan_create_issues"`
	RequireRunner              bool        `mapstructure:"container_scan_require_runner"`
	AllowCoreDockerSocket      bool        `mapstructure:"container_scan_allow_core_docker_socket"`
	Tools                      ScanTools   `mapstructure:"container_scan_tools"`
	PullPolicy                 PullPolicy  `mapstructure:"container_scan_pull_policy"`
	TimeoutSeconds             int         `mapstructure:"container_scan_timeout_seconds"`
	MaxImageSizeMB             int         `mapstructure:"container_scan_max_image_size_mb"`
	IncludeOSPackages          bool        `mapstructure:"container_scan_include_os_packages"`
	IncludeLanguagePackages    bool        `mapstructure:"container_scan_include_language_packages"`
	GenerateSBOM               bool        `mapstructure:"container_scan_generate_sbom"`
	HistoryLayers              bool        `mapstructure:"container_scan_history_layers"`
	FailOnScannerMissing       bool        `mapstructure:"container_scan_fail_on_scanner_missing"`
	RegistryCredentialsEnv     []string    `mapstructure:"container_registry_credentials_env"`
	AllowedRegistries          []string    `mapstructure:"container_scan_allowed_registries"`
	BlockedRegistries          []string    `mapstructure:"container_scan_blocked_registries"`
	AllowedRunnerLabels        []string    `mapstructure:"container_scan_allowed_runner_labels"`
}

// DefaultConfig returns safe defaults (disabled, runner-required, no core socket).
func DefaultConfig() Config {
	return Config{
		Enabled:               false,
		DefaultPolicy:         "report_only",
		CreateIssues:          false,
		RequireRunner:         true,
		AllowCoreDockerSocket: false,
		Tools: ScanTools{
			Trivy: true,
			Grype: true,
			Syft:  true,
		},
		PullPolicy:              PullIfMissing,
		TimeoutSeconds:          900,
		MaxImageSizeMB:          4096,
		IncludeOSPackages:       true,
		IncludeLanguagePackages: true,
		GenerateSBOM:            true,
		HistoryLayers:           true,
		FailOnScannerMissing:    false,
		RegistryCredentialsEnv:  []string{"REGISTRY_AUTH_FILE", "DOCKER_CONFIG"},
		AllowedRunnerLabels:     []string{"container-scan"},
	}
}

// Normalized applies defaults to zero values.
func (c Config) Normalized() Config {
	out := c
	def := DefaultConfig()
	if out.DefaultPolicy == "" {
		out.DefaultPolicy = def.DefaultPolicy
	}
	if out.PullPolicy == "" {
		out.PullPolicy = def.PullPolicy
	}
	if out.TimeoutSeconds <= 0 {
		out.TimeoutSeconds = def.TimeoutSeconds
	}
	if out.MaxImageSizeMB <= 0 {
		out.MaxImageSizeMB = def.MaxImageSizeMB
	}
	if len(out.RegistryCredentialsEnv) == 0 {
		out.RegistryCredentialsEnv = append([]string(nil), def.RegistryCredentialsEnv...)
	}
	if len(out.AllowedRunnerLabels) == 0 {
		out.AllowedRunnerLabels = append([]string(nil), def.AllowedRunnerLabels...)
	}
	return out
}

// EnabledForScan reports whether image scanning may be invoked.
func (c Config) EnabledForScan() bool {
	return c.Normalized().Enabled
}

// ScanPayload is the runner job extension for container_image_scan.
type ScanPayload struct {
	TargetType     TargetType `json:"target_type"`
	Image          string     `json:"image"`
	RepositoryID   int64      `json:"repo_id"`
	ScanID         string     `json:"scan_id"`
	PullPolicy     PullPolicy `json:"pull_policy"`
	Tools          []string   `json:"tools"`
	GenerateSBOM   bool       `json:"generate_sbom"`
	TimeoutSeconds int        `json:"timeout_seconds"`
	SourceFile     string     `json:"source_file,omitempty"`
	SourceLine     int        `json:"source_line,omitempty"`
	ServiceName    string     `json:"service_name,omitempty"`
}

// ImageReference is a discovered image in a repository.
type ImageReference struct {
	Image         string     `json:"image"`
	Tag           string     `json:"tag"`
	Digest        string     `json:"digest,omitempty"`
	TargetType    TargetType `json:"target_type"`
	FilePath      string     `json:"file_path"`
	Line          int        `json:"line"`
	ServiceName   string     `json:"service_name,omitempty"`
	MutableTag    bool       `json:"mutable_tag"`
	PrivateRegistry bool     `json:"private_registry"`
	RepoID        int64      `json:"repository_id,omitempty"`
}

// ScanCoverage describes which tools ran successfully.
type ScanCoverage struct {
	Trivy string `json:"trivy"`
	Grype string `json:"grype"`
	Syft  string `json:"syft"`
}

// ScanResult is normalized output from a container image scan.
type ScanResult struct {
	Image          string       `json:"image"`
	Digest         string       `json:"digest,omitempty"`
	BaseImage      string       `json:"base_image,omitempty"`
	Labels         []string     `json:"labels,omitempty"`
	Coverage       ScanCoverage `json:"coverage"`
	SBOMPath       string       `json:"sbom_path,omitempty"`
	SBOMFormat     string       `json:"sbom_format,omitempty"`
	VulnCount      int          `json:"vuln_count"`
	Findings       []ScanFinding `json:"findings"`
	Warnings       []string     `json:"warnings,omitempty"`
	StartedAt      time.Time    `json:"started_at"`
	FinishedAt     time.Time    `json:"finished_at"`
}

// ScanFinding is one vulnerability or policy finding on an image.
type ScanFinding struct {
	RuleID      string  `json:"rule_id"`
	Severity    string  `json:"severity"`
	Confidence  float64 `json:"confidence"`
	Title       string  `json:"title"`
	Description string  `json:"description"`
	PackageName string  `json:"package_name,omitempty"`
	CVE         string  `json:"cve,omitempty"`
}

// ToolList returns enabled tool names from config.
func (c Config) ToolList() []string {
	c = c.Normalized()
	var out []string
	if c.Tools.Trivy {
		out = append(out, "trivy")
	}
	if c.Tools.Grype {
		out = append(out, "grype")
	}
	if c.Tools.Syft {
		out = append(out, "syft")
	}
	return out
}

// RegistryAllowed checks allow/block lists (empty allowlist = allow all non-blocked).
func (c Config) RegistryAllowed(image string) bool {
	image = strings.ToLower(strings.TrimSpace(image))
	for _, blocked := range c.BlockedRegistries {
		b := strings.ToLower(strings.TrimSpace(blocked))
		if b != "" && strings.Contains(image, b) {
			return false
		}
	}
	if len(c.AllowedRegistries) == 0 {
		return true
	}
	for _, allowed := range c.AllowedRegistries {
		if registryMatchesAllowed(image, allowed) {
			return true
		}
	}
	return false
}

func registryMatchesAllowed(image, allowed string) bool {
	a := strings.ToLower(strings.TrimSpace(allowed))
	if a == "" {
		return false
	}
	if strings.Contains(image, a) {
		return true
	}
	if a == "docker.io" || a == "index.docker.io" {
		return isImplicitDockerHubImage(image)
	}
	return false
}

func isImplicitDockerHubImage(image string) bool {
	image = strings.TrimSpace(image)
	if image == "" || strings.Contains(image, "://") {
		return false
	}
	host := image
	if slash := strings.Index(host, "/"); slash >= 0 {
		host = host[:slash]
	}
	if colon := strings.Index(host, ":"); colon >= 0 {
		host = host[:colon]
	}
	if host == "localhost" {
		return false
	}
	return !strings.Contains(host, ".")
}
