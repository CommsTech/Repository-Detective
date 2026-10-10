package ai_test

import (
	"context"
	"testing"
	"time"

	"git.commsnet.org/commstech/repository-detective/ai"
)

type fakeTransport struct{}

func (fakeTransport) Name() string { return "fake" }
func (fakeTransport) Complete(context.Context, ai.ChatRequest) (*ai.ChatResponse, error) {
	return &ai.ChatResponse{Content: "ok"}, nil
}

func TestStartupSkipsChatWhenMetadataMode(t *testing.T) {
	ai.ConfigureStatusCache(60)
	client := ai.NewClientWithTransport(fakeTransport{}, "test", nil)
	st, err := ai.RunConnectionTest(context.Background(), client, ai.TestModeMetadataOnly, true)
	if err == nil {
		t.Fatal("expected metadata test unsupported for fake transport")
	}
	if st.TestMode != ai.TestModeMetadataOnly {
		t.Fatalf("mode %s", st.TestMode)
	}
}

func TestConnectionTestCache(t *testing.T) {
	ai.ConfigureStatusCache(60)
	client := ai.NewClientWithTransport(fakeTransport{}, "test", nil)
	_, _ = ai.RunConnectionTest(context.Background(), client, ai.TestModeChatCompletion, true)
	st1, _ := ai.RunConnectionTest(context.Background(), client, ai.TestModeChatCompletion, false)
	time.Sleep(10 * time.Millisecond)
	st2, _ := ai.RunConnectionTest(context.Background(), client, ai.TestModeChatCompletion, false)
	if st1.LastTestAt == nil || st2.LastTestAt == nil {
		t.Fatal("expected test timestamps")
	}
	if !st1.LastTestOK || !st2.LastTestOK {
		t.Fatal("expected cached ok status")
	}
}
