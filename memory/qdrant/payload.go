package qdrant

import (
	"encoding/json"
	"fmt"
)

// HitPayload is a normalized view of a Qdrant search hit payload.
type HitPayload struct {
	IssueURL    string
	IssueNumber int
	ClusterID   string
	Title       string
}

// ParseHitPayload extracts match fields from legacy or cah_findings payloads.
func ParseHitPayload(raw map[string]any) HitPayload {
	if raw == nil {
		return HitPayload{}
	}
	out := HitPayload{
		IssueURL:  stringField(raw, "issue_url"),
		ClusterID: stringField(raw, "cluster_id"),
	}
	if n := intField(raw, "gitea_issue"); n > 0 {
		out.IssueNumber = n
	}
	if n := intField(raw, "issue_number"); n > 0 && out.IssueNumber == 0 {
		out.IssueNumber = n
	}
	out.Title = stringField(raw, "title")
	if out.Title == "" {
		out.Title = stringField(raw, "description")
	}
	return out
}

// PayloadToMap converts a typed payload to a generic map for Qdrant upsert.
func PayloadToMap(payload CAHFindingPayload) (map[string]any, error) {
	raw, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	var out map[string]any
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func stringField(raw map[string]any, key string) string {
	v, ok := raw[key]
	if !ok || v == nil {
		return ""
	}
	switch t := v.(type) {
	case string:
		return t
	default:
		return fmt.Sprint(t)
	}
}

func intField(raw map[string]any, key string) int {
	v, ok := raw[key]
	if !ok || v == nil {
		return 0
	}
	switch t := v.(type) {
	case float64:
		return int(t)
	case int:
		return t
	case int64:
		return int(t)
	case json.Number:
		n, _ := t.Int64()
		return int(n)
	default:
		return 0
	}
}
