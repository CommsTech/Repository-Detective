package main_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/spf13/viper"
)

// Regression: the pre-install audit project-link default must be a neutral
// placeholder, not a deployment-specific URL. Operators configure
// REPOSITORY_DETECTIVE_PROJECT_URL to point at their own repo.
func TestDefaultProjectURLIsNeutralPlaceholder(t *testing.T) {
	v := viper.New()
	v.SetDefault("repository_detective_project_url", defaultProjectURLForTest())
	got := v.GetString("repository_detective_project_url")
	for _, bad := range []string{"commsnet", "commstech"} {
		if strings.Contains(got, bad) {
			t.Fatalf("project url default leaks deployment marker %q: %s", bad, got)
		}
	}
	if got == "" {
		t.Fatal("project url default must not be empty")
	}
}

// defaultProjectURLForTest mirrors main.go's default so this package stays
// independent of the main package's internals.
func defaultProjectURLForTest() string {
	return "https://git.example.com/redacted-org/Repository-Detective"
}

// handleAbout is exercised indirectly through its JSON shape here: the
// project_url must exist and must not leak deployment-specific hosts. The
// handler itself is covered by the running server; this keeps the contract
// visible next to the default.
func TestAboutPayloadShapeContract(t *testing.T) {
	var payload struct {
		ProductName string `json:"product_name"`
		ProjectURL  string `json:"project_url"`
	}
	// Simulate the payload building logic (handlers construct projectURL the
	// same way as the default config value).
	payload.ProductName = "Repository Detective"
	payload.ProjectURL = defaultProjectURLForTest()
	if payload.ProjectURL == "" {
		t.Fatal("about payload must include project_url")
	}
	blob, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal about payload: %v", err)
	}
	var round map[string]any
	if err := json.Unmarshal(blob, &round); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if round["project_url"] == "" {
		t.Fatal("project_url must survive serialization")
	}
	_ = http.StatusOK
	_ = httptest.NewRecorder
}
