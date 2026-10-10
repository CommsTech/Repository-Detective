package doctor_test

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"git.commsnet.org/commstech/repository-detective/doctor"
	"git.commsnet.org/commstech/repository-detective/internal/privacy"
	"git.commsnet.org/commstech/repository-detective/store"
)

func baseHealthyInput() doctor.Input {
	return doctor.Input{
		Version:                 "test",
		Commit:                  "abc",
		Edition:                 "community",
		DatabaseEnabled:         true,
		DatabaseOK:              true,
		SchemaVersion:           24,
		WorkspaceDir:            "",
		WorkspaceOK:             true,
		ConfigValid:             true,
		AuthMode:                "api_key_only",
		APIKeyConfigured:        true,
		RejectQueryStringAPIKey: true,
		PrivacyMode:             privacy.ModeHybrid,
		ForgeURL:                "https://git.example.com",
		ForgeTokenSet:           true,
		ForgeReachable:          true,
		ForgeAuthOK:             true,
		ForgeVersion:            "1.22",
		SkipLiveForgeProbe:      false,
		PublicURL:               "https://rd.example.com",
		WebhookSecretSet:        true,
		ScanProfile:             store.ScanProfileLight,
		PolicyMode:              "Observe",
		EnforcementMode:         store.EnforcementObserve,
		RequiredScanners:        []string{"gitleaks", "trivy"},
		ScannerTools: []doctor.ScannerToolInput{
			{Name: "gitleaks", Role: store.ScannerRoleRequired, EnabledInConfig: true, Available: true, Version: "8.0"},
			{Name: "trivy", Role: store.ScannerRoleRequired, EnabledInConfig: true, Available: true, Version: "0.50"},
			{Name: "semgrep", Role: store.ScannerRoleOptional, EnabledInConfig: false, Available: false},
		},
		ClassBIsolation: "NOT_PROVEN",
	}
}

func TestHealthyDeterministicOnly(t *testing.T) {
	in := baseHealthyInput()
	in.AIEnabled = false
	r := doctor.Run(in)
	if r.Overall != doctor.OverallHealthy && r.Overall != doctor.OverallDegraded {
		// webhook delivery NOT_PROVEN is optional warning → may be DEGRADED
		t.Fatalf("overall=%s summary=%s", r.Overall, r.Summary)
	}
	if r.RequiredFailed > 0 {
		t.Fatalf("required failures: %+v", r.Checks)
	}
	foundAI := false
	for _, c := range r.Checks {
		if c.ID == "ai.status" {
			foundAI = true
			if c.State != doctor.StatePass || !strings.Contains(c.Summary, "DISABLED") {
				t.Fatalf("AI disabled check: %+v", c)
			}
		}
	}
	if !foundAI {
		t.Fatal("missing ai.status")
	}
}

func TestLocalOnlyExternalAIRejected(t *testing.T) {
	in := baseHealthyInput()
	in.PrivacyMode = privacy.ModeLocalOnly
	in.AIEnabled = true
	in.AIProvider = "openai"
	in.AIBaseURL = "https://api.openai.com/v1"
	in.AILocality = privacy.ClassExternal
	in.AIEgressAllowed = false
	in.AIEgressReason = "LOCAL_ONLY rejects cloud AI providers"
	r := doctor.Run(in)
	if r.Overall != doctor.OverallNotReady {
		t.Fatalf("want NOT_READY got %s", r.Overall)
	}
}

func TestLocalOnlyLocalOllama(t *testing.T) {
	in := baseHealthyInput()
	in.PrivacyMode = privacy.ModeLocalOnly
	in.AIEnabled = true
	in.AIProvider = "ollama"
	in.AIBaseURL = "http://10.0.0.5:11434"
	in.AILocality = privacy.ClassLocal
	in.AIEgressAllowed = true
	in.ForgeLocality = privacy.ClassLocal
	r := doctor.Run(in)
	if r.RequiredFailed > 0 {
		t.Fatalf("unexpected required failures: %s", r.Summary)
	}
}

func TestLocalOnlyExternalForgeDisclosure(t *testing.T) {
	in := baseHealthyInput()
	in.PrivacyMode = privacy.ModeLocalOnly
	in.ForgeLocality = privacy.ClassExternal
	r := doctor.Run(in)
	found := false
	for _, c := range r.Checks {
		if c.ID == "privacy.forge_egress" {
			found = true
			if c.State != doctor.StateWarning {
				t.Fatalf("%+v", c)
			}
			if strings.Contains(strings.ToLower(c.Detail), "no data leaves") {
				t.Fatal("must not claim no data leaves network")
			}
		}
	}
	if !found {
		t.Fatal("expected forge egress disclosure")
	}
}

