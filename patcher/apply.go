package patcher

import (
	"fmt"
	"strings"

	"git.commsnet.org/commstech/repository-detective/remediation"
)

// PatchResult describes an applied patch.
type PatchResult struct {
	FilesChanged []string
	DiffLines    int
	Summary      string
}

// SupportsPatch reports whether a deterministic patcher exists for the plan.
func SupportsPatch(plan remediation.Plan) bool {
	source := strings.ToLower(plan.Source)
	rule := strings.ToLower(plan.RuleID)
	switch source {
	case "staticcheck":
		return isStaticcheckPatchable(rule)
	case "hadolint":
		return isHadolintPatchable(rule)
	default:
		return false
	}
}

// ApplyPatch applies a supported deterministic patch in workspaceDir.
func ApplyPatch(plan remediation.Plan, workspaceDir string, maxFiles, maxLines int) (PatchResult, error) {
	if !SupportsPatch(plan) {
		return PatchResult{}, fmt.Errorf("no patcher available for this rule yet")
	}
	source := strings.ToLower(plan.Source)
	switch source {
	case "staticcheck":
		return applyStaticcheckPatch(plan, workspaceDir, maxFiles, maxLines)
	case "hadolint":
		return applyHadolintPatch(plan, workspaceDir, maxFiles, maxLines)
	default:
		return PatchResult{}, fmt.Errorf("no patcher available for this rule yet")
	}
}

func isStaticcheckPatchable(rule string) bool {
	switch rule {
	case "s1039", "SA1006", "ST1017":
		return true
	default:
		return strings.HasPrefix(rule, "s1039")
	}
}

func isHadolintPatchable(rule string) bool {
	switch strings.ToUpper(rule) {
	case "DL3015", "DL3016", "DL3018":
		return true
	default:
		upper := strings.ToUpper(rule)
		return strings.HasPrefix(upper, "DL3015") || strings.HasPrefix(upper, "DL3018")
	}
}
