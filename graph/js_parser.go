package graph

import (
	"regexp"
	"strings"
)

var (
	jsImportRE  = regexp.MustCompile(`(?m)(?:import\s+(?:[\w*{}\s,]+\s+from\s+)?['"]([^'"]+)['"]|require\s*\(\s*['"]([^'"]+)['"]\s*\))`)
	jsExportFn  = regexp.MustCompile(`(?m)(?:export\s+(?:async\s+)?function\s+(\w+)|export\s+const\s+(\w+)\s*=)`)
)

func parseJSFile(content string, info *parsedFile) {
	for _, m := range jsImportRE.FindAllStringSubmatch(content, -1) {
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
	for _, m := range jsExportFn.FindAllStringSubmatch(content, -1) {
		name := m[1]
		if name == "" {
			name = m[2]
		}
		if name != "" {
			info.functions = append(info.functions, funcRef{name: name, exported: true})
		}
	}
}
