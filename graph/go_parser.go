package graph

import (
	"go/ast"
	"go/parser"
	"go/token"
	"strings"
)

func parseGoFile(content, path string, info *parsedFile) {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, path, content, parser.ParseComments)
	if err != nil {
		parseGenericFile(content, path, info)
		return
	}
	if f.Name != nil {
		info.packageName = f.Name.Name
	}
	for _, imp := range f.Imports {
		if imp == nil || imp.Path == nil {
			continue
		}
		p := strings.Trim(imp.Path.Value, `"`)
		ext := !strings.Contains(p, ".") && !strings.HasPrefix(p, info.packageName)
		if strings.Contains(p, "/") && !strings.HasPrefix(p, ".") {
			ext = true
		}
		info.imports = append(info.imports, importRef{target: p, external: ext})
	}
	ast.Inspect(f, func(n ast.Node) bool {
		fn, ok := n.(*ast.FuncDecl)
		if !ok || fn.Name == nil {
			return true
		}
		exported := ast.IsExported(fn.Name.Name)
		line := fset.Position(fn.Pos()).Line
		info.functions = append(info.functions, funcRef{name: fn.Name.Name, exported: exported, line: line})
		if fn.Name.Name == "main" && info.packageName == "main" {
			info.isEntry = true
		}
		return true
	})
}
