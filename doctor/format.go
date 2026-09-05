package doctor

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"git.commsnet.org/commstech/repository-detective/internal/security"
)

// FormatHuman writes an operator-facing doctor report.
func FormatHuman(w io.Writer, r Report) error {
	r = RedactReport(r)
	_, _ = fmt.Fprintf(w, "Repository Detective Doctor\n")
	_, _ = fmt.Fprintf(w, "Overall: %s\n", r.Overall)
	if r.OnboardingReady != "" {
		_, _ = fmt.Fprintf(w, "Onboarding readiness: %s\n", r.OnboardingReady)
	}
	_, _ = fmt.Fprintf(w, "Summary: %s\n", r.Summary)
	if r.Version != "" {
		_, _ = fmt.Fprintf(w, "Build: version=%s commit=%s edition=%s\n", r.Version, r.Commit, r.Edition)
	}
	_, _ = fmt.Fprintf(w, "Generated: %s\n\n", r.GeneratedAt.Format(timeRFC3339))
	currentCat := ""
	for _, c := range r.Checks {
		if c.Category != currentCat {
			currentCat = c.Category
			_, _ = fmt.Fprintf(w, "== %s ==\n", strings.ToUpper(currentCat))
		}
		req := ""
		if c.Required {
			req = " [required]"
		}
		_, _ = fmt.Fprintf(w, "  [%s]%s %s\n", c.State, req, c.Summary)
		if c.Detail != "" {
			_, _ = fmt.Fprintf(w, "           %s\n", c.Detail)
		}
		if c.Remediation != "" {
			_, _ = fmt.Fprintf(w, "           → %s\n", c.Remediation)
		}
	}
	_, _ = fmt.Fprintf(w, "\nThese results describe system readiness, not whether repositories are safe or secure.\n")
	return nil
}

const timeRFC3339 = "2006-01-02T15:04:05Z07:00"

// FormatJSON writes indented JSON with secrets redacted from string fields.
func FormatJSON(w io.Writer, r Report) error {
	cleaned := RedactReport(r)
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(cleaned)
}

// RedactReport deep-copies and redacts secret-like material from check details.
func RedactReport(r Report) Report {
	out := r
	out.Summary = sanitizeDetail(r.Summary)
	out.Checks = make([]Check, len(r.Checks))
	for i, c := range r.Checks {
		c.Summary = sanitizeDetail(c.Summary)
		c.Detail = sanitizeDetail(c.Detail)
		c.Remediation = sanitizeDetail(c.Remediation)
		out.Checks[i] = c
	}
	return out
}

// SupportBundle is a safe export for operators (RD-014).
type SupportBundle struct {
	Report          Report            `json:"report"`
	SanitizedConfig map[string]string `json:"sanitized_config"`
	ScannerVersions map[string]string `json:"scanner_versions"`
	RecentErrors    []string          `json:"recent_errors,omitempty"`
	Notes           []string          `json:"notes"`
}

// BuildSupportBundle creates a redacted diagnostic export.
func BuildSupportBundle(r Report, sanitizedConfig map[string]string, recentErrors []string) SupportBundle {
	r = RedactReport(r)
	versions := map[string]string{}
	for _, c := range r.Checks {
		if strings.HasPrefix(c.ID, "scanner.") && strings.HasSuffix(c.ID, ".available") {
			versions[c.ID] = c.Detail
		}
	}
	cfg := map[string]string{}
	for k, v := range sanitizedConfig {
		lk := strings.ToLower(k)
		if strings.Contains(lk, "token") || strings.Contains(lk, "secret") || strings.Contains(lk, "password") || strings.Contains(lk, "api_key") || strings.Contains(lk, "apikey") {
			cfg[k] = "[REDACTED]"
			continue
		}
		cfg[k] = security.RedactSecrets(v)
	}
	var errs []string
	for _, e := range recentErrors {
		errs = append(errs, sanitizeDetail(e))
	}
	return SupportBundle{
		Report:          r,
		SanitizedConfig: cfg,
		ScannerVersions: versions,
		RecentErrors:    errs,
		Notes: []string{
			"Support bundle excludes API keys, tokens, webhook secrets, session secrets, source code, diffs, and secret finding evidence.",
			"Results describe readiness, not repository security.",
		},
	}
}
