package qdrant

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
)

// Store coordinates semantic dedup against an existing Qdrant deployment.
type Store struct {
	client *Client
}

// NewStore creates a semantic dedup store.
func NewStore(cfg Config) *Store {
	return &Store{client: NewClient(cfg)}
}

// Enabled reports whether Qdrant dedup is active.
func (s *Store) Enabled() bool {
	return s != nil && s.client != nil && s.client.Enabled()
}

// Prepare ensures the target collection exists and matches configured vector size.
func (s *Store) Prepare(ctx context.Context) error {
	if !s.Enabled() {
		return nil
	}
	if err := s.client.EnsureCollection(ctx); err != nil {
		return err
	}
	return s.client.ValidateCollection(ctx)
}

// MatchInput describes a finding being checked or stored.
type MatchInput struct {
	Repository  string
	Title       string
	Description string
	File        string
	Line        int
	Severity    string
	Category    string
	Source      string
	RuleID      string
	Verdict     string
	Confidence  float64
	ClusterID   string
	Fingerprint string
}

// MatchResult describes a prior similar finding.
type MatchResult struct {
	Score       float64
	IssueURL    string
	IssueNumber int
	ClusterID   string
	Title       string
}

// FindSimilar returns the best prior match above the configured threshold.
func (s *Store) FindSimilar(ctx context.Context, input MatchInput, vector []float32) (*MatchResult, error) {
	if !s.Enabled() {
		return nil, nil
	}
	if err := s.client.ValidateVectorLen(len(vector)); err != nil {
		return nil, err
	}

	hits, err := s.client.SearchSimilar(ctx, input.Repository, vector, 5)
	if err != nil {
		return nil, err
	}

	threshold := s.client.cfg.SimilarityThreshold
	if threshold <= 0 {
		threshold = 0.7
	}

	for _, hit := range hits {
		if hit.Score < threshold {
			continue
		}
		parsed := ParseHitPayload(hit.Payload)
		return &MatchResult{
			Score:       hit.Score,
			IssueURL:    parsed.IssueURL,
			IssueNumber: parsed.IssueNumber,
			ClusterID:   parsed.ClusterID,
			Title:       parsed.Title,
		}, nil
	}
	return nil, nil
}

// Remember stores a redacted finding vector for future dedup checks.
func (s *Store) Remember(ctx context.Context, input MatchInput, issueURL string, issueNumber int, vector []float32) error {
	if !s.Enabled() {
		return nil
	}
	if err := s.client.ValidateVectorLen(len(vector)); err != nil {
		return err
	}

	safeInput := input
	safeInput.Title = RedactSummary(input.Title, input.Description, "")
	safeInput.Description = ""

	payload := BuildCAHPayload(safeInput, issueURL, issueNumber, "")
	return s.client.Upsert(ctx, payload.ID, vector, payload)
}

func fingerprintValue(input MatchInput) string {
	if strings.TrimSpace(input.Fingerprint) != "" {
		return strings.TrimSpace(input.Fingerprint)
	}
	return fingerprint(input)
}

func fingerprint(input MatchInput) string {
	key := strings.Join([]string{
		input.Repository,
		input.Title,
		input.Description,
		input.File,
		fmt.Sprintf("%d", input.Line),
		input.Category,
		input.Source,
		input.RuleID,
	}, "|")
	sum := sha256.Sum256([]byte(key))
	return hex.EncodeToString(sum[:16])
}
