package ui

import "testing"

func TestAPIKeyQuerySuffix(t *testing.T) {
	if got := apiKeyQuerySuffix("", "key"); got != "?api_key=key" {
		t.Fatalf("empty path: got %q", got)
	}
	if got := apiKeyQuerySuffix("/findings?severity=critical", "key"); got != "&api_key=key" {
		t.Fatalf("existing query: got %q", got)
	}
}
