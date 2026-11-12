package scanners

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"git.commsnet.org/commstech/repository-detective/internal/security"
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
	// Tool logs often contain brackets that are not JSON (`[gosec] ...`) ahead of
	// the real report. Skip any start the decoder cannot consume.
	for i := 0; i < len(s); i++ {
		if s[i] != '{' && s[i] != '[' {
			continue
		}
		dec := json.NewDecoder(bytes.NewReader(s[i:]))
		var raw json.RawMessage
		if err := dec.Decode(&raw); err != nil {
			continue
		}
		if len(bytes.TrimSpace(raw)) == 0 {
			continue
		}
		return []byte(raw), nil
	}
	return nil, errNoJSONValue
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

// redactScannerDetail collapses noisy ANSI tool logs into a single-line operator message
// and routes through the central credential sanitizer (RD-034).
func redactScannerDetail(detail string) string {
	clean := string(stripANSI([]byte(detail)))
	clean = strings.Join(strings.Fields(clean), " ")
	return security.SanitizeDiagnostic(clean, 400)
}
