package sbom

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"git.commsnet.org/commstech/repository-detective/internal/security"
)

// Status describes SBOM generation and vulnerability check outcomes.
type Status string

const (
	StatusGenerated          Status = "sbom_generated"
	StatusNoSupportedManifest Status = "sbom_no_supported_manifest"
	StatusToolMissing        Status = "sbom_tool_missing"
	StatusCheckClean         Status = "sbom_check_clean"
	StatusVulnerabilitiesFound Status = "sbom_vulnerabilities_found"
	StatusCheckFailed        Status = "sbom_check_failed"
)

// Result is the outcome of generate + optional grype check.
type Result struct {
	Status       Status `json:"status"`
	Format       string `json:"format,omitempty"`
	PackageCount int    `json:"package_count,omitempty"`
	VulnCount    int    `json:"vuln_count,omitempty"`
	Detail       string `json:"detail,omitempty"`
	ArtifactPath string `json:"artifact_path,omitempty"`
}

type cycloneDXDoc struct {
	Components []struct {
		Name string `json:"name"`
	} `json:"components"`
	Matches []json.RawMessage `json:"matches"`
}

// GenerateAndCheck creates an SBOM for dir and runs grype when available.
func GenerateAndCheck(ctx context.Context, dir, outDir string) (Result, error) {
	dir = strings.TrimSpace(dir)
	if dir == "" {
		return Result{Status: StatusNoSupportedManifest, Detail: "empty workspace directory"}, nil
	}
	if outDir == "" {
		outDir = dir
	}
	_ = os.MkdirAll(outDir, 0o755)

	if hasGoModule(dir) {
		return generateGoModuleSBOM(ctx, dir, outDir)
	}
	if !hasSupportedManifest(dir) {
		return Result{Status: StatusNoSupportedManifest, Detail: "no supported dependency manifest detected"}, nil
	}
	if commandAvailable("syft") {
		return generateSyftSBOM(ctx, dir, outDir)
	}
	return Result{Status: StatusToolMissing, Detail: "syft not installed — install syft or use a Go module repository"}, nil
}

func generateGoModuleSBOM(ctx context.Context, dir, outDir string) (Result, error) {
	outPath := filepath.Join(outDir, "sbom-go.cdx.json")
	if !commandAvailable("cyclonedx-gomod") {
		if commandAvailable("syft") {
			return generateSyftSBOM(ctx, dir, outDir)
		}
		return Result{Status: StatusToolMissing, Detail: "cyclonedx-gomod and syft unavailable for Go SBOM"}, nil
	}
	cmd := exec.CommandContext(ctx, "cyclonedx-gomod", "mod", "-json", "-output", outPath)
	cmd.Dir = dir
	cmd.Env = append(security.MinimalSubprocessEnv(), "GOTOOLCHAIN=auto")
	if out, err := cmd.CombinedOutput(); err != nil {
		// Prefer Syft when the Go toolchain cannot satisfy module requirements
		// (common when the container Go is older than go.mod).
		if commandAvailable("syft") {
			res, syftErr := generateSyftSBOM(ctx, dir, outDir)
			// Accept any Syft result that produced an artifact — grype DB issues
			// still leave a usable SBOM (StatusCheckFailed with ArtifactPath).
			if syftErr == nil && strings.TrimSpace(res.ArtifactPath) != "" {
				prefix := "generated with syft after cyclonedx-gomod failed"
				if strings.TrimSpace(res.Detail) == "" {
					res.Detail = prefix
				} else if !strings.Contains(res.Detail, "syft") {
					res.Detail = prefix + "; " + res.Detail
				}
				return res, nil
			}
		}
		return Result{Status: StatusCheckFailed, Detail: redactSBOMDetail(string(out))}, nil
	}
	return checkWithGrype(ctx, outPath, "CycloneDX")
}

func redactSBOMDetail(detail string) string {
	// Strip ANSI color codes from tool logs (syft / cyclonedx-gomod).
	var b strings.Builder
	s := detail
	for i := 0; i < len(s); i++ {
		if s[i] == 0x1b {
			for i < len(s) && s[i] != 'm' {
				i++
			}
			continue
		}
		b.WriteByte(s[i])
	}
	clean := strings.Join(strings.Fields(b.String()), " ")
	if len(clean) > 400 {
		return clean[:397] + "..."
	}
	return clean
}

