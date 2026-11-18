// Package slopscore computes a deterministic, bounded "slop" score for source
// files: a heuristic signal for the patterns generated code tends to leave
// behind. It is a quality signal, never a security verdict, and a high score
// means no known pattern matched, not that the code is correct.
package slopscore

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"
)

const (
	// MaxFiles bounds how many files one analysis will read.
	MaxFiles = 4000
	// MaxFileBytes bounds the size of any single file inspected.
	MaxFileBytes = 1024 * 1024
	// MaxFindings bounds total reported tells so a pathological repository
	// cannot produce unbounded output.
	MaxFindings = 2000
	// DefaultTimeout bounds wall-clock work for one analysis.
	DefaultTimeout = 20 * time.Second
)

// File is one source file to analyse.
type File struct{ Path, Content string }

// Finding is one matched slop tell.
type Finding struct {
	ID       string
	Title    string
	Severity string
	Category string
	File     string
	Line     int
	Detail   string
}

// Metrics summarises an analysis. Score is 0-100, higher is cleaner.
type Metrics struct {
	Score        int
	Grade        string
	FilesScanned int
	LinesScanned int
	Findings     int
	ByRule       map[string]int
}

type tell struct {
	id       string
	title    string
	severity string
	category string
	match    func(path, line string) (detail string, ok bool)
}

var tells = []tell{
	{"SLOP-TODO", "Placeholder marker left in source", "low", "tech_debt", matchTodo},
	{"SLOP-COMMENTED-CODE", "Commented-out code block", "medium", "tech_debt", matchCommentedCode},
	{"SLOP-TAUTOLOGY", "Test assertion that cannot fail", "high", "testing", matchTautology},
	{"SLOP-SWALLOW", "Error swallowed without handling", "high", "reliability", matchSwallow},
	{"SLOP-DEBUG", "Debug output left in code", "medium", "quality", matchDebug},
	{"SLOP-EMOJI", "Emoji used in source or comments", "low", "quality", matchEmoji},
	{"SLOP-MARKETING", "Marketing filler in prose", "info", "copy", matchMarketing},
}

var marketingWords = []string{
	"seamless", "seamlessly", "cutting-edge", "revolutionary", "game-changing",
	"world-class", "blazing fast", "unleash", "empower", "delve", "elevate your",
	"supercharge", "effortless", "state-of-the-art",
}

func matchTodo(_, line string) (string, bool) {
	upper := strings.ToUpper(line)
	for _, marker := range []string{"TODO", "FIXME", "XXX", "HACK"} {
		if idx := strings.Index(upper, marker); idx >= 0 {
			beforeOK := idx == 0 || !isIdent(upper[idx-1])
			after := idx + len(marker)
			afterOK := after >= len(upper) || !isIdent(upper[after])
			if beforeOK && afterOK {
				return marker + " marker", true
			}
		}
	}
	return "", false
}

func matchCommentedCode(_, line string) (string, bool) {
	t := strings.TrimSpace(line)
	var body string
	switch {
	case strings.HasPrefix(t, "//"):
		body = strings.TrimSpace(strings.TrimPrefix(t, "//"))
	case strings.HasPrefix(t, "#"):
		body = strings.TrimSpace(strings.TrimPrefix(t, "#"))
	default:
		return "", false
	}
	if body == "" {
		return "", false
	}
	for _, sig := range []string{"func ", "return ", "if (", "if err", "for (", "for i", "var ", "const ", "print(", "fmt.", "console.", "def ", "class ", "import ", "SELECT ", "UPDATE ", "INSERT "} {
		if strings.HasPrefix(body, sig) {
			return "commented code: " + sig, true
		}
	}
	return "", false
}

func matchTautology(_, line string) (string, bool) {
	l := strings.ToLower(strings.ReplaceAll(line, " ", ""))
	for _, sig := range []string{"expect(true).tobe(true)", "expect(1).tobe(1)", "asserttrue(true)", "assert.true(t,true)", "assert.equal(1,1)"} {
		if strings.Contains(l, sig) {
			return "assertion cannot fail", true
		}
	}
	return "", false
}

func matchSwallow(_, line string) (string, bool) {
	t := strings.ToLower(strings.TrimSpace(line))
	for _, sig := range []string{"except:", "except exception:", "catch {}", "catch (_) {}", "catch(e){}", "catch {}"} {
		if strings.HasPrefix(t, sig) {
			return "empty error handler", true
		}
	}
	return "", false
}

