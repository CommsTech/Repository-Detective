package graph

import (
	"regexp"
	"strings"
)

var (
	pyImportRE = regexp.MustCompile(`(?m)^(?:from\s+([\w.]+)\s+import|import\s+([\w.]+))`)
	pyDefRE    = regexp.MustCompile(`(?m)^def\s+(\w+)\s*\(`)
)

func parsePythonFile(content string, info *parsedFile) {
	for _, m := range pyImportRE.FindAllStringSubmatch(content, -1) {
		target := m[1]
		if target == "" {
			target = m[2]
		}
		if target == "" {
			continue
		}
		ext := !strings.HasPrefix(target, ".")
		info.imports = append(info.imports, importRef{target: target, external: ext})
	}
	for _, m := range pyDefRE.FindAllStringSubmatch(content, -1) {
		if m[1] != "" {
			info.functions = append(info.functions, funcRef{name: m[1], exported: !strings.HasPrefix(m[1], "_")})
		}
	}
	if strings.Contains(content, `if __name__ == "__main__"`) {
		info.isEntry = true
	}
}
