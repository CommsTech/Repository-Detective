package forge

import (
	"encoding/json"
	"strings"
)

// DecodeRepositoryContents handles JSON array (directory) or single object (file).
func DecodeRepositoryContents(body []byte) ([]RepositoryContent, error) {
	trimmed := strings.TrimSpace(string(body))
	if trimmed == "" || trimmed == "null" {
		return nil, nil
	}
	if trimmed[0] == '[' {
		var contents []RepositoryContent
		if err := json.Unmarshal(body, &contents); err != nil {
			return nil, err
		}
		return contents, nil
	}
	var content RepositoryContent
	if err := json.Unmarshal(body, &content); err != nil {
		return nil, err
	}
	return []RepositoryContent{content}, nil
}
