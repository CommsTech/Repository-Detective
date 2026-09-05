package graph

import (
	"path/filepath"
	"regexp"
	"strings"
)

var genericImportRE = regexp.MustCompile(`(?m)(?:import|require|include|use)\s+['"]?([\w./-]+)['"]?`)

func parseGenericFile(content, path string, info *parsedFile) {
	for _, m := range genericImportRE.FindAllStringSubmatch(content, -1) {
		if m[1] != "" {
			ext := !strings.HasPrefix(m[1], ".")
			info.imports = append(info.imports, importRef{target: m[1], external: ext})
		}
	}
	base := strings.ToLower(filepath.Base(path))
	switch base {
	case "main.go", "main.py", "main.js", "main.ts", "index.js", "index.ts", "app.js", "app.ts", "server.js", "cli.py", "manage.py":
		info.isEntry = true
	}
}
