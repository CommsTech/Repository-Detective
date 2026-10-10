package issues

import "strings"

// IssueBodyMetadata holds fields parsed from a filed issue body.
type IssueBodyMetadata struct {
	Title    string
	Severity string
	Category string
	Source   string
	RuleID   string
	File     string
}

// ParseIssueBodyMetadata extracts structured fields from Repository Detective issue bodies.
func ParseIssueBodyMetadata(body string) IssueBodyMetadata {
	meta := IssueBodyMetadata{}
	for _, line := range strings.Split(body, "\n") {
		line = strings.TrimSpace(line)
		line = strings.TrimPrefix(line, "- ")
		switch {
		case strings.HasPrefix(line, "Category:"):
			meta.Category = strings.TrimSpace(strings.TrimPrefix(line, "Category:"))
		case strings.HasPrefix(line, "Severity:"):
			meta.Severity = strings.TrimSpace(strings.TrimPrefix(line, "Severity:"))
		case strings.HasPrefix(line, "Source:"):
			meta.Source = strings.TrimSpace(strings.TrimPrefix(line, "Source:"))
		case strings.HasPrefix(line, "Rule ID:"):
			meta.RuleID = strings.TrimSpace(strings.TrimPrefix(line, "Rule ID:"))
		case strings.HasPrefix(line, "**File:**"):
			meta.File = strings.TrimSpace(strings.TrimPrefix(line, "**File:**"))
		case strings.HasPrefix(line, "## Summary") && meta.Title == "":
			continue
		case meta.Title == "" && line != "" && !strings.HasPrefix(line, "#") && !strings.HasPrefix(line, "**"):
			if !strings.Contains(line, ":") {
				meta.Title = line
			}
		}
	}
	return meta
}
