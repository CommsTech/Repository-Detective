package ui

import (
	"testing"

	"git.commsnet.org/commstech/repository-detective/store"
)

func TestEnrichCalibrationRecommendationViews(t *testing.T) {
	repoID := int64(7)
	recs := []store.CalibrationRecommendation{
		{ID: 1, Scope: "global", Category: "maintainability", Source: "graph", RuleID: "GRAPH-ORPHAN-FUNCTION", RecommendedAction: "report_only"},
		{ID: 2, Scope: "repo", RepositoryID: &repoID, Category: "quality", Source: "static", RuleID: "X", RecommendedAction: "report_only"},
		{ID: 3, Scope: "global", Category: "security", Source: "gosec", RuleID: "G304", RecommendedAction: "report_only"},
		{ID: 4, Scope: "global", Category: "secret", Source: "static", RuleID: "SEC-HARDCODED-SECRET", RecommendedAction: "report_only"},
	}
	views := enrichCalibrationRecommendationViews(recs)
	if !views[0].CanAccept || views[0].AcceptLabel != "Accept for affected repos" {
		t.Fatalf("global maintainability: %+v", views[0])
	}
	if !views[1].CanAccept || views[1].AcceptLabel != "Accept" {
		t.Fatalf("repo quality: %+v", views[1])
	}
	if views[2].CanAccept || views[2].BlockReason == "" {
		t.Fatalf("security should block accept: %+v", views[2])
	}
	if views[3].CanAccept || views[3].BlockReason == "" {
		t.Fatalf("secret should block accept: %+v", views[3])
	}
}
