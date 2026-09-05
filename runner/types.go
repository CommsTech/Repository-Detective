package runner

const (
	ContractVersion = 1

	JobTypeScanFullRepo       = "scan"
	JobTypeSBOM               = "sbom"
	JobTypeGraph              = "graph"
	JobTypePreinstallAudit    = "preinstall_audit"
	JobTypeRemediationVerify  = "remediation_verify"
	JobTypeContainerImageScan = "container_image_scan"
	JobTypeScanFullRepoLegacy = "scan_full_repo"

	JobStatusCompleted = "completed"
	JobStatusFailed    = "failed"

	ModeCore         = "core"
	ModeNative       = "native"
	ModeGiteaActions = "gitea_actions"
	ModeAuto         = "auto"

	HeaderTimestamp = "X-Runner-Timestamp"
	HeaderNonce     = "X-Runner-Nonce"
	HeaderSignature = "X-Runner-Signature"

	MaxClockSkewSeconds = 300
)

var (
	AllowedTasks = []string{"scanners", "health", "graph"}
	ForbiddenTasks = []string{
		"issue_create", "status_update", "pull_request_create",
		"secret_access", "dependency_install", "repo_script_execution",
	}
	ContainerScanTasks = []string{"container_scan"}
)
