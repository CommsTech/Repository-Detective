package learning_test

import (
	"testing"

	"git.commsnet.org/commstech/repository-detective/learning"
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
	if learning.IsProtectedFromAutoDowngrade("", "maintainability") {
		t.Fatal("empty severity + non-security category should allow calibration")
	}
	if learning.IsProtectedFromAutoDowngrade("", "") {
		t.Fatal("empty inputs should allow calibration")
	}
}

func TestValidateCalibrationAccept(t *testing.T) {
	if err := learning.ValidateCalibrationAccept("maintainability", "repo"); err != nil {
		t.Fatalf("repo quality accept should be allowed: %v", err)
	}
	if err := learning.ValidateCalibrationAccept("", "repo"); err != nil {
		t.Fatalf("empty category repo accept should be allowed: %v", err)
	}
	if err := learning.ValidateCalibrationAccept("hardcoded_secret", "repo"); err == nil {
		t.Fatal("secret category accept should be blocked")
	}
	if err := learning.ValidateCalibrationAccept("quality", "global"); err != nil {
		t.Fatalf("global accept should be allowed (expands to repo-scoped rules): %v", err)
	}
	if err := learning.ValidateCalibrationAccept("security", "global"); err == nil {
		t.Fatal("protected global category should still be blocked")
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
