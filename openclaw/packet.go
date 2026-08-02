package openclaw

import (
	"encoding/json"
	"fmt"
	"strings"

	"git.commsnet.org/commstech/repository-detective/store"
)

// PacketInput is raw scan context for building a review packet.
type PacketInput struct {
	ScanID          string
	Repository      store.Repository
	ScanType        ScanType
	IssueFiling     string
	RemediationPR   string
	Languages       []string
	ScannerCoverage []string
	GraphState      string
	SBOMState       string
	ContainerState  string
	Findings        []store.Finding
	Instances       map[int64]store.FindingInstance
	History         map[string]FindingHistory
}

// BuildPacket constructs a review packet from scan findings (no raw source).
func BuildPacket(in PacketInput, cfg Config) (ReviewPacket, error) {
	cfg = cfg.Normalized()
	if cfg.SendFullFiles {
		return ReviewPacket{}, fmt.Errorf("full file sending is disabled by policy")
	}
	pkt := ReviewPacket{
		ScanID:   strings.TrimSpace(in.ScanID),
		RepoID:   in.Repository.ID,
		RepoName: in.Repository.FullName,
		ScanType: in.ScanType,
		Policy: ReviewPolicy{
			IssueFiling:   in.IssueFiling,
			RemediationPR: in.RemediationPR,
			AdvisoryOnly:  true,
		},
		Summary: ReviewSummary{
			Languages:          append([]string(nil), in.Languages...),
			ScannerCoverage:    append([]string(nil), in.ScannerCoverage...),
			GraphState:         in.GraphState,
			SBOMState:          in.SBOMState,
			ContainerScanState: in.ContainerState,
		},
	}
	candidates, _ := SelectCAHCandidates(in.Findings, in.Instances, in.History, cfg, cfg.CAH)
	for _, f := range candidates {
		inst := in.Instances[f.ID]
		hist := in.History[f.Fingerprint]
		evidence := inst.EvidenceRedacted
		if cfg.SendSourceSnippets {
			if snip := extractSnippet(inst.RawMetadataJSON); snip != "" {
				evidence = snip
			}
		}
		pkt.Findings = append(pkt.Findings, FindingInput{
			Fingerprint:         f.Fingerprint,
			RuleID:              f.RuleID,
			Title:               f.Title,
			Severity:            f.Severity,
			Confidence:          confidenceBand(f.Confidence),
			Source:              f.Source,
			Path:                f.FilePath,
			Line:                f.Line,
			DescriptionRedacted: f.Title,
			EvidenceRedacted:    evidence,
			History:             hist,
		})
	}
	return pkt, nil
}

func confidenceBand(c float64) string {
	switch {
	case c >= 0.85:
		return "high"
	case c >= 0.6:
		return "medium"
	default:
		return "low"
	}
}

func extractSnippet(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	var meta map[string]any
	if err := json.Unmarshal(raw, &meta); err != nil {
		return ""
	}
	for _, key := range []string{"code_snippet", "snippet", "evidence"} {
		if v, ok := meta[key].(string); ok && strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}
