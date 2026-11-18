package ai

import "strings"

// RepairRelaxedJSON escapes literal newlines/tabs/carriage-returns inside JSON
// string values. Models often emit multi-line fields without \n escapes, which
// encoding/json rejects with: invalid character '\n' in string literal.
func RepairRelaxedJSON(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return s
	}
	var b strings.Builder
	b.Grow(len(s) + 32)
	inString := false
	escape := false
	for i := 0; i < len(s); i++ {
		ch := s[i]
		if inString {
			if escape {
				b.WriteByte(ch)
				escape = false
				continue
			}
			if ch == '\\' {
				b.WriteByte(ch)
				escape = true
				continue
			}
			if ch == '"' {
				inString = false
				b.WriteByte(ch)
				continue
			}
			switch ch {
			case '\n':
				b.WriteString(`\n`)
			case '\r':
				b.WriteString(`\r`)
			case '\t':
				b.WriteString(`\t`)
			default:
				b.WriteByte(ch)
			}
			continue
		}
		if ch == '"' {
			inString = true
		}
		b.WriteByte(ch)
	}
	return b.String()
}

// ExtractJSONObject pulls the first top-level JSON object from model output.
// Handles prose prefixes and optional markdown fences (common LLM failure mode).
func ExtractJSONObject(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return s
	}
	if strings.HasPrefix(s, "```") {
		s = strings.TrimPrefix(s, "```json")
		s = strings.TrimPrefix(s, "```JSON")
		s = strings.TrimPrefix(s, "```")
		if idx := strings.LastIndex(s, "```"); idx >= 0 {
			s = s[:idx]
		}
		s = strings.TrimSpace(s)
	}
	start := strings.Index(s, "{")
	if start < 0 {
		return strings.TrimSpace(s)
	}
	depth := 0
	inString := false
	escape := false
	for i := start; i < len(s); i++ {
		ch := s[i]
		if inString {
			if escape {
				escape = false
				continue
			}
			if ch == '\\' {
				escape = true
				continue
			}
			if ch == '"' {
				inString = false
			}
			continue
		}
		switch ch {
		case '"':
			inString = true
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				return strings.TrimSpace(s[start : i+1])
			}
		}
	}
	return strings.TrimSpace(s[start:])
}

// DecodeModelJSON prepares model output for encoding/json by stripping fences
// / prose wrappers and repairing unescaped whitespace inside string literals.
func DecodeModelJSON(s string) string {
	return RepairRelaxedJSON(ExtractJSONObject(s))
}
