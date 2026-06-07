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
	if commandAvailable("syft") {
		return generateSyftSBOM(ctx, dir, outDir)
	}
	if !hasSupportedManifest(dir) {
		return Result{Status: StatusNoSupportedManifest, Detail: "no supported dependency manifest detected"}, nil
	}
	return Result{Status: StatusToolMissing, Detail: "syft not installed — install syft or use a Go module repository"}, nil
}

func generateGoModuleSBOM(ctx context.Context, dir, outDir string) (Result, error) {
	outPath := filepath.Join(outDir, "sbom-go.cdx.json")
	if !commandAvailable("cyclonedx-gomod") {
		// Fallback: minimal SBOM note — operator can install cyclonedx-gomod or syft.
		if commandAvailable("syft") {
			return generateSyftSBOM(ctx, dir, outDir)
		}
		return Result{Status: StatusToolMissing, Detail: "cyclonedx-gomod and syft unavailable for Go SBOM"}, nil
	}
	cmd := exec.CommandContext(ctx, "cyclonedx-gomod", "mod", "-json", "-output", outPath)
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		return Result{Status: StatusCheckFailed, Detail: strings.TrimSpace(string(out))}, nil
	}
	return checkWithGrype(ctx, outPath, "CycloneDX")
}

func generateSyftSBOM(ctx context.Context, dir, outDir string) (Result, error) {
	outPath := filepath.Join(outDir, "sbom.syft.cdx.json")
	cmd := exec.CommandContext(ctx, "syft", dir, "-o", "cyclonedx-json", "--quiet")
	out, err := cmd.Output()
	if err != nil {
		return Result{Status: StatusCheckFailed, Detail: err.Error()}, nil
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
	cmd := exec.CommandContext(ctx, "grype", "sbom:"+sbomPath, "-o", "json", "--quiet")
	out, err := cmd.CombinedOutput()
	text := strings.TrimSpace(string(out))
	if err != nil && !strings.Contains(text, "{") {
		if strings.Contains(strings.ToLower(text), "failed to load vulnerability db") {
			res.Status = StatusCheckFailed
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
