// e2e acceptance fixtures — synthetic only, never real credentials.
package e2efixtures

// SyntheticGitleaksSecret is a deliberately fake AWS-shaped token for Gitleaks.
// Built at runtime in tests/harness so static scanners do not flag this file.
func SyntheticGitleaksSecret() string {
	return "AKI" + "A" + "0TEST" + "SECRET" + "FIXTURE00" + "EXAMPLE"
}

// SyntheticSemgrepSQL is a trivial SQL concatenation pattern for Semgrep/rules.
const SyntheticSemgrepSQL = `package vuln

import "database/sql"

func Lookup(db *sql.DB, userInput string) (*sql.Rows, error) {
	q := "SELECT * FROM users WHERE name = '" + userInput + "'"
	return db.Query(q)
}
`

// VulnerableRequirements is a pinned historical urllib3 pin used for dependency scanners.
// Exact CVE text may drift; assert scanner SUCCESS + dependency evidence, not brittle CVE IDs.
const VulnerableRequirements = `urllib3==1.26.4
requests==2.25.1
`
