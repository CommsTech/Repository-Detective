// rd-calibration-recompute generates repo-scoped calibration recommendations.
package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"git.commsnet.org/commstech/bugbot/store"
)

func main() {
	path := os.Getenv("REPOSITORY_DETECTIVE_DATABASE_PATH")
	if path == "" {
		path = filepath.Join("data", "bugbot.db")
	}
	s, err := store.Open(store.Config{Enabled: true, Path: path})
	if err != nil {
		fmt.Fprintf(os.Stderr, "open: %v\n", err)
		os.Exit(1)
	}
	defer s.Close()
	ctx := context.Background()
	stats, _ := s.RecomputeCalibrationRuleStats(ctx)
	global, _ := s.GenerateCalibrationRecommendations(ctx, 5)
	qs := s.(store.QueryStore)
	repos, _ := qs.ListRepositoriesWithSummary(ctx, store.ListOptions{Limit: 100})
	repoRecs := 0
	for _, r := range repos {
		if r.FullName != "commstech/netmapper" && r.FullName != "commstech/commsnet_optimizer" && r.FullName != "commstech/nextcloud_scripts" {
			continue
		}
		n, _ := qs.GenerateRepoScopedRecommendations(ctx, r.ID, 3)
		repoRecs += n
	}
	recs, _ := s.ListCalibrationRecommendations(ctx, "proposed", 50)
	fmt.Printf("stats=%d global=%d repo_recs=%d proposed=%d\n", stats, global, repoRecs, len(recs))
	for _, rec := range recs {
		fmt.Printf("rec id=%d scope=%s repo=%v %s/%s action=%s conf=%.2f\n",
			rec.ID, rec.Scope, rec.RepositoryID, rec.Source, rec.RuleID, rec.RecommendedAction, rec.Confidence)
	}
}