func TestRequiredScannerMissing(t *testing.T) {
	in := baseHealthyInput()
	in.ScannerTools = []doctor.ScannerToolInput{
		{Name: "gitleaks", Role: store.ScannerRoleRequired, EnabledInConfig: true, Available: false},
		{Name: "trivy", Role: store.ScannerRoleRequired, EnabledInConfig: true, Available: true, Version: "0.50"},
	}
	r := doctor.Run(in)
	if r.Overall != doctor.OverallNotReady {
		t.Fatalf("want NOT_READY got %s", r.Overall)
	}
}

func TestOptionalScannerMissing(t *testing.T) {
	in := baseHealthyInput()
	in.ScannerTools = append(in.ScannerTools, doctor.ScannerToolInput{
		Name: "checkov", Role: store.ScannerRoleOptional, EnabledInConfig: true, Available: false,
	})
	r := doctor.Run(in)
	// optional missing should not alone force NOT_READY if required OK
	if r.RequiredFailed > 0 {
		t.Fatalf("optional missing should not be required failure")
	}
}

func TestForgeAuthFailure(t *testing.T) {
	in := baseHealthyInput()
	in.ForgeAuthOK = false
	in.ForgeReachable = true
	in.ForgeAuthDetail = "401 unauthorized"
	r := doctor.Run(in)
	if r.Overall != doctor.OverallNotReady {
		t.Fatalf("want NOT_READY got %s", r.Overall)
	}
}

func TestDBUnavailable(t *testing.T) {
	in := baseHealthyInput()
	in.DatabaseOK = false
	in.DatabaseDetail = "connection refused"
	r := doctor.Run(in)
	if r.Overall != doctor.OverallNotReady {
		t.Fatalf("want NOT_READY got %s", r.Overall)
	}
}

func TestNoSecretsInDoctorOutput(t *testing.T) {
	secret := "super-secret-token-value-xyz"
	in := baseHealthyInput()
	in.ForgeAuthDetail = "token=" + secret
	in.ConfigDetail = "api_key=\"" + secret + "\""
	r := doctor.Run(in)
	var buf bytes.Buffer
	if err := doctor.FormatHuman(&buf, r); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(buf.String(), secret) {
		t.Fatal("secret leaked in human output")
	}
	buf.Reset()
	if err := doctor.FormatJSON(&buf, r); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(buf.String(), secret) {
		t.Fatal("secret leaked in JSON")
	}
	bundle := doctor.BuildSupportBundle(r, map[string]string{
		"gitea_token": secret,
		"gitea_url":   "https://git.example.com",
	}, []string{"auth failed token=" + secret})
	raw, _ := json.Marshal(bundle)
	if strings.Contains(string(raw), secret) {
		t.Fatal("secret leaked in support bundle")
	}
	if bundle.SanitizedConfig["gitea_token"] != "[REDACTED]" {
		t.Fatalf("token not redacted: %q", bundle.SanitizedConfig["gitea_token"])
	}
}

func TestJSONSchemaStability(t *testing.T) {
	r := doctor.Run(baseHealthyInput())
	raw, err := json.Marshal(doctor.RedactReport(r))
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"overall", "checks", "generated_at", "summary"} {
		if _, ok := m[key]; !ok {
			t.Fatalf("missing key %s", key)
		}
	}
	checks, _ := m["checks"].([]any)
	if len(checks) == 0 {
		t.Fatal("empty checks")
	}
	first := checks[0].(map[string]any)
	for _, key := range []string{"id", "category", "state", "summary"} {
		if _, ok := first[key]; !ok {
			t.Fatalf("check missing %s", key)
		}
	}
}

func TestRecommendProfile(t *testing.T) {
	p, _ := doctor.RecommendProfile("Go", nil)
	if p != "standard" {
		t.Fatalf("go -> %s", p)
	}
	p, _ = doctor.RecommendProfile("", []string{"main.tf", "modules/net.tf"})
	if p != "standard" {
		t.Fatalf("terraform -> %s", p)
	}
	p, _ = doctor.RecommendProfile("", []string{"README.md", "docs/guide.md"})
	if p != "light" {
		t.Fatalf("docs -> %s", p)
	}
}

func TestQueryAPIKeyWarning(t *testing.T) {
	in := baseHealthyInput()
	in.RejectQueryStringAPIKey = false
	r := doctor.Run(in)
	found := false
	for _, c := range r.Checks {
		if c.ID == "auth.query_api_key" && c.State == doctor.StateWarning {
			found = true
		}
	}
	if !found {
		t.Fatal("expected query api key warning")
	}
}
