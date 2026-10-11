package models

// WorkspaceMeta describes how the scanner workspace was prepared.
type WorkspaceMeta struct {
	ModeUsed           string `json:"workspace_mode_used"`
	RefUsed            string `json:"ref_used"`
	CommitPinned       bool   `json:"commit_pinned"`
	FallbackUsed       bool   `json:"fallback_used"`
	FileCount          int    `json:"file_count"`
	TotalSizeBytes     int64  `json:"total_size_bytes"`
	TruncatedOrLimited bool   `json:"truncated_or_limited"`
	WorkspaceError     string `json:"workspace_error,omitempty"`
}
