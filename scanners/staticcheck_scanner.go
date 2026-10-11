package scanners

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/sirupsen/logrus"
)

func init() {
	RegisterDeterministicSource("staticcheck")
}

type staticcheckMessage struct {
	Code     string `json:"code"`
	Message  string `json:"message"`
	Location struct {
		File   string `json:"file"`
		Line   int    `json:"line"`
		Column int    `json:"column"`
	} `json:"location"`
}

// RunStaticcheck scans Go code using staticcheck as a first-class scanner.
func RunStaticcheck(ctx context.Context, logger *logrus.Logger, dir string, cfg Config) RunResult {
	return runStaticcheckWithCommand(ctx, logger, dir, cfg, "staticcheck")
}

func runStaticcheckWithCommand(ctx context.Context, logger *logrus.Logger, dir string, cfg Config, commandName string) RunResult {
	result := RunResult{Scanner: "staticcheck"}
	if !commandAvailable(commandName) {
		if logger != nil {
			logger.Warn("[SCANNER:staticcheck] binary not found — install staticcheck (go install honnef.co/go/tools/cmd/staticcheck@latest)")
		}
		result.Status = StatusBinaryMissing
		return result
	}

	timeout := scannerTimeoutSeconds(cfg.StaticcheckTimeoutSeconds, cfg.TimeoutSeconds)
	output, err := runCommand(ctx, timeout, dir, commandName, "-f", "json", "./...")
	if err != nil && len(output) == 0 {
		result.Status = classifyCommandError(err)
		result.Detail = err.Error()
		return result
	}

	parsed, parseErr := parseStaticcheckOutput(output, dir, cfg)
	if parseErr != nil && len(parsed.Findings) == 0 {
		msg := strings.ToLower(parseErr.Error())
		if strings.Contains(msg, "compile") ||
			strings.Contains(msg, "requires at least go") ||
			strings.Contains(msg, "built with go") {
			result.Status = StatusScannerUnavailable
			result.Detail = parseErr.Error()
			return result
		}
		result.Status = StatusParseFailed
		result.Detail = parseErr.Error()
		return result
	}

	result = resultWithFindings("staticcheck", parsed.Findings)
	if parsed.Truncated {
		max := goScannerMaxFindings(cfg)
		result.Detail = truncateDetailForScanner("findings", max, parsed.Total)
	}
	if logger != nil {
		logResultInfo(logger, "staticcheck", result.Status, len(parsed.Findings), result.Detail)
	}
	return result
}

func parseStaticcheckOutput(output []byte, dir string, cfg Config) (cappedFindings, error) {
	clean := string(stripANSI(output))
	var findings []Finding
	parsedAny := false
	for _, line := range strings.Split(clean, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || !strings.HasPrefix(line, "{") {
			continue
		}
		var msg staticcheckMessage
		if err := json.Unmarshal([]byte(line), &msg); err != nil {
			continue
		}
		if msg.Code == "" {
			continue
		}
		if strings.EqualFold(msg.Code, "compile") {
			detail := strings.TrimSpace(msg.Message)
			if detail == "" {
				detail = "staticcheck compile error"
			}
			return cappedFindings{}, fmt.Errorf("%s", detail)
		}
		parsedAny = true
		findings = append(findings, staticcheckFinding(msg, dir))
	}
	if !parsedAny && len(strings.TrimSpace(clean)) > 0 && strings.TrimSpace(clean) != "[]" {
		lower := strings.ToLower(clean)
		if strings.Contains(lower, "requires at least go") ||
			strings.Contains(lower, "built with go") ||
			strings.Contains(lower, "compile") {
			return cappedFindings{}, fmt.Errorf("%s", firstNonEmptyLine(clean))
		}
		return cappedFindings{}, fmt.Errorf("no staticcheck findings parsed from output")
	}
	capped := capFindings(findings, goScannerMaxFindings(cfg))
	return capped, nil
}

func staticcheckFinding(msg staticcheckMessage, dir string) Finding {
	code := strings.TrimSpace(msg.Code)
	category, severity, confidence := mapStaticcheckCode(code)
	title := msg.Message
	if title == "" {
		title = code
	}
	return Finding{
		Source:      "staticcheck",
		Category:    category,
		Severity:    severity,
		Title:       title,
		Description: trimEvidence(msg.Message),
		File:        relPath(dir, msg.Location.File),
		Line:        msg.Location.Line,
		Confidence:  confidence,
		Reference:   code,
		ID:          code,
	}
}

func mapStaticcheckCode(code string) (category, severity string, confidence float64) {
	switch {
	case strings.HasPrefix(code, "SA"):
		return "reliability", "medium", 0.90
	case strings.HasPrefix(code, "ST"):
		return "maintainability", "low", 0.80
	case strings.HasPrefix(code, "QF"):
		return "code_quality", "low", 0.80
	case strings.HasPrefix(code, "S"):
		return "code_quality", "low", 0.90
	default:
		return "code_quality", "low", 0.80
	}
}

func ParseStaticcheckOutputForTest(output []byte, dir string, cfg Config) (cappedFindings, error) {
	return parseStaticcheckOutput(output, dir, cfg)
}

func firstNonEmptyLine(text string) string {
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			return line
		}
	}
	return strings.TrimSpace(text)
}
