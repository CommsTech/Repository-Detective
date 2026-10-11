package scanners

import (
	"encoding/json"
	"testing"
)

func TestStripANSI(t *testing.T) {
	raw := []byte("\x1b[31m[{\"RuleID\":\"test\"}]\x1b[0m")
	got := stripANSI(raw)
	if string(got) != `[{"RuleID":"test"}]` {
		t.Fatalf("stripANSI: got %q", got)
	}
}

func TestExtractJSONArrayWithANSIPrefix(t *testing.T) {
	raw := []byte("\x1b[0mWARN: scanning\n[{\"RuleID\":\"r1\"}]\n")
	payload, err := extractJSONArray(raw)
	if err != nil {
		t.Fatal(err)
	}
	var arr []map[string]string
	if err := json.Unmarshal(payload, &arr); err != nil {
		t.Fatal(err)
	}
	if len(arr) != 1 || arr[0]["RuleID"] != "r1" {
		t.Fatalf("unexpected payload: %s", payload)
	}
}

func TestExtractJSONObjectWithTrailingNoise(t *testing.T) {
	raw := []byte("noise\n{\"Issues\":[{\"rule_id\":\"G101\"}]}\nmore")
	payload, err := extractJSONObject(raw)
	if err != nil {
		t.Fatal(err)
	}
	if !json.Valid(payload) {
		t.Fatalf("invalid json: %s", payload)
	}
}

func TestExtractJSONObjectWithDashProgressTrailer(t *testing.T) {
	raw := []byte("{\"SchemaVersion\":2,\"Results\":[]}\n- scanning complete\n")
	payload, err := extractJSONObject(raw)
	if err != nil {
		t.Fatal(err)
	}
	var report map[string]any
	if err := json.Unmarshal(payload, &report); err != nil {
		t.Fatalf("unmarshal extracted: %v payload=%s", err, payload)
	}
}
