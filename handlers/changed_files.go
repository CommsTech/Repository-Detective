package handlers

// CollectChangedFiles returns unique added and modified paths from push commits.
// Deleted files are excluded because there is no content to analyze.
func CollectChangedFiles(commits []Commit) []string {
	seen := make(map[string]struct{})
	var paths []string

	for _, commit := range commits {
		for _, path := range commit.Added {
			if path == "" {
				continue
			}
			if _, ok := seen[path]; ok {
				continue
			}
			seen[path] = struct{}{}
			paths = append(paths, path)
		}
		for _, path := range commit.Modified {
			if path == "" {
				continue
			}
			if _, ok := seen[path]; ok {
				continue
			}
			seen[path] = struct{}{}
			paths = append(paths, path)
		}
	}

	return paths
}
