package scanners

// DeterministicRunResult records that an in-process deterministic stage ran during CAH scan.
// Used for health, static analysis, and graph stages so reconciliation can verify findings.
func DeterministicRunResult(scanner string, findingsCount int) RunResult {
	status := StatusClean
	if findingsCount > 0 {
		status = StatusFound
	}
	return RunResult{
		Scanner: scanner,
		Status:  status,
		Detail:  "deterministic pipeline stage",
	}
}
