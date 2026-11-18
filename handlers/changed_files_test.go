package handlers

import "testing"

func TestCollectChangedFiles(t *testing.T) {
	commits := []Commit{
		{
			Added:    []string{"src/a.go", "docs/readme.md"},
			Modified: []string{"src/b.go"},
			Removed:  []string{"old.txt"},
		},
		{
			Added:    []string{"src/a.go"},
			Modified: []string{"src/c.go"},
		},
	}

	got := CollectChangedFiles(commits)
	want := []string{"src/a.go", "docs/readme.md", "src/b.go", "src/c.go"}

	if len(got) != len(want) {
		t.Fatalf("expected %d paths, got %d: %v", len(want), len(got), got)
	}

	for i, path := range want {
		if got[i] != path {
			t.Fatalf("index %d: expected %q, got %q", i, path, got[i])
		}
	}
}

func TestCollectChangedFilesEmpty(t *testing.T) {
	if got := CollectChangedFiles(nil); len(got) != 0 {
		t.Fatalf("expected empty slice, got %v", got)
	}
}
