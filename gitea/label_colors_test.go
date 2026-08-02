package gitea

import "testing"

func TestDefaultLabelColorSeverity(t *testing.T) {
	cases := map[string]string{
		"severity/critical": "dc2626",
		"severity/high":     "ea580c",
		"severity/medium":   "f59e0b",
		"severity/low":      "3b82f6",
	}
	for name, want := range cases {
		if got := DefaultLabelColor(name); got != want {
			t.Fatalf("DefaultLabelColor(%q) = %q, want %q", name, got, want)
		}
	}
}

func TestDefaultLabelColorCategory(t *testing.T) {
	if got := DefaultLabelColor("repository-detective/security"); got != "991b1b" {
		t.Fatalf("security color = %q", got)
	}
	if got := DefaultLabelColor("repository-detective/open"); got != "0ea5a4" {
		t.Fatalf("open color = %q", got)
	}
}
