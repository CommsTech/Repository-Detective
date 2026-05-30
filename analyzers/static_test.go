package analyzers

import (
	"testing"
)

func TestRunStaticAnalysisFindsHardcodedSecret(t *testing.T) {
	findings := RunStaticAnalysis([]FileContent{{
		Path:    "config.go",
		Content: `api_key := "super-secret-token-12345"`,
	}}, true, false)

	if len(findings) == 0 {
		t.Fatal("expected static finding")
	}
	if findings[0].AuditorType != "static" {
		t.Fatalf("expected static source, got %s", findings[0].AuditorType)
	}
}

func TestRunStaticAnalysisQualityDisabled(t *testing.T) {
	findings := RunStaticAnalysis([]FileContent{{
		Path:    "app.js",
		Content: "console.log('debug');",
	}}, true, false)

	if len(findings) != 0 {
		t.Fatalf("expected no quality findings when disabled, got %d", len(findings))
	}
}