func generateSyftSBOM(ctx context.Context, dir, outDir string) (Result, error) {
	outPath := filepath.Join(outDir, "sbom.syft.cdx.json")
	cmd := exec.CommandContext(ctx, "syft", dir, "-o", "cyclonedx-json", "--quiet")
	out, err := cmd.Output()
	if err != nil {
		detail := err.Error()
		if ee, ok := err.(*exec.ExitError); ok && len(ee.Stderr) > 0 {
			detail = redactSBOMDetail(string(ee.Stderr))
		}
		return Result{Status: StatusCheckFailed, Detail: detail}, nil
	}
	if err := os.WriteFile(outPath, out, 0o644); err != nil {
		return Result{}, err
	}
	var doc cycloneDXDoc
	_ = json.Unmarshal(out, &doc)
	return checkWithGrype(ctx, outPath, "CycloneDX", len(doc.Components))
}

func checkWithGrype(ctx context.Context, sbomPath, format string, pkgCount ...int) (Result, error) {
	count := 0
	if len(pkgCount) > 0 {
		count = pkgCount[0]
	} else if raw, err := os.ReadFile(sbomPath); err == nil {
		var doc cycloneDXDoc
		if json.Unmarshal(raw, &doc) == nil {
			count = len(doc.Components)
		}
	}
	res := Result{
		Status:       StatusGenerated,
		Format:       format,
		PackageCount: count,
		ArtifactPath: sbomPath,
	}
	if !commandAvailable("grype") {
		res.Detail = "SBOM generated; grype unavailable for vulnerability check"
		return res, nil
	}
	cleanSBOM := filepath.Clean(sbomPath)
	cleanDir := filepath.Clean(filepath.Dir(cleanSBOM))
	if cleanDir == "" || cleanDir == "." {
		return Result{Status: StatusCheckFailed, Detail: "invalid sbom path"}, nil
	}
	grypeArgs := []string{"sbom:" + cleanSBOM, "-o", "json", "--quiet"}
	cmd := exec.CommandContext(ctx, "grype", grypeArgs...)
	cmd.Env = append(security.MinimalSubprocessEnv(), "GOTOOLCHAIN=auto")
	if os.Getenv("HOME") == "" {
		cmd.Env = append(cmd.Env, "HOME=/home/repositorydetective", "XDG_CACHE_HOME=/home/repositorydetective/.cache")
	}
	out, err := cmd.CombinedOutput()
	text := strings.TrimSpace(string(out))
	if err != nil && !strings.Contains(text, "{") {
		if strings.Contains(strings.ToLower(text), "failed to load vulnerability db") ||
			strings.Contains(strings.ToLower(text), "database disk image is malformed") {
			res.Status = StatusGenerated
			res.Detail = "SBOM generated; grype vulnerability DB unavailable"
			return res, nil
		}
		res.Status = StatusCheckFailed
		res.Detail = text
		return res, nil
	}
	var report cycloneDXDoc
	if err := json.Unmarshal(extractJSON(out), &report); err != nil {
		res.Status = StatusCheckFailed
		res.Detail = err.Error()
		return res, nil
	}
	vulns := len(report.Matches)
	res.VulnCount = vulns
	if vulns > 0 {
		res.Status = StatusVulnerabilitiesFound
		res.Detail = fmt.Sprintf("%d vulnerabilities in SBOM", vulns)
	} else {
		res.Status = StatusCheckClean
		res.Detail = "SBOM vulnerability check clean"
	}
	return res, nil
}

func hasGoModule(dir string) bool {
	_, err := os.Stat(filepath.Join(dir, "go.mod"))
	return err == nil
}

func hasSupportedManifest(dir string) bool {
	names := []string{"go.mod", "package.json", "requirements.txt", "Pipfile", "poetry.lock", "Gemfile.lock", "pom.xml", "Cargo.lock"}
	for _, n := range names {
		if _, err := os.Stat(filepath.Join(dir, n)); err == nil {
			return true
		}
	}
	return false
}

func commandAvailable(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}

func extractJSON(b []byte) []byte {
	s := string(b)
	start := strings.Index(s, "{")
	end := strings.LastIndex(s, "}")
	if start < 0 || end <= start {
		return nil
	}
	return []byte(s[start : end+1])
}

// DefaultTimeout returns a conservative SBOM operation timeout.
func DefaultTimeout() time.Duration { return 5 * time.Minute }
