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

func TestReachabilityDocsHighDowngrade(t *testing.T) {
	in := ClassifyPath("docs/guides/SETUP.md")
	sev, conf, note := ActionabilityAdjust("high", 0.95, in)
	if sev != "medium" || conf > 0.7 || note == "" {
		t.Fatalf("docs high should downgrade to medium: sev=%s conf=%v note=%q", sev, conf, note)
	}
	in = ClassifyPath("archive/session_summaries/NOTE.md")
	sev, _, note = ActionabilityAdjust("critical", 0.9, in)
	if sev != "medium" || note == "" {
		t.Fatalf("archive critical should downgrade: sev=%s note=%q", sev, note)
	}
}

func TestReachabilityExampleAndVendorPaths(t *testing.T) {
	in := ClassifyPath("collaboration-framework/config.yaml.example")
	sev, _, note := ActionabilityAdjust("high", 0.9, in)
	if sev != "medium" || note == "" {
		t.Fatalf("example path high should downgrade: sev=%s note=%q", sev, note)
	}
	in = ClassifyPath("vendor/pdf.js")
	if !in.VendorPath {
		t.Fatal("expected pdf.js vendor classification")
	}
	in = ClassifyPath("ansible/collections/ansible_collections/community/windows/plugins/lookup/laps_password.py")
	if !in.VendorPath {
		t.Fatal("expected ansible_collections vendor classification")
	}
}