func matchDebug(_, line string) (string, bool) {
	t := strings.TrimSpace(line)
	for _, sig := range []string{"console.log(", "console.debug(", "debugger;", "print("} {
		if strings.HasPrefix(t, sig) {
			return sig, true
		}
	}
	return "", false
}

func matchEmoji(_, line string) (string, bool) {
	for _, r := range line {
		if r >= 0x1F300 && r <= 0x1FAFF {
			return "emoji in source", true
		}
	}
	return "", false
}

func matchMarketing(path, line string) (string, bool) {
	if !strings.HasSuffix(strings.ToLower(path), ".md") {
		return "", false
	}
	lower := strings.ToLower(line)
	for _, word := range marketingWords {
		if strings.Contains(lower, word) {
			return "marketing filler: " + word, true
		}
	}
	return "", false
}

func isIdent(b byte) bool {
	return b == '_' || (b >= 'A' && b <= 'Z') || (b >= 'a' && b <= 'z') || (b >= '0' && b <= '9')
}

var weights = map[string]float64{"high": 8, "medium": 4, "low": 1.5, "info": 0.5}

// Analyze scores the supplied files. It returns an error only for cancelled
// work or exhausted budgets; a repository that exceeds a budget is reported as
// an explicit incomplete result rather than a clean one.
func Analyze(ctx context.Context, files []File) (Metrics, []Finding, error) {
	ctx, cancel := context.WithTimeout(ctx, DefaultTimeout)
	defer cancel()

	metrics := Metrics{ByRule: map[string]int{}}
	findings := make([]Finding, 0)
	if len(files) > MaxFiles {
		return metrics, nil, fmt.Errorf("slop analysis incomplete: %d files exceeds the %d-file budget", len(files), MaxFiles)
	}
	for _, f := range files {
		if err := ctx.Err(); err != nil {
			return metrics, nil, err
		}
		if len(f.Content) > MaxFileBytes {
			return metrics, nil, fmt.Errorf("slop analysis incomplete: %s exceeds %d bytes", f.Path, MaxFileBytes)
		}
		metrics.FilesScanned++
		line := 1
		start := 0
		for start <= len(f.Content) {
			end := strings.IndexByte(f.Content[start:], '\n')
			var text string
			last := end == -1
			if last {
				text = f.Content[start:]
			} else {
				text = f.Content[start : start+end]
			}
			// A trailing newline ends the file; it does not start a new line.
			if !(last && text == "") {
				metrics.LinesScanned++
			}
			for _, t := range tells {
				if detail, ok := t.match(f.Path, text); ok {
					findings = append(findings, Finding{
						ID: t.id, Title: t.title, Severity: t.severity, Category: t.category,
						File: f.Path, Line: line, Detail: detail,
					})
					metrics.ByRule[t.id]++
					if len(findings) >= MaxFindings {
						return metrics, findings, fmt.Errorf("slop analysis incomplete: reached the %d-finding budget", MaxFindings)
					}
				}
			}
			if end == -1 {
				break
			}
			start += end + 1
			line++
		}
	}
	metrics.Findings = len(findings)
	metrics.Score, metrics.Grade = score(findings, metrics.LinesScanned)
	return metrics, findings, nil
}

// score converts weighted tell density into a 0-100 cleanliness score.
func score(findings []Finding, lines int) (int, string) {
	if lines <= 0 {
		return 100, "A"
	}
	penalty := 0.0
	for _, f := range findings {
		penalty += weights[f.Severity]
	}
	// Normalise to a per-1000-line rate so large repositories are not punished
	// merely for size.
	rate := penalty * 1000 / float64(lines)
	value := 100 - rate
	if value < 0 {
		value = 0
	}
	rounded := int(value + 0.5)
	switch {
	case rounded >= 90:
		return rounded, "A"
	case rounded >= 80:
		return rounded, "B"
	case rounded >= 70:
		return rounded, "C"
	case rounded >= 60:
		return rounded, "D"
	default:
		return rounded, "F"
	}
}

// Summarise renders the top rules by count, for logs and reports.
func (m Metrics) Summarise() string {
	type kv struct {
		K string
		V int
	}
	rows := make([]kv, 0, len(m.ByRule))
	for k, v := range m.ByRule {
		rows = append(rows, kv{k, v})
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].V > rows[j].V })
	parts := make([]string, 0, 3)
	for i, r := range rows {
		if i == 3 {
			break
		}
		parts = append(parts, fmt.Sprintf("%s=%d", r.K, r.V))
	}
	return strings.Join(parts, ",")
}
