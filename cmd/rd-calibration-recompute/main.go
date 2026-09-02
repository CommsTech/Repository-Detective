// rd-calibration-recompute backfills learning events and generates calibration recommendations.
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"git.commsnet.org/commstech/repository-detective/store"
)

func main() {
	repoID := flag.Int64("repo", 0, "optional repository id to scope repo recommendations")
	limit := flag.Int( "backfill-limit", 10000, "max findings to backfill per run")
	flag.Parse()

	path := os.Getenv("REPOSITORY_DETECTIVE_DATABASE_PATH")
	if path == "" {
		path = filepath.Join("data", "repository-detective.db")
	}
	s, err := store.Open(store.Config{Enabled: true, Path: path})
	if err != nil {
		fmt.Fprintf(os.Stderr, "open: %v\n", err)
		os.Exit(1)
	}
	defer s.Close()
	ctx := context.Background()

	backfilled, err := s.BackfillFalsePositiveLearningEvents(ctx, *limit)
	if err != nil {
		fmt.Fprintf(os.Stderr, "backfill: %v\n", err)
		os.Exit(1)
	}
	stats, _ := s.RecomputeCalibrationRuleStats(ctx)
	global, _ := s.GenerateCalibrationRecommendations(ctx, 5)
	repos, _ := s.ListRepositoriesWithSummary(ctx, store.ListOptions{Limit: 100})
	repoRecs := 0
	for _, r := range repos {
		if *repoID > 0 && r.ID != *repoID {
			continue
		}
		n, _ := s.GenerateRepoScopedRecommendations(ctx, r.ID, 5)
		repoRecs += n
	}
	recs, _ := s.ListCalibrationRecommendations(ctx, "proposed", 50)
	fmt.Printf("backfilled=%d stats=%d global=%d repo_recs=%d proposed=%d\n", backfilled, stats, global, repoRecs, len(recs))
	for _, rec := range recs {
		fmt.Printf("rec id=%d scope=%s repo=%v %s/%s action=%s conf=%.2f\n",
			rec.ID, rec.Scope, rec.RepositoryID, rec.Source, rec.RuleID, rec.RecommendedAction, rec.Confidence)
	}
}
