package scanners

import (
	"bytes"
	"fmt"
	"strings"
)

// extractJSONArray returns the first JSON array found in scanner output (stdout+stderr).
func extractJSONArray(output []byte) ([]byte, error) {
	clean := stripANSI(output)
	s := strings.TrimSpace(string(clean))
	start := strings.Index(s, "[")
	end := strings.LastIndex(s, "]")
	if start < 0 || end <= start {
		return nil, fmt.Errorf("no JSON array in output")
	}
	return []byte(s[start : end+1]), nil
}

// extractJSONObject returns the first JSON object found in scanner output (stdout+stderr).
func extractJSONObject(output []byte) ([]byte, error) {
	clean := stripANSI(output)
	s := strings.TrimSpace(string(clean))
	start := strings.Index(s, "{")
	end := strings.LastIndex(s, "}")
	if start < 0 || end <= start {
		return nil, fmt.Errorf("no JSON object in output")
	}
	return []byte(s[start : end+1]), nil
}

// stripANSI removes terminal escape sequences that break JSON parsers.
func stripANSI(b []byte) []byte {
	if len(b) == 0 {
		return b
	}
	var out bytes.Buffer
	out.Grow(len(b))
	for i := 0; i < len(b); i++ {
		if b[i] == '\x1b' {
			for i < len(b) && b[i] != 'm' {
				i++
			}
			continue
		}
		out.WriteByte(b[i])
	}
	return out.Bytes()
}
