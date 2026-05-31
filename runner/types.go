package runner

const (
	ContractVersion = 1

	JobTypeScanFullRepo = "scan_full_repo"

	JobStatusCompleted = "completed"
	JobStatusFailed    = "failed"

	ModeCore         = "core"
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
)
