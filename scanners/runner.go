package scanners

import (
	"context"

	"git.commsnet.org/commstech/repository-detective/models"
	"github.com/sirupsen/logrus"
)

var defaultRegistry = DefaultScannerRegistry()

// RunAll executes enabled external scanners against a workspace.
func RunAll(ctx context.Context, logger *logrus.Logger, dir string, entries []FileEntry, cfg Config, enableSecurity, enableQuality bool) RunSummary {
	return defaultRegistry.RunAll(ctx, RunRequest{
		Logger:         logger,
		Workspace:      dir,
		Entries:        entries,
		Config:         cfg,
		EnableSecurity: enableSecurity,
		EnableQuality:  enableQuality,
	})
}

// RunAllCandidates preserves the previous helper return type for callers that only need findings.
func RunAllCandidates(ctx context.Context, logger *logrus.Logger, dir string, entries []FileEntry, cfg Config, enableSecurity, enableQuality bool) []models.CandidateFinding {
	summary := RunAll(ctx, logger, dir, entries, cfg, enableSecurity, enableQuality)
	raw := summary.Candidates()
	candidates := make([]models.CandidateFinding, 0, len(raw))
	for _, finding := range raw {
		candidates = append(candidates, finding.ToCandidateFinding())
	}
	return candidates
}

// BuildFileEntries converts analyzer file content into scanner workspace entries.
func BuildFileEntries(files []FileContent) []FileEntry {
	entries := make([]FileEntry, 0, len(files))
	for _, file := range files {
		entries = append(entries, FileEntry{
			Path:    file.Path,
			Content: file.Content,
		})
	}
	return entries
}

// FileContent mirrors analyzers.FileContent to avoid an import cycle.
type FileContent struct {
	Path     string
	Content  string
	Language string
}
