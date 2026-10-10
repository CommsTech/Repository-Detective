package scanners

import "testing"

func TestStableLinterRuleIDAggregatesByRuleNotLine(t *testing.T) {
	got := stableLinterRuleID("LINT-GO", "typecheck")
	want := "LINT-GO-typecheck"
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
	if stableLinterRuleID("LINT-RUFF", "F401") != "LINT-RUFF-F401" {
		t.Fatal("expected stable ruff rule id")
	}
	if stableLinterRuleID("LINT-SHELL", "2034") != "LINT-SHELL-2034" {
		t.Fatal("expected stable shellcheck rule id")
	}
}
