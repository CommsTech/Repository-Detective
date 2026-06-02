package scanners

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/sirupsen/logrus"
)

func init() {
	RegisterDeterministicSource("govulncheck")
}

// RunGovulncheck scans Go modules for known vulnerabilities using govulncheck.
func RunGovulncheck(ctx context.Context, logger *logrus.Logger, dir string, cfg Config) RunResult {
	return runGovulncheckWithCommand(ctx, logger, dir, cfg, "govulncheck")
}

func runGovulncheckWithCommand(ctx context.Context, logger *logrus.Logger, dir string, cfg Config, commandName string) RunResult {
	result := RunResult{Scanner: "govulncheck"}
	if !commandAvailable(commandName) {
		if logger != nil {
			logger.Warn("[SCANNER:govulncheck] binary not found — install govulncheck (go install golang.org/x/vuln/cmd/govulncheck@latest)")
		}
		result.Status = StatusBinaryMissing
		return result
	}

	timeout := scannerTimeoutSeconds(cfg.GovulncheckTimeoutSeconds, cfg.TimeoutSeconds)
	output, err := runCommand(ctx, timeout, dir, commandName, "-json", "./...")
	if err != nil && len(output) == 0 {
		result.Status = classifyCommandError(err)
		result.Detail = err.Error()
		return result
	}

	parsed, parseErr := parseGovulncheckOutput(output, dir, cfg)
	if parseErr != nil && len(parsed.Findings) == 0 {
		result.Status = StatusParseFailed
		result.Detail = parseErr.Error()
		return result
	}

	result = resultWithFindings("govulncheck", parsed.Findings)
	if parsed.Truncated {
		max := goScannerMaxFindings(cfg)
		result.Detail = truncateDetailForScanner("findings", max, parsed.Total)
	}
	if logger != nil {
		logResultInfo(logger, "govulncheck", result.Status, len(parsed.Findings), result.Detail)
	}
	return result
}

func parseGovulncheckOutput(output []byte, dir string, cfg Config) (cappedFindings, error) {
	var findings []Finding
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		var msg govulncheckMessage
		if err := json.Unmarshal([]byte(line), &msg); err != nil {
			continue
		}
		if msg.Finding == nil {
			continue
		}
		findings = append(findings, govulncheckFinding(*msg.Finding, dir))
	}
	if len(findings) == 0 && len(output) > 0 && !looksLikeGovulncheckOutput(output) {
		return cappedFindings{}, fmt.Errorf("no govulncheck findings parsed from output")
	}
	capped := capFindings(findings, goScannerMaxFindings(cfg))
	return capped, nil
}

type govulncheckMessage struct {
	Finding *govulncheckFindingJSON `json:"finding"`
}

type govulncheckFindingJSON struct {
	OSV          json.RawMessage          `json:"osv"`
	FixedVersion string                   `json:"fixed_version"`
	Trace        []govulncheckTraceFrame  `json:"trace"`
	Symbol       string                   `json:"symbol"`
	Module       string                   `json:"module"`
	Package      string                   `json:"package"`
}

type govulncheckTraceFrame struct {
	Function string `json:"function"`
	Position struct {
		Filename string `json:"filename"`
		Line     int    `json:"line"`
	} `json:"position"`
}

func govulncheckFinding(raw govulncheckFindingJSON, dir string) Finding {
	osvID := extractOSVID(raw.OSV)
	pkg := firstNonEmpty(raw.Module, raw.Package)
	title := "Go vulnerability " + osvID
	if pkg != "" {
		title = fmt.Sprintf("Go vulnerability %s in %s", osvID, pkg)
	}
	file, line := "", 0
	var traceParts []string
	for _, frame := range raw.Trace {
		if frame.Function != "" {
			traceParts = append(traceParts, frame.Function)
		}
		if file == "" && frame.Position.Filename != "" {
			file = relPath(dir, frame.Position.Filename)
			line = frame.Position.Line
		}
	}
	if raw.Symbol != "" {
		traceParts = append([]string{raw.Symbol}, traceParts...)
	}
	desc := strings.Join(traceParts, " → ")
	if raw.FixedVersion != "" {
		desc = trimEvidence("fixed in " + raw.FixedVersion + "; " + desc)
	}
	return Finding{
		Source:      "govulncheck",
		Category:    "dependency",
		Severity:    "high",
		Title:       title,
		Description: trimEvidence(desc),
		File:        file,
		Line:        line,
		Confidence:  0.95,
		Reference:   osvID,
		ID:          osvID,
	}
}

func extractOSVID(raw json.RawMessage) string {
	if len(raw) == 0 {
		return "GO-UNKNOWN"
	}
	var asString string
	if err := json.Unmarshal(raw, &asString); err == nil && asString != "" {
		return asString
	}
	var obj struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(raw, &obj); err == nil && obj.ID != "" {
		return obj.ID
	}
	return "GO-UNKNOWN"
}

func looksLikeGovulncheckOutput(output []byte) bool {
	return strings.Contains(string(output), `"finding"`) || strings.Contains(string(output), `"config"`)
}

func relPath(dir, abs string) string {
	abs = filepath.Clean(abs)
	if dir != "" {
		if rel, err := filepath.Rel(dir, abs); err == nil {
			return filepath.ToSlash(rel)
		}
	}
	return filepath.ToSlash(abs)
}

const maxGoEvidenceLen = 500

func trimEvidence(value string) string {
	value = strings.TrimSpace(value)
	if len(value) > maxGoEvidenceLen {
		return value[:maxGoEvidenceLen] + "..."
	}
	return value
}

func ParseGovulncheckOutputForTest(output []byte, dir string, cfg Config) (cappedFindings, error) {
	return parseGovulncheckOutput(output, dir, cfg)
}
