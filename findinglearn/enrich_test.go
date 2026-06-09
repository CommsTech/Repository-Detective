package findinglearn

import "testing"

func TestStructuralHashSamePattern(t *testing.T) {
	a := StructuralHash("SEC-EVAL", "security", `eval(userInput)`)
	b := StructuralHash("SEC-EVAL", "security", `eval(otherVar)`)
	if a != b {
		t.Fatalf("expected same hash for same shape")
	}
}

func TestStructuralHashDifferentRule(t *testing.T) {
	a := StructuralHash("SEC-EVAL", "security", `eval(x)`)
	b := StructuralHash("SEC-XSS", "security", `eval(x)`)
	if a == b {
		t.Fatal("different rules should not merge")
	}
}

func TestReachabilityTestPathDowngrade(t *testing.T) {
	in := ClassifyPath("pkg/foo_test.go")
	sev, conf, note := ActionabilityAdjust("medium", 0.8, in)
	if sev != "info" || note == "" {
		t.Fatalf("got %s conf=%v note=%q", sev, conf, note)
	}
}

func TestReachabilityTestdataFixtureDowngrade(t *testing.T) {
	in := ClassifyPath("testdata/fixtures/go-single/main.go")
	sev, _, note := ActionabilityAdjust("medium", 0.8, in)
	if sev != "info" || note == "" {
		t.Fatalf("testdata should downgrade: sev=%s note=%q", sev, note)
	}
}
