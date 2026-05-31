package operator

import "testing"

func TestCheckToolsRespectsConfig(t *testing.T) {
	tools := CheckTools(ScannerConfig{EnableTrivy: true, EnableGrype: false})
	byName := map[string]ToolStatus{}
	for _, tool := range tools {
		byName[tool.Name] = tool
	}
	if !byName["git"].Configured {
		t.Fatal("git should always be configured")
	}
	if !byName["trivy"].Configured {
		t.Fatal("trivy should be configured when enabled")
	}
	if byName["grype"].Configured {
		t.Fatal("grype should not be configured when disabled")
	}
	for _, tool := range tools {
		if tool.LastChecked == "" {
			t.Fatalf("tool %s missing last_checked", tool.Name)
		}
	}
}
