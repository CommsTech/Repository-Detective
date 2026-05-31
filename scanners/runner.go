package scanners

import (
	"context"

	"git.commsnet.org/commstech/bugbot/models"
	"github.com/sirupsen/logrus"
)

// RunAll executes enabled external scanners against a workspace.
func RunAll(ctx context.Context, logger *logrus.Logger, dir string, entries []FileEntry, cfg Config, enableSecurity, enableQuality bool) []models.CandidateFinding {
	if !enableSecurity && !enableQuality {
		return nil
	}

	var raw []Finding

	if cfg.EnableTrivy && enableSecurity {
		findings, err := RunTrivy(ctx, logger, dir, cfg)
		if err != nil {
			logger.Warnf("[SCANNER] trivy error: %v", err)
		} else {
			raw = append(raw, findings...)
		}
	}

	if cfg.EnableGrype && enableSecurity {
		findings, err := RunGrype(ctx, logger, dir, cfg)
		if err != nil {
			logger.Warnf("[SCANNER] grype error: %v", err)
		} else {
			raw = append(raw, findings...)
		}
	}

	if cfg.EnableLinters {
		findings, err := RunLinters(ctx, logger, dir, entries, enableSecurity, enableQuality, cfg)
		if err != nil {
			logger.Warnf("[SCANNER] linters error: %v", err)
		} else {
			raw = append(raw, findings...)
		}
	}

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
