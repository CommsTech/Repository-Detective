package issues_test

import (
	"context"
	"testing"

	"git.commsnet.org/commstech/repository-detective/issues"
)

func TestExtractFingerprintFromRepositoryDetectiveMarker(t *testing.T) {
	body := "## Tracking\n\n- Repository Detective fingerprint: rd-legacy99\n"
	if got := issues.ExtractFingerprintFromBody(body); got != "rd-legacy99" {
		t.Fatalf("got %q", got)
	}
}

func TestFindIssueByFingerprintPaginatesBeyondFirstPage(t *testing.T) {
	var pages int
	forge := &paginatingForge{pages: &pages}
	match, err := issues.FindIssueByFingerprint(context.Background(), forge, "o", "r", "rd-page2")
	if err != nil {
		t.Fatalf("find: %v", err)
	}
	if match == nil || match.IssueNumber != 200 {
		t.Fatalf("unexpected match: %+v", match)
	}
	if pages < 2 {
		t.Fatalf("expected pagination, pages=%d", pages)
	}
}

type paginatingForge struct {
	pages *int
}

func (p *paginatingForge) ListOpenLabeledIssues(_ context.Context, _, _ string, labels []string, limit, page int) ([]issues.ForgeIssue, error) {
	(*p.pages)++
	if page == 1 {
		out := make([]issues.ForgeIssue, limit)
		for i := range out {
			out[i] = issues.ForgeIssue{Number: i + 1, Body: "- Repository Detective fingerprint: rd-other\n"}
		}
		return out, nil
	}
	if page == 2 {
		return []issues.ForgeIssue{{
			Number: 200, HTMLURL: "http://x/200",
			Body: "- Repository Detective fingerprint: rd-page2\n",
		}}, nil
	}
	return nil, nil
}
func (p *paginatingForge) ListOpenIssues(_ context.Context, _, _ string, limit, page int) ([]issues.ForgeIssue, error) {
	return nil, nil
}
func (p *paginatingForge) CreateIssue(context.Context, string, string, string, string, []string) (*issues.ForgeIssue, error) {
	return nil, nil
}
func (p *paginatingForge) CreateIssueComment(context.Context, string, string, int, string) error {
	return nil
}
func (p *paginatingForge) AddIssueLabels(context.Context, string, string, int, []string) error {
	return nil
}
