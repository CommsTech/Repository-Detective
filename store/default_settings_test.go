package store_test

import (
	"testing"

	"git.commsnet.org/commstech/repository-detective/store"
)

func TestDefaultGlobalSettingsAIOptional(t *testing.T) {
	g := store.DefaultGlobalSettings()
	if g.EnableLLMAuditors {
		t.Fatal("DefaultGlobalSettings must leave LLM auditors off (AI optional)")
	}
	if g.AIPolicy != store.AIPolicyDisabled {
		t.Fatalf("DefaultGlobalSettings AIPolicy=%q want %q", g.AIPolicy, store.AIPolicyDisabled)
	}
}
