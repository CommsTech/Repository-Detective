package learning_test

import (
	"testing"

	"git.commsnet.org/commstech/bugbot/learning"
)

func TestStructuralHashSamePattern(t *testing.T) {
	a := learning.StructuralHash("SEC-EVAL", "security", `eval(userInput)`)
	b := learning.StructuralHash("SEC-EVAL", "security", `eval(otherVar)`)
	if a != b {
		t.Fatalf("expected same hash for same shape")
	}
}

func TestStructuralHashDifferentRule(t *testing.T) {
	a := learning.StructuralHash("SEC-EVAL", "security", `eval(x)`)
	b := learning.StructuralHash("SEC-XSS", "security", `eval(x)`)
	if a == b {
		t.Fatal("different rules should not merge")
	}
}

func TestProtectedSecurityNotAutoDowngrade(t *testing.T) {
	if !learning.IsProtectedFromAutoDowngrade("high", "quality") {
		t.Fatal("high severity protected")
	}
	if !learning.IsProtectedFromAutoDowngrade("medium", "hardcoded_secret") {
		t.Fatal("secret category protected")
	}
}

func TestReachabilityTestPathDowngrade(t *testing.T) {
	in := learning.ClassifyPath("pkg/foo_test.go")
	in.FromEntrypoint = false
	sev, conf, note := learning.ActionabilityAdjust("medium", 0.8, in)
	if sev != "info" || note == "" {
		t.Fatalf("got %s conf=%v note=%q", sev, conf, note)
	}
}

func TestSanityGateDisabledByDefault(t *testing.T) {
	g := learning.NewSanityGate(learning.DefaultSanityConfig())
	if g.ShouldEvaluate("low", "quality") {
		t.Fatal("disabled gate should not evaluate")
	}
}
