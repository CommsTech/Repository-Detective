package issues

import (
	"fmt"
	"path/filepath"
	"strings"

	"git.commsnet.org/commstech/repository-detective/ai"
)

// OperatorIssueTitle builds a developer-facing title (rule + concrete location).
func OperatorIssueTitle(issue *ai.CodeIssue) string {
	if issue == nil {
		return "Repository Detective finding"
	}
	rule := strings.TrimSpace(issue.RuleID)
	file := filepath.Base(strings.TrimSpace(issue.File))
	if file == "." || file == "/" {
		file = ""
	}
	src := strings.ToLower(strings.TrimSpace(issue.Source))
	title := strings.TrimSpace(issue.Title)

	switch {
	case rule != "" && file != "":
		short := titleHint(issue)
		if short != "" {
			return fmt.Sprintf("%s: %s in %s", rule, short, file)
		}
		return fmt.Sprintf("%s in %s", rule, file)
	case rule != "":
		if title != "" && !strings.EqualFold(title, rule) {
			return fmt.Sprintf("%s: %s", rule, title)
		}
		return rule
	case src == "shellcheck" && file != "":
		return fmt.Sprintf("ShellCheck: %s in %s", trimTitle(title), file)
	case file != "" && title != "":
		return fmt.Sprintf("%s — %s", trimTitle(title), file)
	case title != "":
		return trimTitle(title)
	default:
		return "Repository Detective finding"
	}
}

func titleHint(issue *ai.CodeIssue) string {
	rule := strings.ToUpper(strings.TrimSpace(issue.RuleID))
	switch rule {
	case "G201":
		return "SQL query constructed with fmt.Sprintf"
	case "G104":
		return "unchecked error return"
	}
	desc := strings.TrimSpace(issue.Description)
	if desc != "" && !isGenericScannerBlurb(desc, issue) && len(desc) < 80 {
		return desc
	}
	title := strings.TrimSpace(issue.Title)
	// Strip leading "[MEDIUM] " style prefixes and rule echoes.
	title = trimTitle(title)
	if strings.HasPrefix(strings.ToUpper(title), rule+":") {
		title = strings.TrimSpace(title[len(rule)+1:])
	}
	if title == "" || strings.EqualFold(title, rule) {
		return ""
	}
	if len(title) > 72 {
		return title[:69] + "…"
	}
	return title
}

func trimTitle(title string) string {
	title = strings.TrimSpace(title)
	for _, prefix := range []string{"[CRITICAL] ", "[HIGH] ", "[MEDIUM] ", "[LOW] ", "[INFO] "} {
		if strings.HasPrefix(strings.ToUpper(title), strings.TrimSpace(prefix)) {
			title = strings.TrimSpace(title[len(prefix):])
		}
	}
	return title
}
