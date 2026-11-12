package scanners

import (
	"bufio"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const defaultGoScannerMaxFindings = 100

// HasGoModule reports whether the workspace root contains a go.mod file.
func HasGoModule(dir string) bool {
	if dir == "" {
		return false
	}
	info, err := os.Stat(filepath.Join(dir, "go.mod"))
	return err == nil && !info.IsDir()
}

// HasGoFiles reports whether any workspace entry is a Go source file.
func HasGoFiles(entries []FileEntry) bool {
	for _, entry := range entries {
		if strings.HasSuffix(strings.ToLower(entry.Path), ".go") {
			return true
		}
	}
	return false
}

// WorkspaceHasGo reports whether govulncheck/staticcheck/gosec should attempt a scan.
func WorkspaceHasGo(dir string, entries []FileEntry) bool {
	return HasGoModule(dir) || HasGoFiles(entries)
}

// GoTypecheckWorkspaceReady reports whether a workspace is safe for typechecking
// scanners (staticcheck/gosec/govulncheck). Incomplete API/scoped copies and
// mixed package clauses in one directory produce compile diagnostics that were
// previously mislabeled parse_failed (e.g. undefined: builder / found packages).
func GoTypecheckWorkspaceReady(dir string, entries []FileEntry) (ok bool, reason string) {
	// Prefer in-memory entries when present. When only a directory is provided,
	// walk the tree — do not require go.mod up front (API-scoped copies often
	// include .go files without a module file).
	if len(entries) > 0 && !WorkspaceHasGo(dir, entries) {
		return false, "no Go files"
	}
	if len(entries) == 0 && dir == "" {
		return false, "no Go files"
	}

	pkgByDir := map[string]map[string]struct{}{}
	addPackage := func(relDir, pkg string) {
		if pkg == "" || pkg == "documentation" {
			return
		}
		if pkgByDir[relDir] == nil {
			pkgByDir[relDir] = map[string]struct{}{}
		}
		pkgByDir[relDir][pkg] = struct{}{}
	}

	if len(entries) > 0 {
		for _, entry := range entries {
			path := strings.ReplaceAll(entry.Path, "\\", "/")
			if !strings.HasSuffix(strings.ToLower(path), ".go") {
				continue
			}
			base := filepath.Base(path)
			if strings.HasSuffix(base, "_test.go") {
				continue
			}
			relDir := filepath.ToSlash(filepath.Dir(path))
			if relDir == "." {
				relDir = ""
			}
			pkg := packageClauseFromSource(string(entry.Content))
			if pkg == "" && dir != "" {
				pkg = packageClauseFromFile(filepath.Join(dir, filepath.FromSlash(path)))
			}
			addPackage(relDir, pkg)
		}
	} else {
		_ = filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
			if err != nil || d.IsDir() {
				return nil
			}
			name := d.Name()
			if !strings.HasSuffix(strings.ToLower(name), ".go") || strings.HasSuffix(name, "_test.go") {
				return nil
			}
			rel, relErr := filepath.Rel(dir, path)
			if relErr != nil {
				return nil
			}
			relDir := filepath.ToSlash(filepath.Dir(rel))
			if relDir == "." {
				relDir = ""
			}
			addPackage(relDir, packageClauseFromFile(path))
			return nil
		})
	}

	if len(pkgByDir) == 0 {
		return false, "no Go files"
	}

	for relDir, pkgs := range pkgByDir {
		if len(pkgs) <= 1 {
			continue
		}
		names := make([]string, 0, len(pkgs))
		for name := range pkgs {
			names = append(names, name)
		}
		sort.Strings(names)
		where := relDir
		if where == "" {
			where = "."
		}
		return false, "mixed Go packages in " + where + " (" + strings.Join(names, ", ") + "); incomplete or fixture-contaminated workspace"
	}
	return true, ""
}

func packageClauseFromFile(path string) string {
	f, err := os.Open(path)
	if err != nil {
		return ""
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	var b strings.Builder
	for scanner.Scan() {
		b.WriteString(scanner.Text())
		b.WriteByte('\n')
		if b.Len() > 8*1024 {
			break
		}
	}
	return packageClauseFromSource(b.String())
}

func packageClauseFromSource(src string) string {
	for _, line := range strings.Split(src, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "//") {
			continue
		}
		if strings.HasPrefix(line, "/*") {
			continue
		}
		if strings.HasPrefix(line, "package ") {
			pkg := strings.TrimSpace(strings.TrimPrefix(line, "package "))
			pkg = strings.TrimSuffix(pkg, ";")
			if i := strings.IndexAny(pkg, " \t/"); i >= 0 {
				pkg = pkg[:i]
			}
			return pkg
		}
		// Stop at first non-comment code that is not a package clause.
		if !strings.HasPrefix(line, "package") {
			return ""
		}
	}
	return ""
}

func goScannerMaxFindings(cfg Config) int {
	if cfg.GoScannerMaxFindings > 0 {
		return cfg.GoScannerMaxFindings
	}
	return defaultGoScannerMaxFindings
}

type cappedFindings struct {
	Findings  []Finding
	Total     int
	Truncated bool
}

func capFindings(findings []Finding, max int) cappedFindings {
	if max <= 0 {
		max = defaultGoScannerMaxFindings
	}
	total := len(findings)
	if total <= max {
		return cappedFindings{Findings: findings, Total: total}
	}
	return cappedFindings{
		Findings:  append([]Finding(nil), findings[:max]...),
		Total:     total,
		Truncated: true,
	}
}

func truncateDetailForScanner(scanner string, max, total int) string {
	if total <= max {
		return ""
	}
	if scanner != "" {
		return truncateDetailLabel(scanner, max, total)
	}
	return truncateDetailLabel("findings", max, total)
}

func truncateDetailLabel(label string, max, total int) string {
	return "truncated to " + itoa(max) + " " + label + " (" + itoa(total) + " total)"
}

func itoa(v int) string {
	if v == 0 {
		return "0"
	}
	var buf [20]byte
	i := len(buf)
	n := v
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	return string(buf[i:])
}

func scannerTimeoutSeconds(specific, fallback int) time.Duration {
	seconds := specific
	if seconds <= 0 {
		seconds = fallback
	}
	if seconds <= 0 {
		seconds = 120
	}
	return time.Duration(seconds) * time.Second
}
