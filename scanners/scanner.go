package scanners

import (
	"context"

	"github.com/sirupsen/logrus"
)

// Scanner executes one external deterministic tool against a prepared workspace.
type Scanner interface {
	// Name is the registry identifier used for ordering and RunResult.Scanner.
	Name() string
	// Run executes the scanner and returns one or more normalized results.
	Run(ctx context.Context, req RunRequest) []RunResult
}

// RunRequest carries shared inputs for scanner execution.
type RunRequest struct {
	Logger         *logrus.Logger
	Workspace      string
	Entries        []FileEntry
	Config         Config
	EnableSecurity bool
	EnableQuality  bool
}
