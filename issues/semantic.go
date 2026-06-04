package issues

import (
	"context"
	"fmt"

	"git.commsnet.org/commstech/bugbot/ai"
	"git.commsnet.org/commstech/bugbot/memory/qdrant"
	"github.com/sirupsen/logrus"
)

// SemanticStore connects Bugbot issue creation to an existing Qdrant deployment.
type SemanticStore struct {
	store    *qdrant.Store
	embedder *ai.Embedder
	logger   *logrus.Logger
}

// NewSemanticStore creates a semantic dedup helper.
func NewSemanticStore(store *qdrant.Store, embedder *ai.Embedder, logger *logrus.Logger) *SemanticStore {
	return &SemanticStore{store: store, embedder: embedder, logger: logger}
}

// Enabled reports whether semantic dedup is configured.
func (s *SemanticStore) Enabled() bool {
	return s != nil && s.store != nil && s.store.Enabled() && s.embedder != nil && s.embedder.Enabled()
}

// Prepare ensures the remote collection exists.
func (s *SemanticStore) Prepare(ctx context.Context) error {
	if !s.Enabled() {
		return nil
	}
	return s.store.Prepare(ctx)
}

// DuplicateMatch describes a prior similar finding in Qdrant.
type DuplicateMatch struct {
	Score       float64
	IssueURL    string
	IssueNumber int
	ClusterID   string
	Title       string
}

// FindDuplicate checks Qdrant for a semantically similar prior finding.
func (s *SemanticStore) FindDuplicate(ctx context.Context, repository string, issue *ai.CodeIssue) (*DuplicateMatch, error) {
	if !s.Enabled() {
		return nil, nil
	}

	input := qdrant.MatchInput{
		Repository:  repository,
		Title:       issue.Title,
		Description: issue.Description,
		File:        issue.File,
		Line:        issue.LineNumber,
		Severity:    issue.Severity,
		Category:    issue.Category,
		Source:      issue.Source,
		RuleID:      issue.RuleID,
		Confidence:  issue.Confidence,
		Fingerprint: issue.Fingerprint,
	}

	vector, err := s.embedder.Embed(ctx, qdrant.EmbeddingText(input))
	if err != nil {
		return nil, err
	}

	match, err := s.store.FindSimilar(ctx, input, vector)
	if err != nil {
		return nil, err
	}
	if match == nil {
		return nil, nil
	}

	return &DuplicateMatch{
		Score:       match.Score,
		IssueURL:    match.IssueURL,
		IssueNumber: match.IssueNumber,
		ClusterID:   match.ClusterID,
		Title:       match.Title,
	}, nil
}

// Remember stores a finding vector after issue creation.
func (s *SemanticStore) Remember(ctx context.Context, repository string, issue *ai.CodeIssue, issueURL string, issueNumber int, clusterID string) error {
	if !s.Enabled() {
		return nil
	}

	input := qdrant.MatchInput{
		Repository:  repository,
		Title:       issue.Title,
		Description: issue.Description,
		File:        issue.File,
		Line:        issue.LineNumber,
		Severity:    issue.Severity,
		Category:    issue.Category,
		Confidence:  issue.Confidence,
		ClusterID:   clusterID,
		Fingerprint: issue.Fingerprint,
		Source:      issue.Source,
		RuleID:      issue.RuleID,
	}

	vector, err := s.embedder.Embed(ctx, qdrant.EmbeddingText(input))
	if err != nil {
		return err
	}
	return s.store.Remember(ctx, input, issueURL, issueNumber, vector)
}

// DuplicateCommentBody formats a comment when updating an existing issue.
func DuplicateCommentBody(issue *ai.CodeIssue, score float64) string {
	snippet := SanitizeSecretEvidence(issue.CodeSnippet)
	body := fmt.Sprintf(
		"Repository Detective detected a semantically similar finding (score %.2f).\n\n**%s**\n\nSeverity: %s\nCategory: %s\nConfidence: %.2f\nSource: %s\nFingerprint: %s\nFile: `%s` line %d\n\n%s",
		score,
		issue.Title,
		issue.Severity,
		issue.Category,
		issue.Confidence,
		issue.Source,
		issue.Fingerprint,
		issue.File,
		issue.LineNumber,
		issue.Description,
	)
	if snippet != "" {
		body += "\n\nEvidence:\n```\n" + snippet + "\n```"
	}
	return body
}
