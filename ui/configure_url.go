package ui

import "strings"

// configureSectionURL returns a Configure page deep link for a feature section.
func configureSectionURL(basePath, section string) string {
	basePath = strings.TrimSuffix(strings.TrimSpace(basePath), "/")
	section = strings.TrimPrefix(strings.TrimSpace(section), "#")
	if section == "" {
		return basePath + "/configure"
	}
	return basePath + "/configure#" + section
}
