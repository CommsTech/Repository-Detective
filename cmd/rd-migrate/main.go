// rd-migrate opens the configured SQLite database and applies pending schema migrations.
package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"git.commsnet.org/commstech/repository-detective/store"
)

func main() {
	path := os.Getenv("REPOSITORY_DETECTIVE_DATABASE_PATH")
	if path == "" {
		path = filepath.Join("data", "repository-detective.db")
	}
	s, err := store.Open(store.Config{Enabled: true, Path: path})
	if err != nil {
		fmt.Fprintf(os.Stderr, "migrate failed: %v\n", err)
		os.Exit(1)
	}
	defer s.Close()
	if _, err := s.LearningHealthSummary(context.Background()); err != nil {
		fmt.Fprintf(os.Stderr, "learning tables verify failed: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("database migrated: %s\n", path)
}
