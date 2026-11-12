package ai_test

import (
	"encoding/json"
	"testing"

	"git.commsnet.org/commstech/repository-detective/ai"
)

func TestExtractJSONObjectFromProseAndFence(t *testing.T) {
	raw := "Here is the analysis:\n```json\n{\"findings\":[{\"id\":\"1\"}]}\n```\nThanks!"
	got := ai.ExtractJSONObject(raw)
	if got != `{"findings":[{"id":"1"}]}` {
		t.Fatalf("extract = %q", got)
	}
	var parsed map[string]any
	if err := json.Unmarshal([]byte(ai.DecodeModelJSON(raw)), &parsed); err != nil {
		t.Fatalf("decode: %v", err)
	}
}

func TestRepairRelaxedJSONEscapesNewlinesInStrings(t *testing.T) {
	raw := "{\n  \"arguments\": \"line1\nline2\"\n}"
	fixed := ai.RepairRelaxedJSON(raw)
	var parsed map[string]any
	if err := json.Unmarshal([]byte(fixed), &parsed); err != nil {
		t.Fatalf("repair failed: %v (fixed=%q)", err, fixed)
	}
}

func TestDecodeModelJSONHandlesLeadingAsteriskProse(t *testing.T) {
	raw := "**Note**\n```JSON\n{\"confidence\":0.9,\"arguments\":\"ok\"}\n```"
	var parsed struct {
		Confidence float64 `json:"confidence"`
		Arguments  string  `json:"arguments"`
	}
	if err := json.Unmarshal([]byte(ai.DecodeModelJSON(raw)), &parsed); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if parsed.Confidence != 0.9 || parsed.Arguments != "ok" {
		t.Fatalf("unexpected parse: %+v", parsed)
	}
}
