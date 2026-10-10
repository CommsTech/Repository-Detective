package containers

import (
	"fmt"
	"strings"
)

// ErrScanningDisabled is returned when container scanning is off.
var ErrScanningDisabled = fmt.Errorf("container scanning is disabled")

// ErrRunnerRequired is returned when core cannot execute image scans directly.
var ErrRunnerRequired = fmt.Errorf("container image scan requires a native runner")

// ErrCoreDockerForbidden is returned when core Docker socket would be needed.
var ErrCoreDockerForbidden = fmt.Errorf("core Docker socket access is disabled by policy")

// ErrImageNotAllowed is returned when image fails allowlist policy.
var ErrImageNotAllowed = fmt.Errorf("image not allowed by container scan policy")

// ValidateEnqueue checks whether core may enqueue an image scan job.
func ValidateEnqueue(cfg Config, image string, onCore bool) error {
	cfg = cfg.Normalized()
	if !cfg.Enabled {
		return ErrScanningDisabled
	}
	image = strings.TrimSpace(image)
	if image == "" {
		return fmt.Errorf("image reference required")
	}
	if !cfg.RegistryAllowed(image) {
		return ErrImageNotAllowed
	}
	if cfg.RequireRunner && onCore {
		return ErrRunnerRequired
	}
	if onCore && !cfg.AllowCoreDockerSocket {
		return ErrCoreDockerForbidden
	}
	return nil
}

// BuildScanPayload constructs a runner payload from config and target.
func BuildScanPayload(cfg Config, repoID int64, scanID string, ref ImageReference) ScanPayload {
	cfg = cfg.Normalized()
	return ScanPayload{
		TargetType:     ref.TargetType,
		Image:          ref.Image,
		RepositoryID:   repoID,
		ScanID:         scanID,
		PullPolicy:     cfg.PullPolicy,
		Tools:          cfg.ToolList(),
		GenerateSBOM:   cfg.GenerateSBOM,
		TimeoutSeconds: cfg.TimeoutSeconds,
		SourceFile:     ref.FilePath,
		SourceLine:     ref.Line,
		ServiceName:    ref.ServiceName,
	}
}
