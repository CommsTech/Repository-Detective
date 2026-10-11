package containers

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

// ScanOptions configures an image scan on a runner host.
type ScanOptions struct {
	Image          string
	PullPolicy     PullPolicy
	Tools          []string
	GenerateSBOM   bool
	TimeoutSeconds int
	WorkDir        string
}

// RunImageScan executes Trivy/Grype/Syft against an OCI image reference.
// Intended for native runner execution only (may invoke docker/trivy/grype/syft).
func RunImageScan(ctx context.Context, opts ScanOptions) (ScanResult, error) {
	started := time.Now().UTC()
	result := ScanResult{
		Image:     strings.TrimSpace(opts.Image),
		StartedAt: started,
		Coverage:  ScanCoverage{Trivy: "skipped", Grype: "skipped", Syft: "skipped"},
	}
	if result.Image == "" {
		return result, fmt.Errorf("image reference required")
	}
	timeout := time.Duration(opts.TimeoutSeconds) * time.Second
	if timeout <= 0 {
		timeout = 15 * time.Minute
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	toolSet := make(map[string]bool)
	for _, t := range opts.Tools {
		toolSet[strings.ToLower(strings.TrimSpace(t))] = true
	}
	if len(toolSet) == 0 {
		toolSet["trivy"] = true
		toolSet["grype"] = true
		toolSet["syft"] = true
	}

	if err := maybePullImage(ctx, result.Image, opts.PullPolicy); err != nil {
		result.Warnings = append(result.Warnings, RedactLogLine(err.Error()))
	}

	if toolSet["syft"] && opts.GenerateSBOM {
		if commandAvailable("syft") {
			if path, format, err := runSyftImage(ctx, result.Image, opts.WorkDir); err != nil {
				result.Coverage.Syft = "failed"
				result.Warnings = append(result.Warnings, RedactLogLine(err.Error()))
			} else {
				result.Coverage.Syft = "ok"
				result.SBOMPath = path
				result.SBOMFormat = format
				result.Findings = append(result.Findings, ScanFinding{
					RuleID: "CONTAINER-SBOM-GENERATED", Severity: "info", Confidence: 0.95,
					Title:       "SBOM generated for container image",
					Description: "Syft produced " + format + " SBOM for " + result.Image,
				})
			}
		} else {
			result.Coverage.Syft = "missing"
			result.Warnings = append(result.Warnings, "syft not installed on runner — SBOM skipped")
		}
	}

	if toolSet["trivy"] {
		if commandAvailable("trivy") {
			findings, err := runTrivyImage(ctx, result.Image)
			if err != nil {
				result.Coverage.Trivy = "failed"
				result.Warnings = append(result.Warnings, RedactLogLine(err.Error()))
			} else {
				result.Coverage.Trivy = "ok"
				result.Findings = append(result.Findings, findings...)
				result.VulnCount += len(findings)
			}
		} else {
			result.Coverage.Trivy = "missing"
			result.Warnings = append(result.Warnings, "trivy not installed on runner — image vuln scan skipped")
		}
	}

	if toolSet["grype"] {
		if commandAvailable("grype") {
			findings, err := runGrypeImage(ctx, result.Image)
			if err != nil {
				result.Coverage.Grype = "failed"
				result.Warnings = append(result.Warnings, RedactLogLine(err.Error()))
			} else {
				result.Coverage.Grype = "ok"
				result.Findings = append(result.Findings, findings...)
			}
		} else {
			result.Coverage.Grype = "missing"
			result.Warnings = append(result.Warnings, "grype not installed on runner — image vuln scan skipped")
		}
	}

	result.FinishedAt = time.Now().UTC()
	if result.Coverage.Trivy == "missing" && result.Coverage.Grype == "missing" && result.Coverage.Syft == "missing" {
		return result, fmt.Errorf("no container scan tools available on runner")
	}
	return result, nil
}

func maybePullImage(ctx context.Context, image string, policy PullPolicy) error {
	switch policy {
	case PullNever:
		return nil
	case PullAlways, PullIfMissing:
		if !commandAvailable("docker") {
			return nil
		}
		if policy == PullIfMissing {
			out, err := exec.CommandContext(ctx, "docker", "image", "inspect", image).CombinedOutput()
			if err == nil && len(out) > 0 {
				return nil
			}
		}
		out, err := exec.CommandContext(ctx, "docker", "pull", image).CombinedOutput()
		if err != nil {
			return fmt.Errorf("docker pull: %s", RedactLogLine(string(out)))
		}
	}
	return nil
}

func runTrivyImage(ctx context.Context, image string) ([]ScanFinding, error) {
	out, err := exec.CommandContext(ctx, "trivy", "image", "--format", "json", "--quiet", image).CombinedOutput()
	if err != nil && len(out) == 0 {
		return nil, err
	}
	return parseTrivyImageJSON(out, image)
}

func runGrypeImage(ctx context.Context, image string) ([]ScanFinding, error) {
	out, err := exec.CommandContext(ctx, "grype", image, "-o", "json").CombinedOutput()
	if err != nil && len(out) == 0 {
		return nil, err
	}
	return parseGrypeImageJSON(out, image)
}

func runSyftImage(ctx context.Context, image, workDir string) (path, format string, err error) {
	path = workDir + "/sbom.cdx.json"
	format = "cyclonedx-json"
	out, err := exec.CommandContext(ctx, "syft", image, "-o", "cyclonedx-json").CombinedOutput()
	if err != nil {
		return "", "", fmt.Errorf("syft: %s", RedactLogLine(string(out)))
	}
	if workDir != "" {
		if werr := writeFile(path, out); werr != nil {
			return "", format, werr
		}
	}
	return path, format, nil
}

func commandAvailable(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}

func writeFile(path string, data []byte) error {
	return os.WriteFile(path, data, 0o600)
}

// Simplified parsers — production uses full JSON structures.
func parseTrivyImageJSON(data []byte, image string) ([]ScanFinding, error) {
	text := string(data)
	if !strings.Contains(text, "VulnerabilityID") && !strings.Contains(text, "vulnerabilities") {
		return nil, nil
	}
	var out []ScanFinding
	if strings.Contains(text, "CRITICAL") || strings.Contains(text, "HIGH") {
		out = append(out, ScanFinding{
			RuleID: "CONTAINER-VULNERABLE-IMAGE", Severity: "high", Confidence: 0.88,
			Title:       "Vulnerable container image",
			Description: "Trivy reported vulnerabilities in " + image,
		})
	} else if strings.Contains(text, "MEDIUM") || strings.Contains(text, "LOW") {
		out = append(out, ScanFinding{
			RuleID: "CONTAINER-VULNERABLE-IMAGE", Severity: "medium", Confidence: 0.82,
			Title:       "Vulnerable container image",
			Description: "Trivy reported vulnerabilities in " + image,
		})
	}
	return out, nil
}

func parseGrypeImageJSON(data []byte, image string) ([]ScanFinding, error) {
	text := string(data)
	if !strings.Contains(text, "matches") {
		return nil, nil
	}
	if strings.Contains(text, `"severity":"Critical"`) || strings.Contains(text, `"severity":"High"`) {
		return []ScanFinding{{
			RuleID: "CONTAINER-VULNERABLE-IMAGE", Severity: "high", Confidence: 0.86,
			Title:       "Vulnerable container image",
			Description: "Grype reported vulnerabilities in " + image,
		}}, nil
	}
	return nil, nil
}
