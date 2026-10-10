package preinstall

import (
	"strings"

	"git.commsnet.org/commstech/repository-detective/store"
)

func filterPreinstallFindings(in []store.AuditFinding) []store.AuditFinding {
	if len(in) == 0 {
		return in
	}
	out := make([]store.AuditFinding, 0, len(in))
	for _, f := range in {
		if skipPreinstallPath(f.FilePath) {
			continue
		}
		if isEnvFallbackPattern(f.Title, f.EvidenceRedacted) {
			f.Severity = "info"
			f.Confidence = minConfidence(f.Confidence, 0.45)
			f.Title = "Environment fallback pattern (review only): " + f.Title
		}
		if f.Confidence > 0 && f.Confidence < 0.6 {
			if strings.EqualFold(f.Source, "graph") || isHealthQualityFinding(f.Source, f.Category) {
				f.Severity = "info"
				f.Confidence = minConfidence(f.Confidence, 0.55)
			}
		}
		out = append(out, f)
	}
	return out
}

func minConfidence(current, cap float64) float64 {
	if current <= 0 {
		return cap
	}
	if current < cap {
		return current
	}
	return cap
}
