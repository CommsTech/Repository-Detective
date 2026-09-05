package analyzers_test

import (
	"testing"

	"git.commsnet.org/commstech/repository-detective/analyzers"
)

func TestLLMEnabledMatrix(t *testing.T) {
	cfg := &analyzers.Config{AnalysisDepth: 1, EnableLLMAuditors: false}
	if analyzers.LLMEnabled(cfg, true) {
		t.Fatal("depth 1 should disable LLM")
	}

	cfg = &analyzers.Config{AnalysisDepth: 3, EnableLLMAuditors: false}
	if analyzers.LLMEnabled(cfg, true) {
		t.Fatal("enable_llm_auditors=false should disable LLM")
	}

	cfg = &analyzers.Config{AnalysisDepth: 3, EnableLLMAuditors: true}
	if analyzers.LLMEnabled(cfg, false) {
		t.Fatal("missing AI client should disable LLM")
	}

	if !analyzers.LLMEnabled(cfg, true) {
		t.Fatal("depth 3 with auditors enabled should allow LLM")
	}
}
