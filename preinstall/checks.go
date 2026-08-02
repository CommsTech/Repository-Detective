package preinstall

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"git.commsnet.org/commstech/repository-detective/issues"
	"git.commsnet.org/commstech/repository-detective/scanners"
	"git.commsnet.org/commstech/repository-detective/store"
)

var (
	curlPipeShell = regexp.MustCompile(`(?i)(curl|wget)\s+[^\n|]*\|\s*(ba)?sh`)
	riskyWorkflow = regexp.MustCompile(`(?i)(pull_request_target|contents:\s*write|id-token:\s*write|packages:\s*write)`)
)

var lockfilePairs = map[string]string{
	"package.json":    "package-lock.json",
	"requirements.txt": "requirements.txt", // pip without lock is flagged via requirements only
	"pyproject.toml":  "poetry.lock",
	"Pipfile":         "Pipfile.lock",
	"Cargo.toml":      "Cargo.lock",
	"Gemfile":         "Gemfile.lock",
	"composer.json":   "composer.lock",
	"go.mod":          "go.sum",
}

// RunStaticChecks performs deterministic pre-install supply-chain checks.
func RunStaticChecks(workspace string, repoRef string, maxFindings int) []store.AuditFinding {
	if maxFindings <= 0 {
		maxFindings = 200
	}
	var findings []store.AuditFinding
	if err := filepath.WalkDir(workspace, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		if d.Type()&os.ModeSymlink != 0 {
			return nil
		}
		rel, err := filepath.Rel(workspace, path)
		if err != nil {
			return nil
		}
		rel = filepath.ToSlash(rel)
		if _, err := scanners.ValidateWorkspacePath(workspace, rel); err != nil {
			return nil
		}
		base := strings.ToLower(filepath.Base(rel))
		switch {
		case base == "package.json":
			findings = append(findings, checkPackageJSON(workspace, rel, repoRef)...)
		case base == "dockerfile" || strings.HasPrefix(base, "dockerfile."):
			findings = append(findings, checkDockerfile(workspace, rel, repoRef)...)
		case strings.Contains(rel, ".github/workflows/") && (strings.HasSuffix(base, ".yml") || strings.HasSuffix(base, ".yaml")):
			findings = append(findings, checkWorkflow(workspace, rel, repoRef)...)
		case base == "install.sh" || base == "setup.sh" || base == "bootstrap.sh":
			findings = append(findings, makePreinstallFinding(repoRef, rel, 1, "medium", 0.85,
				"supply_chain", "preinstall.install_script", "Risky install script present",
				"Install/bootstrap script detected; review before running.", "")...)
		}
		if len(findings) >= maxFindings {
			return filepath.SkipAll
		}
		return nil
	}); err != nil {
		return findings
	}

	findings = append(findings, checkMissingLockfiles(workspace, repoRef)...)
	if len(findings) > maxFindings {
		findings = findings[:maxFindings]
	}
	return findings
}

func checkPackageJSON(workspace, rel, repoRef string) []store.AuditFinding {
	safe, err := scanners.ValidateWorkspacePath(workspace, rel)
	if err != nil {
		return nil
	}
	data, err := os.ReadFile(filepath.Join(workspace, filepath.FromSlash(safe)))
	if err != nil {
		return nil
	}
	var pkg struct {
		Scripts map[string]string `json:"scripts"`
	}
	if err := json.Unmarshal(data, &pkg); err != nil {
		return nil
	}
	var out []store.AuditFinding
	for name, script := range pkg.Scripts {
		lower := strings.ToLower(name)
		if lower == "preinstall" || lower == "postinstall" || lower == "prepare" {
			out = append(out, makePreinstallFinding(repoRef, rel, 1, "high", 0.9,
				"supply_chain", "preinstall.npm_lifecycle_script",
				"Risky npm lifecycle script requires review",
				"Script "+name+" runs automatically on install; review for supply-chain risk.",
				issues.SanitizeSecretEvidence(script))...)
		}
		if curlPipeShell.MatchString(script) {
			out = append(out, makePreinstallFinding(repoRef, rel, 1, "high", 0.92,
				"supply_chain", "preinstall.curl_pipe_shell",
				"Possible risky install pattern (curl/wget piped to shell)",
				"Package script pipes remote content to a shell interpreter.",
				issues.SanitizeSecretEvidence(script))...)
		}
	}
	return out
}

