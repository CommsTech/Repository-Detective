package scanners

import "testing"

func TestParseGrypeOutputValidJSON(t *testing.T) {
	raw := `{"matches":[{"vulnerability":{"id":"CVE-2024-1","severity":"High","description":"test"},"artifact":{"name":"pkg","version":"1.0.0","locations":[{"path":"/tmp/go.mod"}]}}]}`
	findings, status, detail := parseGrypeOutput([]byte(raw), "/tmp", Config{GrypeFailOn: "high"})
	if status != StatusFound {
		t.Fatalf("status=%s detail=%q", status, detail)
	}
	if len(findings) != 1 || findings[0].ID != "GRYPE-CVE-2024-1" {
		t.Fatalf("findings=%+v", findings)
	}
}

func TestParseGrypeOutputWarningPrefixJSON(t *testing.T) {
	raw := "WARN: stderr noise\n{\"matches\":[]}\n"
	findings, status, detail := parseGrypeOutput([]byte(raw), "/tmp", Config{})
	if status != StatusClean {
		t.Fatalf("status=%s detail=%q", status, detail)
	}
	if len(findings) != 0 {
		t.Fatalf("expected no findings, got %d", len(findings))
	}
}

func TestParseGrypeOutputNoSupportedManifest(t *testing.T) {
	raw := "No packages were discovered in the source path"
	_, status, detail := parseGrypeOutput([]byte(raw), "/tmp", Config{})
	if status != StatusNoSupportedManifest {
		t.Fatalf("status=%s detail=%q", status, detail)
	}
}

func TestParseGrypeOutputScannerUnavailable(t *testing.T) {
	raw := "failed to load vulnerability db: database disk image is malformed (11)"
	_, status, detail := parseGrypeOutput([]byte(raw), "/tmp", Config{})
	if status != StatusScannerUnavailable {
		t.Fatalf("status=%s detail=%q", status, detail)
	}
}

func TestParseGrypeOutputMalformedJSON(t *testing.T) {
	raw := "invalid character 'i' in literal false (expecting 'l')"
	_, status, _ := parseGrypeOutput([]byte(raw), "/tmp", Config{})
	if status != StatusParseFailed {
		t.Fatalf("status=%s", status)
	}
}

func TestParseGrypeOutputEmpty(t *testing.T) {
	_, status, detail := parseGrypeOutput(nil, "/tmp", Config{})
	if status != StatusNoSupportedManifest {
		t.Fatalf("status=%s detail=%q", status, detail)
	}
}
