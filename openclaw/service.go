package openclaw

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"git.commsnet.org/commstech/repository-detective/ai"
	"git.commsnet.org/commstech/repository-detective/store"
)

// ReviewStore persists advisory reviews.
type ReviewStore interface {
	CreateAIAdvisoryReview(ctx context.Context, rec store.AIAdvisoryReview) (store.AIAdvisoryReview, error)
	UpdateAIAdvisoryReview(ctx context.Context, rec store.AIAdvisoryReview) error
	GetAIAdvisoryReviewByScanID(ctx context.Context, scanID string) (store.AIAdvisoryReview, error)
	ListAIAdvisoryRecommendations(ctx context.Context, reviewID string) ([]store.AIAdvisoryRecommendation, error)
	CreateAIAdvisoryRecommendation(ctx context.Context, rec store.AIAdvisoryRecommendation) (store.AIAdvisoryRecommendation, error)
	UpdateAIAdvisoryRecommendationStatus(ctx context.Context, id int64, status string) error
}

// Service orchestrates advisory OpenClaw reviews.
type Service struct {
	cfg       Config
	store     ReviewStore
	transport ai.ChatTransport
}

// NewService creates an advisory review service.
func NewService(cfg Config, s ReviewStore, transport ai.ChatTransport) *Service {
	return &Service{cfg: cfg.Normalized(), store: s, transport: transport}
}

// RunReview builds a redacted packet, calls OpenClaw, and stores advisory results.
// Deterministic findings are never modified.
func (s *Service) RunReview(ctx context.Context, in PacketInput) (ReviewResult, error) {
	if s == nil || s.store == nil {
		return ReviewResult{Status: "skipped", Error: "review service unavailable"}, fmt.Errorf("review service unavailable")
	}
	cfg := s.cfg.Normalized()
	result := ReviewResult{Status: "skipped"}
	if !cfg.CanInvoke() {
		result.Error = "ai recommendations disabled or not configured"
		return result, nil
	}
	if !cfg.AllowsScanType(string(in.ScanType)) {
		result.Error = "scan type not allowed for ai review"
		return result, nil
	}
	reviewID, err := newReviewID()
	if err != nil {
		return result, err
	}
	result.ReviewID = reviewID
	pkt, err := BuildPacket(in, cfg)
	if err != nil {
		return result, err
	}
	redactions, err := RedactPacket(&pkt, cfg)
	result.RedactionCount = redactions
	if err != nil {
		result.Status = "failed"
		result.Error = err.Error()
		if cfg.CAH.FailClosedOnRedaction {
			return result, err
		}
		return result, nil
	}
	result.FindingsSent = len(pkt.Findings)
	if len(pkt.Findings) == 0 {
		result.Status = "skipped"
		result.Error = "no findings to review"
		return result, nil
	}
	packetJSON, _ := json.Marshal(pkt)
	now := time.Now().UTC()
	rec := store.AIAdvisoryReview{
		ReviewID: reviewID, ScanID: in.ScanID, RepositoryID: in.Repository.ID,
		ScanType: string(in.ScanType), Status: "running",
		FindingsSent: len(pkt.Findings), RedactionCount: redactions,
		PacketJSON: packetJSON, StartedAt: now, CreatedAt: now,
	}
	rec, err = s.store.CreateAIAdvisoryReview(ctx, rec)
	if err != nil {
		return result, err
	}
	client, err := NewClient(cfg, s.transport)
	if err != nil {
		result.Error = err.Error()
		if failErr := s.failReview(ctx, rec, result.Error); failErr != nil {
			return result, fmt.Errorf("%w (also failed to persist review failure: %v)", err, failErr)
		}
		return result, err
	}
	reviewResult, reviewErr := client.Review(ctx, cfg, reviewID, pkt)
	if reviewErr != nil && reviewResult.Error == "" {
		reviewResult.Error = reviewErr.Error()
		if reviewResult.Status == "" {
			reviewResult.Status = "failed"
		}
	}
	reviewResult.RedactionCount = redactions
	if reviewResult.FindingsSent == 0 {
		reviewResult.FindingsSent = len(pkt.Findings)
	}
	finished := time.Now().UTC()
	rec.Status = reviewResult.Status
	rec.Model = reviewResult.Model
	rec.FindingsSent = reviewResult.FindingsSent
	rec.RedactionCount = reviewResult.RedactionCount
	rec.RecommendationsCount = reviewResult.RecommendationsCount
	rec.OverallAssessment = reviewResult.OverallAssessment
	rec.ErrorMessage = reviewResult.Error
	rec.FinishedAt = &finished
	if cfg.StoreResponses && reviewResult.Response != nil {
		sanitized, marshalErr := json.Marshal(reviewResult.Response)
		if marshalErr != nil {
			rec.ErrorMessage = strings.TrimSpace(rec.ErrorMessage + "; response marshal: " + marshalErr.Error())
		} else {
			rec.ResponseJSON = sanitized
		}
	}
	if err := s.store.UpdateAIAdvisoryReview(ctx, rec); err != nil {
		return reviewResult, fmt.Errorf("update ai advisory review: %w", err)
	}
	if reviewResult.Response != nil {
		for _, r := range reviewResult.Response.Recommendations {
			gaps, gapErr := json.Marshal(r.EvidenceGaps)
			if gapErr != nil {
				gaps = []byte("[]")
			}
			if _, err := s.store.CreateAIAdvisoryRecommendation(ctx, store.AIAdvisoryRecommendation{
				ReviewID: reviewID, FindingFingerprint: r.Fingerprint,
				Classification: r.Classification, SuggestedAction: r.SuggestedAction,
				SuggestedSeverity: r.SuggestedSeverity, SuggestedConfidence: r.SuggestedConfidence,
				Reason: r.Reason, EvidenceGapsJSON: string(gaps),
				OperatorStatus: "pending", CreatedAt: now, UpdatedAt: now,
			}); err != nil {
				return reviewResult, fmt.Errorf("create ai advisory recommendation: %w", err)
			}
		}
	}
	return reviewResult, reviewErr
}

func (s *Service) failReview(ctx context.Context, rec store.AIAdvisoryReview, msg string) error {
	rec.Status = "failed"
	rec.ErrorMessage = msg
	now := time.Now().UTC()
	rec.FinishedAt = &now
	return s.store.UpdateAIAdvisoryReview(ctx, rec)
}

func newReviewID() (string, error) {
	buf := make([]byte, 8)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return "air-" + hex.EncodeToString(buf), nil
}

// SanitizeStoredResponse redacts secrets from stored JSON for display.
func SanitizeStoredResponse(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	out, _ := RedactText(string(raw), true)
	return out
}

// FormatScanType normalizes scan type strings.
func FormatScanType(trigger string) ScanType {
	switch strings.ToLower(strings.TrimSpace(trigger)) {
	case "preinstall", "preinstall_audit":
		return ScanTypePreinstall
	case "container", "container_image_scan":
		return ScanTypeContainer
	default:
		return ScanTypeRepo
	}
}
