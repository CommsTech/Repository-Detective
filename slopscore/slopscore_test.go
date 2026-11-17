package slopscore

import (
	"context"
	"strings"
	"testing"
)

func TestCleanFileScoresPerfect(t *testing.T) {
	m, f, err := Analyze(context.Background(), []File{{Path: "a.go", Content: "package a\n\nfunc Add(x, y int) int {\n\treturn x + y\n}\n"}})
	if err != nil {
		t.Fatal(err)
	}
	if len(f) != 0 || m.Score != 100 || m.Grade != "A" {
		t.Fatalf("clean file scored %+v findings=%v", m, f)
	}
	if m.LinesScanned != 5 {
		t.Fatalf("line count = %d", m.LinesScanned)
	}
}

func TestDetectsEachTell(t *testing.T) {
	files := []File{
		{Path: "a.go", Content: "// TODO: finish this\n// return nil\n"},
		{Path: "b_test.go", Content: "expect(true).toBe(true)\n"},
		{Path: "c.py", Content: "try:\n    work()\nexcept:\n    pass\n"},
		{Path: "d.js", Content: "console.log(\"x\")\n"},
		{Path: "e.go", Content: "// ship it 🚀\n"},
		{Path: "f.md", Content: "A seamless, world-class experience.\n"},
	}
	m, findings, err := Analyze(context.Background(), files)
	if err != nil {
		t.Fatal(err)
	}
	seen := map[string]bool{}
	for _, f := range findings {
		seen[f.ID] = true
	}
	for _, want := range []string{"SLOP-TODO", "SLOP-COMMENTED-CODE", "SLOP-TAUTOLOGY", "SLOP-SWALLOW", "SLOP-DEBUG", "SLOP-EMOJI", "SLOP-MARKETING"} {
		if !seen[want] {
			t.Errorf("missing %s in %v", want, seen)
		}
	}
	if m.Score >= 90 || m.Grade == "A" {
		t.Fatalf("slop-filled input scored too well: %+v", m)
	}
}

func TestScoreIsDensityNormalised(t *testing.T) {
	small := []File{{Path: "a.go", Content: "// TODO one\n"}}
	large := []File{{Path: "a.go", Content: "// TODO one\n" + strings.Repeat("var x = 1\n", 2000)}}
	smallM, _, err := Analyze(context.Background(), small)
	if err != nil {
		t.Fatal(err)
	}
	largeM, _, err := Analyze(context.Background(), large)
	if err != nil {
		t.Fatal(err)
	}
	if largeM.Score <= smallM.Score {
		t.Fatalf("density normalisation failed: small=%d large=%d", smallM.Score, largeM.Score)
	}
}

func TestBudgetsFailClosed(t *testing.T) {
	if _, _, err := Analyze(context.Background(), []File{{Path: "big.go", Content: strings.Repeat("a", MaxFileBytes+1)}}); err == nil {
		t.Fatal("oversized file must be an explicit error")
	}
	many := make([]File, MaxFiles+1)
	for i := range many {
		many[i] = File{Path: "a.go", Content: "x"}
	}
	if _, _, err := Analyze(context.Background(), many); err == nil {
		t.Fatal("file budget must be an explicit error")
	}
}

func TestCanceled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, _, err := Analyze(ctx, []File{{Path: "a.go", Content: "x"}}); err == nil {
		t.Fatal("cancellation ignored")
	}
}

func TestCommentDoesNotTriggerTodoInsideWord(t *testing.T) {
	_, f, err := Analyze(context.Background(), []File{{Path: "a.go", Content: "var TODOS = 1\n"}})
	if err != nil {
		t.Fatal(err)
	}
	if len(f) != 0 {
		t.Fatalf("identifier containing TODO matched: %+v", f)
	}
}