func checkDockerfile(workspace, rel, repoRef string) []store.AuditFinding {
	safe, err := scanners.ValidateWorkspacePath(workspace, rel)
	if err != nil {
		return nil
	}
	content, err := os.ReadFile(filepath.Join(workspace, filepath.FromSlash(safe)))
	if err != nil {
		return nil
	}
	text := string(content)
	if curlPipeShell.MatchString(text) {
		return makePreinstallFinding(repoRef, rel, 1, "high", 0.9,
			"supply_chain", "preinstall.docker_remote_script",
			"Dockerfile runs remote script via curl/wget pipe",
			"Remote script execution during image build requires manual review.",
			issues.SanitizeSecretEvidence(firstMatchingLine(text, curlPipeShell)))
	}
	return nil
}

func checkWorkflow(workspace, rel, repoRef string) []store.AuditFinding {
	safe, err := scanners.ValidateWorkspacePath(workspace, rel)
	if err != nil {
		return nil
	}
	content, err := os.ReadFile(filepath.Join(workspace, filepath.FromSlash(safe)))
	if err != nil {
		return nil
	}
	text := string(content)
	if riskyWorkflow.MatchString(text) {
		severity := "medium"
		conf := 0.88
		if strings.Contains(strings.ToLower(text), "pull_request_target") {
			severity = "high"
			conf = 0.91
		}
		return makePreinstallFinding(repoRef, rel, 1, severity, conf,
			"supply_chain", "preinstall.workflow_permissions",
			"CI workflow uses elevated or risky permissions",
			"Workflow file contains permissions or triggers that may increase supply-chain risk.",
			issues.SanitizeSecretEvidence(firstMatchingLine(text, riskyWorkflow)))
	}
	return nil
}

func checkMissingLockfiles(workspace, repoRef string) []store.AuditFinding {
	var out []store.AuditFinding
	for manifest, lock := range lockfilePairs {
		manifestPath := findFile(workspace, manifest)
		if manifestPath == "" {
			continue
		}
		if manifest == "requirements.txt" {
			// Flag pip projects without poetry/pipfile lock nearby
			if findFile(workspace, "poetry.lock") == "" && findFile(workspace, "Pipfile.lock") == "" {
				out = append(out, makePreinstallFinding(repoRef, manifestPath, 1, "low", 0.8,
					"supply_chain", "preinstall.missing_lockfile",
					"Python project without lockfile",
					"Dependency versions may float; review requirements pinning before install.", "")...)
			}
			continue
		}
		if lock == manifest {
			continue
		}
		if findFile(workspace, lock) == "" {
			out = append(out, makePreinstallFinding(repoRef, manifestPath, 1, "low", 0.82,
				"supply_chain", "preinstall.missing_lockfile",
				"Missing lockfile for dependency manifest",
				"Manifest "+manifest+" has no companion "+lock+"; versions may be unpinned.", "")...)
		}
	}
	return out
}

func findFile(root, name string) string {
	var found string
	if err := filepath.WalkDir(root, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() || !d.Type().IsRegular() {
			return nil
		}
		if strings.EqualFold(filepath.Base(path), name) {
			rel, err := filepath.Rel(root, path)
			if err == nil {
				found = filepath.ToSlash(rel)
			}
			return filepath.SkipAll
		}
		return nil
	}); err != nil {
		return found
	}
	return found
}

func makePreinstallFinding(repoRef, file string, line int, severity string, confidence float64, category, ruleID, title, description, evidence string) []store.AuditFinding {
	evidence = issues.SanitizeSecretEvidence(evidence)
	fp := issues.ComputeFingerprint(issues.FingerprintInput{
		Repository:   repoRef,
		Category:     category,
		Source:       "preinstall",
		RuleID:       ruleID,
		File:         file,
		Line:         line,
		EvidenceHash: issues.SanitizedEvidenceHash(evidence),
	})
	return []store.AuditFinding{{
		Fingerprint:      fp,
		Category:         category,
		Severity:         severity,
		Confidence:       confidence,
		Source:           "preinstall",
		RuleID:           ruleID,
		FilePath:         file,
		Line:             line,
		Title:            title,
		EvidenceRedacted: firstNonEmpty(description, evidence),
	}}
}

func firstMatchingLine(text string, re *regexp.Regexp) string {
	sc := bufio.NewScanner(strings.NewReader(text))
	for sc.Scan() {
		line := sc.Text()
		if re.MatchString(line) {
			return line
		}
	}
	return ""
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}
