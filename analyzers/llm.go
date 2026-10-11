package analyzers

// LLMEnabled reports whether LLM-backed pipeline stages should run.
func LLMEnabled(cfg *Config, aiAvailable bool) bool {
	if cfg == nil || !aiAvailable {
		return false
	}
	depth := cfg.AnalysisDepth
	if depth <= 0 {
		depth = 3
	}
	return depth >= 3 && cfg.EnableLLMAuditors
}
