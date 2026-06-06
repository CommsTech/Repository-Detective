package dogfood

import "fmt"

// StaticcheckE2EMarker is a controlled S1039 target for remediation E2E testing.
// Repository Detective should replace fmt.Sprintf with a plain string literal.
func StaticcheckE2EMarker() string {
	return fmt.Sprintf("repository-detective-staticcheck-e2e")
}
