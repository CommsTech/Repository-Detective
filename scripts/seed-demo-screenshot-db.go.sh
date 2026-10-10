#!/usr/bin/env bash
# Seed a disposable local SQLite DB with synthetic demo data for doc screenshots.
# Does NOT touch production data/repository-detective.db or port 8081.
set -euo pipefail
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
OUT_DB="${RD_DEMO_DB:-/tmp/rd-demo-screenshots/repository-detective.db}"
mkdir -p "$(dirname "$OUT_DB")"
rm -f "$OUT_DB"

export PATH="${GOBIN:-$HOME/go/bin}:$PATH"
cd "$ROOT"

cat > /tmp/rd-seed-demo-screenshots.go <<'EOF'
package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"git.commsnet.org/commstech/repository-detective/store"
)

func main() {
	path := os.Getenv("RD_DEMO_DB")
	if path == "" {
		path = "/tmp/rd-demo-screenshots/repository-detective.db"
	}
	s, err := store.OpenSQLite(path)
	if err != nil {
		panic(err)
	}
	defer s.Close()
	ctx := context.Background()
	now := time.Now().UTC()

	repo, err := s.UpsertRepository(ctx, store.Repository{
		Owner:         "demo",
		Name:          "repository-detective-test",
		FullName:      "demo/repository-detective-test",
		ForgeType:     store.ForgeTypeGitea,
		CloneURL:      "http://127.0.0.1:13000/demo/repository-detective-test.git",
		DefaultBranch: "main",
		ConnectedRepo: true,
		ScanEnabled:   true,
		PolicyLevel:   "monitor_only",
		ScanProfile:   "standard_deterministic",
	})
	if err != nil {
		panic(err)
	}

	f, err := s.UpsertFinding(ctx, store.Finding{
		RepositoryID: repo.ID,
		Fingerprint:  "demo-fixture-slack-bot-token-shape",
		Title:        "Synthetic demo fixture: Slack-bot-shaped token pattern",
		Severity:     "high",
		Category:     "secret",
		Source:       "gitleaks",
		RuleID:       "slack-bot-token",
		Status:       store.FindingStatusOpen,
		FilePath:     "fixtures/demo_token.txt",
		StartLine:    1,
		EndLine:      1,
		Message:      "Harmless synthetic fixture for documentation screenshots — not a real credential.",
		FirstSeenAt:  now,
		LastSeenAt:   now,
	})
	if err != nil {
		panic(err)
	}
	_ = s.AddFindingEvidence(ctx, f.ID, store.FindingEvidence{
		Kind:      "snippet",
		Locator:   "fixtures/demo_token.txt:1",
		Excerpt:   "xoxb-DEMO-NOT-A-REAL-TOKEN-000000000000-aaaaaaaaaaaaaaaaaaaaaaaa",
		CreatedAt: now,
	})

	fmt.Printf("seeded db=%s repo_id=%d finding_id=%d\n", path, repo.ID, f.ID)
}
EOF

RD_DEMO_DB="$OUT_DB" go run /tmp/rd-seed-demo-screenshots.go
echo "OK: $OUT_DB"
