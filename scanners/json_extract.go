package scanners

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

// errNoJSONValue reports that scanner output held only log lines and no JSON at all.
// Scanners that stay silent on a clean run can treat this as an empty result
// instead of a parse failure.
var errNoJSONValue = errors.New("no JSON value in output")

// extractJSONArray returns the first JSON array found in scanner output (stdout+stderr).
func extractJSONArray(output []byte) ([]byte, error) {
	raw, err := extractFirstJSONValue(output)
	if err != nil {
		return nil, err
	}
	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) == 0 || trimmed[0] != '[' {
		return nil, fmt.Errorf("no JSON array in output")
	}
	return trimmed, nil
}

// extractJSONObject returns the first JSON object found in scanner output (stdout+stderr).
func extractJSONObject(output []byte) ([]byte, error) {
	raw, err := extractFirstJSONValue(output)
	if err != nil {
		return nil, err
	}
	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) == 0 || trimmed[0] != '{' {
		return nil, fmt.Errorf("no JSON object in output")
	}
	return trimmed, nil
}

// extractFirstJSONValue decodes the first JSON value after optional log noise.
// Trailing progress lines (often starting with '-') are ignored.
func extractFirstJSONValue(output []byte) ([]byte, error) {
	clean := stripANSI(output)
	s := bytes.TrimSpace(clean)
	if len(s) == 0 {
		return nil, fmt.Errorf("empty scanner output")
	}
	start := -1
	for i, b := range s {
		if b == '{' || b == '[' {
			start = i
			break
		}
	}
	if start < 0 {
		return nil, errNoJSONValue
	}
	dec := json.NewDecoder(bytes.NewReader(s[start:]))
	var raw json.RawMessage
	if err := dec.Decode(&raw); err != nil {
		return nil, err
	}
	return []byte(raw), nil
}

// stripANSI removes terminal escape sequences that break JSON parsers.
func stripANSI(b []byte) []byte {
	if len(b) == 0 {
		return b
	}
	var out bytes.Buffer
	out.Grow(len(b))
	for i := 0; i < len(b); i++ {
		if b[i] == 0x1b {
			for i < len(b) && b[i] != 'm' {
				i++
			}
			continue
		}
		out.WriteByte(b[i])
	}
	return out.Bytes()
}

// redactScannerDetail collapses noisy ANSI tool logs into a single-line operator message.
func redactScannerDetail(detail string) string {
	clean := string(stripANSI([]byte(detail)))
	clean = strings.Join(strings.Fields(clean), " ")
	if len(clean) > 400 {
		return clean[:397] + "..."
	}
	return clean
}
