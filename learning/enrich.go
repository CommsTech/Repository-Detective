package learning

import "git.commsnet.org/commstech/repository-detective/findinglearn"

// StructuralHash computes a deterministic shape hash for grouping repeated patterns.
func StructuralHash(ruleID, category, codeSnippet string) string {
	return findinglearn.StructuralHash(ruleID, category, codeSnippet)
}

// ReachabilityInput describes graph-informed priority hints.
type ReachabilityInput = findinglearn.ReachabilityInput

// ActionabilityAdjust returns severity/confidence deltas from reachability.
func ActionabilityAdjust(severity string, confidence float64, in ReachabilityInput) (string, float64, string) {
	return findinglearn.ActionabilityAdjust(severity, confidence, in)
}

// ClassifyPath heuristics for reachability input.
func ClassifyPath(path string) ReachabilityInput {
	return findinglearn.ClassifyPath(path)
}
