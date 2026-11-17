package dogfood

// StaticcheckE2EMarker is a controlled S1039 target for remediation E2E testing.
// Repository Detective should replace fmt.Sprintf with a plain string literal.
func StaticcheckE2EMarker() string {
	return "repository-detective-staticcheck-e2e"
}
