// Package customrules implements bounded, declarative, repository-scoped static checks.
// Rule files are untrusted repository content: no user-defined commands, scripts,
// includes, imports, external URLs, or arbitrary executables are accepted.
package customrules

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"path"
	"regexp"
	"strings"
	"time"
)

const (
	Filename       = ".repository-detective.rules.json"
	MaxPolicyBytes = 64 * 1024
	MaxRules       = 32
	MaxFileBytes   = 2 * 1024 * 1024
	MaxFindings    = 2000
)

var ruleIDRE = regexp.MustCompile(`^[A-Z][A-Z0-9_-]{2,47}$`)
var categories = map[string]bool{"security": true, "quality": true, "performance": true, "maintainability": true, "architecture": true, "misconfiguration": true, "supply_chain": true, "testing": true, "tech_debt": true}
var severities = map[string]bool{"critical": true, "high": true, "medium": true, "low": true, "info": true}

type Policy struct {
	Version int    `json:"version"`
	Rules   []Rule `json:"rules"`
}
type Rule struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description,omitempty"`
	Category    string `json:"category"`
	Severity    string `json:"severity"`
	PathGlob    string `json:"path_glob,omitempty"`
	Pattern     string `json:"pattern"`
}
type File struct{ Path, Content string }
type Finding struct {
	RuleID, Title, Description, Category, Severity, File string
	Line                                                 int
}

type compiledRule struct {
	Rule
	re *regexp.Regexp
}

// Compile rejects malformed and resource-intensive policies before scanning.
func Compile(data []byte) ([]compiledRule, error) {
	if len(data) == 0 || len(data) > MaxPolicyBytes {
		return nil, fmt.Errorf("rule config size must be 1..%d bytes", MaxPolicyBytes)
	}
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	var policy Policy
	if err := dec.Decode(&policy); err != nil {
		return nil, fmt.Errorf("custom rule configuration: %w", err)
	}
	var trailing any
	if err := dec.Decode(&trailing); err != io.EOF {
		return nil, errors.New("rule configuration must contain exactly one JSON object")
	}
	if policy.Version != 1 {
		return nil, errors.New("custom rule version must be 1")
	}
	if len(policy.Rules) == 0 || len(policy.Rules) > MaxRules {
		return nil, fmt.Errorf("custom rules count must be 1..%d", MaxRules)
	}
	compiled := make([]compiledRule, 0, len(policy.Rules))
	ids := make(map[string]bool, len(policy.Rules))
	for i, r := range policy.Rules {
		if !ruleIDRE.MatchString(r.ID) || ids[r.ID] {
			return nil, fmt.Errorf("rule %d: invalid or duplicate ID", i+1)
		}
		ids[r.ID] = true
		if strings.TrimSpace(r.Title) == "" || len(r.Title) > 160 || len(r.Description) > 400 {
			return nil, fmt.Errorf("rule %s: invalid title or description length", r.ID)
		}
		if !categories[r.Category] || !severities[r.Severity] {
			return nil, fmt.Errorf("rule %s: unsupported category or severity", r.ID)
		}
		if len(r.Pattern) == 0 || len(r.Pattern) > 320 {
			return nil, fmt.Errorf("rule %s: regex length must be 1..320", r.ID)
		}
		if len(r.PathGlob) > 200 || strings.Contains(r.PathGlob, "\\") || strings.HasPrefix(r.PathGlob, "/") || strings.Contains(r.PathGlob, "..") {
			return nil, fmt.Errorf("rule %s: unsafe path glob", r.ID)
		}
		if r.PathGlob != "" {
			if _, err := path.Match(r.PathGlob, "a.go"); err != nil {
				return nil, fmt.Errorf("rule %s: invalid path glob: %w", r.ID, err)
			}
		}
		re, err := regexp.Compile(r.Pattern)
		if err != nil {
			return nil, fmt.Errorf("rule %s: invalid RE2 regex: %w", r.ID, err)
		}
		compiled = append(compiled, compiledRule{Rule: r, re: re})
	}
	return compiled, nil
}

// Scan returns deterministic candidate findings or a blocking coverage error.
// The caller must not report a complete/clean scan after any Scan error.
func Scan(ctx context.Context, config []byte, files []File) ([]Finding, error) {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	rules, err := Compile(config)
	if err != nil {
		return nil, err
	}
	findings := make([]Finding, 0)
	for _, f := range files {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		filePath := strings.ReplaceAll(f.Path, "\\", "/")
		if filePath == Filename || strings.HasPrefix(filePath, "/") || strings.HasPrefix(filePath, "../") || strings.Contains(filePath, "/../") {
			continue
		}
		selected := make([]compiledRule, 0)
		for _, r := range rules {
			if r.PathGlob == "" {
				selected = append(selected, r)
				continue
			}
			matches, err := path.Match(r.PathGlob, filePath)
			if err == nil && matches {
				selected = append(selected, r)
			}
		}
		if len(selected) == 0 {
			continue
		}
		if len(f.Content) > MaxFileBytes {
			return nil, fmt.Errorf("custom rule coverage incomplete: %s exceeds %d bytes", filePath, MaxFileBytes)
		}
		line := 1
		start := 0
		for start < len(f.Content) {
			if err := ctx.Err(); err != nil {
				return nil, err
			}
			end := strings.IndexByte(f.Content[start:], '\n')
			if end == -1 {
				end = len(f.Content)
			} else {
				end += start
			}
			lineText := f.Content[start:end]
			for _, r := range selected {
				if r.re.MatchString(lineText) {
					findings = append(findings, Finding{RuleID: r.ID, Title: r.Title, Description: r.Description, Category: r.Category, Severity: r.Severity, File: filePath, Line: line})
					if len(findings) >= MaxFindings {
						return nil, fmt.Errorf("custom rule coverage incomplete: reached %d-finding budget", MaxFindings)
					}
				}
			}
			if end == len(f.Content) {
				break
			}
			start = end + 1
			line++
		}
	}
	return findings, nil
}
