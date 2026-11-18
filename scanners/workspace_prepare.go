package scanners

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"git.commsnet.org/commstech/repository-detective/models"
)

// ArchiveDownloader fetches a repository archive from Gitea.
type ArchiveDownloader interface {
	DownloadRepositoryArchive(ctx context.Context, owner, repo, ref string, maxBytes int64) (path string, cleanup func(), written int64, err error)
}

// PreparedWorkspace is a scanner-ready directory with metadata.
type PreparedWorkspace struct {
	Dir     string
	Cleanup func()
	Meta    models.WorkspaceMeta
	Entries []FileEntry
}

// PrepareWorkspace builds a scanner workspace using the configured mode.
func PrepareWorkspace(
	ctx context.Context,
	cfg WorkspaceConfig,
	downloader ArchiveDownloader,
	owner, repo, ref string,
	commitPinned bool,
	apiEntries []FileEntry,
) (PreparedWorkspace, error) {
	meta := models.WorkspaceMeta{
		RefUsed:      ref,
		CommitPinned: commitPinned,
	}

	mode := cfg.NormalizedMode()
	switch mode {
	case WorkspaceModeArchive:
		meta.ModeUsed = WorkspaceModeArchive
		prepared, err := prepareArchiveWorkspace(ctx, cfg, downloader, owner, repo, ref, meta)
		if err != nil {
			return PreparedWorkspace{}, err
		}
		return prepared, nil
	case WorkspaceModeAuto:
		meta.ModeUsed = WorkspaceModeArchive
		prepared, err := prepareArchiveWorkspace(ctx, cfg, downloader, owner, repo, ref, meta)
		if err == nil {
			return prepared, nil
		}
		meta.FallbackUsed = true
		meta.WorkspaceError = err.Error()
		meta.ModeUsed = WorkspaceModeAPI
		return prepareAPIWorkspace(cfg, apiEntries, meta)
	default:
		meta.ModeUsed = WorkspaceModeAPI
		return prepareAPIWorkspace(cfg, apiEntries, meta)
	}
}

func prepareArchiveWorkspace(
	ctx context.Context,
	cfg WorkspaceConfig,
	downloader ArchiveDownloader,
	owner, repo, ref string,
	meta models.WorkspaceMeta,
) (PreparedWorkspace, error) {
	if downloader == nil {
		return PreparedWorkspace{}, fmt.Errorf("archive downloader not configured")
	}

	timeout := time.Duration(cfg.archiveTimeoutSeconds()) * time.Second
	archiveCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	archivePath, archiveCleanup, archiveSize, err := downloader.DownloadRepositoryArchive(archiveCtx, owner, repo, ref, cfg.maxArchiveBytes())
	if err != nil {
		return PreparedWorkspace{}, err
	}
	defer archiveCleanup()

	destRoot, err := os.MkdirTemp("", "rd-archive-*")
	if err != nil {
		return PreparedWorkspace{}, err
	}
	cleanup := func() {
		_ = os.RemoveAll(destRoot)
	}

	fileCount, totalBytes, truncated, err := ExtractZipArchive(archivePath, destRoot, cfg.maxFiles(), cfg.maxArchiveBytes())
	if err != nil {
		cleanup()
		return PreparedWorkspace{}, err
	}

	entries, err := listWorkspaceEntries(destRoot, cfg.maxFiles())
	if err != nil {
		cleanup()
		return PreparedWorkspace{}, err
	}
	if len(entries) == 0 {
		cleanup()
		return PreparedWorkspace{}, fmt.Errorf("archive extracted zero files")
	}

	meta.FileCount = fileCount
	meta.TotalSizeBytes = totalBytes
	meta.TruncatedOrLimited = truncated
	if archiveSize > cfg.maxArchiveBytes() {
		meta.TruncatedOrLimited = true
	}

	return PreparedWorkspace{
		Dir:     destRoot,
		Cleanup: cleanup,
		Meta:    meta,
		Entries: entries,
	}, nil
}

func prepareAPIWorkspace(cfg WorkspaceConfig, apiEntries []FileEntry, meta models.WorkspaceMeta) (PreparedWorkspace, error) {
	dir, cleanup, err := CreateWorkspace(apiEntries)
	if err != nil {
		return PreparedWorkspace{}, err
	}

	entries := append([]FileEntry(nil), apiEntries...)
	var totalBytes int64
	for _, entry := range entries {
		totalBytes += int64(len(entry.Content))
	}

	meta.FileCount = len(entries)
	meta.TotalSizeBytes = totalBytes
	if len(entries) >= cfg.maxFiles() {
		meta.TruncatedOrLimited = true
	}

	return PreparedWorkspace{
		Dir:     dir,
		Cleanup: cleanup,
		Meta:    meta,
		Entries: entries,
	}, nil
}

func listWorkspaceEntries(root string, maxFiles int) ([]FileEntry, error) {
	if maxFiles <= 0 {
		maxFiles = 5000
	}

	absRoot, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}

	var entries []FileEntry
	err = filepath.WalkDir(absRoot, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			if d.Type()&os.ModeSymlink != 0 {
				return filepath.SkipDir
			}
			return nil
		}
		if d.Type()&os.ModeSymlink != 0 {
			return nil
		}
		if len(entries) >= maxFiles {
			return errWorkspaceFileLimit
		}

		rel, err := filepath.Rel(absRoot, path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		if _, err := ValidateWorkspacePath(absRoot, rel); err != nil {
			return nil
		}
		entries = append(entries, FileEntry{Path: rel})
		return nil
	})
	if err == errWorkspaceFileLimit {
		return entries, nil
	}
	if err != nil {
		return nil, err
	}
	return entries, nil
}

var errWorkspaceFileLimit = fmt.Errorf("workspace file limit reached")

// LogMeta writes workspace metadata to logs with optional scan ID.
func LogMeta(meta models.WorkspaceMeta, scanID string, log interface {
	Infof(string, ...any)
}) {
	if scanID != "" {
		log.Infof("[CAH:WORKSPACE] scan_id=%s mode=%s ref=%s commit_pinned=%v fallback=%v files=%d bytes=%d truncated=%v err=%q",
			scanID, meta.ModeUsed, meta.RefUsed, meta.CommitPinned, meta.FallbackUsed,
			meta.FileCount, meta.TotalSizeBytes, meta.TruncatedOrLimited, meta.WorkspaceError)
		return
	}
	log.Infof("[CAH:WORKSPACE] mode=%s ref=%s commit_pinned=%v fallback=%v files=%d bytes=%d truncated=%v err=%q",
		meta.ModeUsed, meta.RefUsed, meta.CommitPinned, meta.FallbackUsed,
		meta.FileCount, meta.TotalSizeBytes, meta.TruncatedOrLimited, meta.WorkspaceError)
}
