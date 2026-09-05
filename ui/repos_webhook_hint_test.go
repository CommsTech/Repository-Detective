package ui

import "testing"

func TestWebhookRowHint(t *testing.T) {
	ready := WebhookSetupStatus{Ready: true}
	if got := webhookRowHint(false, "", ready); got != "" {
		t.Fatalf("disabled repo should have no hint, got %q", got)
	}
	if got := webhookRowHint(true, "2026-08-02 12:00", ready); got != "" {
		t.Fatalf("enabled with webhook should have no hint, got %q", got)
	}
	if got := webhookRowHint(true, "", ready); got == "" {
		t.Fatal("enabled without webhook should hint")
	}
	notReady := WebhookSetupStatus{Ready: false}
	if got := webhookRowHint(true, "", notReady); got == "" {
		t.Fatal("enabled with incomplete webhook setup should hint")
	}
}

func TestWebhookSetupStatusIssues(t *testing.T) {
	h := &Handler{basePath: "/ui"}
	h.platform = PlatformContext{}
	status := h.webhookSetupStatus()
	if status.Ready {
		t.Fatal("expected not ready when public URL/token/secret missing")
	}
	if len(status.Issues) < 2 {
		t.Fatalf("expected multiple issues, got %v", status.Issues)
	}
	if status.OnboardURL != "/onboard" {
		t.Fatalf("onboard url=%q", status.OnboardURL)
	}

	h.platform = PlatformContext{
		PublicURL:               "https://rd.example.com",
		WebhookSecretConfigured: true,
		GiteaURLConfigured:      true,
		GiteaTokenConfigured:    true,
	}
	status = h.webhookSetupStatus()
	if !status.Ready || len(status.Issues) != 0 {
		t.Fatalf("expected ready, got ready=%v issues=%v", status.Ready, status.Issues)
	}
}
