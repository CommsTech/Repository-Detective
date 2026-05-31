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

// Prepare ensures the target collection exists.
func (s *Store) Prepare(ctx context.Context) error {
	if !s.Enabled() {
		return nil
	}
	return s.client.EnsureCollection(ctx)
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

	hits, err := s.client.SearchSimilar(ctx, input.Repository, vector, 5)
	if err != nil {
		return nil, err
	}

	threshold := s.client.cfg.SimilarityThreshold
	if threshold <= 0 {
		threshold = 0.85
	}

	for _, hit := range hits {
		if hit.Score < threshold {
			continue
		}
		return &MatchResult{
			Score:       hit.Score,
			IssueURL:    hit.Payload.IssueURL,
			IssueNumber: hit.Payload.IssueNumber,
			ClusterID:   hit.Payload.ClusterID,
			Title:       hit.Payload.Title,
		}, nil
	}
	return nil, nil
}

// Remember stores a finding vector for future dedup checks.
func (s *Store) Remember(ctx context.Context, input MatchInput, issueURL string, issueNumber int, vector []float32) error {
	if !s.Enabled() {
		return nil
	}

	payload := FindingPayload{
		Repository:  input.Repository,
		Title:       input.Title,
		Description: input.Description,
		File:        input.File,
		Line:        input.Line,
		Severity:    input.Severity,
		Category:    input.Category,
		Confidence:  input.Confidence,
		IssueURL:    issueURL,
		IssueNumber: issueNumber,
		ClusterID:   input.ClusterID,
		Fingerprint: fingerprintValue(input),
	}

	return s.client.Upsert(ctx, payload.Fingerprint, vector, payload)
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
	}, "|")
	sum := sha256.Sum256([]byte(key))
	return hex.EncodeToString(sum[:16])
}

// EmbeddingText builds the text embedded for semantic comparison.
func EmbeddingText(input MatchInput) string {
	return strings.TrimSpace(strings.Join([]string{
		input.Title,
		input.Description,
		input.Category,
		input.File,
	}, "\n"))
}
