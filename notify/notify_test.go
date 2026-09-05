package notify

import (
	"context"
	"strings"
	"testing"

	"git.commsnet.org/commstech/repository-detective/store"
)

type recordedPost struct {
	url     string
	body    []byte
	headers map[string]string
}

type mockPoster struct {
	posts []recordedPost
	code  int
	err   error
}

func newMockPoster(code int) *mockPoster {
	return &mockPoster{code: code}
}

func (m *mockPoster) Post(ctx context.Context, url string, contentType string, body []byte, headers map[string]string) (int, error) {
	cp := make(map[string]string, len(headers))
	for k, v := range headers {
		cp[k] = v
	}
	m.posts = append(m.posts, recordedPost{url: url, body: append([]byte(nil), body...), headers: cp})
	if m.err != nil {
		return 0, m.err
	}
	if m.code == 0 {
		m.code = 200
	}
	return m.code, nil
}

func (m *mockPoster) lastBody() []byte {
	if len(m.posts) == 0 {
		return nil
	}
	return m.posts[len(m.posts)-1].body
}

func testConfig() Config {
	cfg := DefaultConfig()
	cfg.Enabled = true
	cfg.TelegramEnabled = true
	cfg.TelegramBotToken = "test-token"
	cfg.TelegramChatID = "12345"
	cfg.SlackEnabled = true
	cfg.SlackWebhookURL = "https://hooks.slack.com/test"
	cfg.DiscordEnabled = true
	cfg.DiscordWebhookURL = "https://discord.com/api/webhooks/test"
	cfg.WebhookEnabled = true
	cfg.WebhookURL = "https://example.com/notify"
	cfg.WebhookSecret = "secret"
	cfg.PublicURL = "https://detective.example.com"
	return cfg
}

func TestSanitizeTextRedactsSecrets(t *testing.T) {
	in := "api_key=supersecret token: abc123 ghp_abcdefghijklmnopqrstuvwxyz"
	out := SanitizeText(in)
	if strings.Contains(out, "supersecret") || strings.Contains(out, "ghp_abc") {
		t.Fatalf("secret leaked: %q", out)
	}
}

func TestSanitizeTextTruncatesLongText(t *testing.T) {
	long := strings.Repeat("a", 5000)
	out := SanitizeText(long)
	if len(out) > maxMessageLen+4 {
		t.Fatalf("expected truncation, len=%d", len(out))
	}
}

func TestFormatMessageSafeWording(t *testing.T) {
	msg := FormatMessage(Event{
		Type:       EventHighFinding,
		Severity:   "high",
		Repository: "acme/demo",
		Title:      FindingSeverityWording("high"),
		Summary:    "3 findings across security scanners",
	})
	if msg.Text == "" {
		t.Fatal("expected message text")
	}
	if strings.Contains(msg.Text, "exploitable") || strings.Contains(msg.Text, "malware") {
		t.Fatal("unsafe wording in message")
	}
}

func TestPassesSeverityThreshold(t *testing.T) {
	if !PassesSeverityThreshold("high", "high") {
		t.Fatal("high should pass high threshold")
	}
	if PassesSeverityThreshold("low", "high") {
		t.Fatal("low should not pass high threshold")
	}
}

func TestManagerDisabledSuppresses(t *testing.T) {
	poster := newMockPoster(200)
	cfg := testConfig()
	cfg.Enabled = false
	m := NewManager(cfg, nil, nil, poster)
	m.Emit(context.Background(), 0, Event{Type: EventScanFailed, Severity: "high", Repository: "a/b"})
	if len(poster.posts) != 0 {
		t.Fatal("disabled manager should not post")
	}
}

func TestManagerEventFilter(t *testing.T) {
	poster := newMockPoster(200)
	cfg := testConfig()
	resolve := func(int64) EffectiveSettings {
		return EffectiveSettings{
			Enabled:         true,
			MinSeverity:     "high",
			CooldownSeconds: 0,
			Events:          map[string]bool{EventCriticalFinding: true},
		}
	}
	m := NewManager(cfg, resolve, nil, poster)
	m.Emit(context.Background(), 1, Event{Type: EventHighFinding, Severity: "high", Repository: "a/b"})
	if len(poster.posts) != 0 {
		t.Fatal("filtered event should not send")
	}
}

func TestManagerCooldown(t *testing.T) {
	poster := newMockPoster(200)
	cfg := testConfig()
	resolve := func(int64) EffectiveSettings {
		return EffectiveSettings{
			Enabled:         true,
			MinSeverity:     "low",
			CooldownSeconds: 3600,
			Events:          map[string]bool{EventScanFailed: true},
		}
	}
	m := NewManager(cfg, resolve, nil, poster)
	ev := Event{Type: EventScanFailed, Severity: "high", Repository: "a/b"}
	m.Emit(context.Background(), 1, ev)
	first := len(poster.posts)
	m.Emit(context.Background(), 1, ev)
	if first == 0 {
		t.Fatal("expected first emit")
	}
	if len(poster.posts) != first {
		t.Fatal("cooldown should suppress second emit")
	}
}

func TestTelegramPayload(t *testing.T) {
	poster := newMockPoster(200)
	ch := NewTelegramChannel(testConfig(), poster)
	if err := ch.Send(context.Background(), Message{Text: "hello"}); err != nil {
		t.Fatal(err)
	}
	body := string(poster.lastBody())
	if !strings.Contains(body, "hello") {
		t.Fatalf("unexpected telegram body: %s", body)
	}
	if strings.Contains(body, "test-token") {
		t.Fatal("token must not appear in payload body")
	}
}

func TestSlackPayload(t *testing.T) {
	poster := newMockPoster(200)
	ch := NewSlackChannel(testConfig(), poster)
	if err := ch.Send(context.Background(), Message{Text: "slack test"}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(poster.lastBody()), "slack test") {
		t.Fatal("expected slack text payload")
	}
}

func TestDiscordPayload(t *testing.T) {
	poster := newMockPoster(200)
	ch := NewDiscordChannel(testConfig(), poster)
	if err := ch.Send(context.Background(), Message{Text: "discord test"}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(poster.lastBody()), "discord test") {
		t.Fatal("expected discord content payload")
	}
}

func TestWebhookHMACSignature(t *testing.T) {
	poster := newMockPoster(200)
	cfg := testConfig()
	ch := NewWebhookChannel(cfg, poster)
	if err := ch.Send(context.Background(), Message{Text: "hook", Event: Event{Type: EventTest}}); err != nil {
		t.Fatal(err)
	}
	body := poster.lastBody()
	sig := SignWebhookBody(cfg.WebhookSecret, body)
	if len(poster.posts) == 0 || poster.posts[0].headers[signatureHeader] != sig {
		t.Fatalf("signature mismatch header=%q expected=%q", poster.posts[0].headers[signatureHeader], sig)
	}
}

func TestWebhookHTTPError(t *testing.T) {
	poster := newMockPoster(500)
	ch := NewWebhookChannel(testConfig(), poster)
	if err := ch.Send(context.Background(), Message{Text: "hook", Event: Event{Type: EventTest}}); err == nil {
		t.Fatal("expected HTTP error")
	}
}

func TestSendTest(t *testing.T) {
	poster := newMockPoster(200)
	m := NewManager(testConfig(), nil, nil, poster)
	if err := m.SendTest(context.Background()); err != nil {
		t.Fatal(err)
	}
	if len(poster.posts) == 0 {
		t.Fatal("expected test posts")
	}
}

func TestValidateEventCSV(t *testing.T) {
	if err := ValidateEventCSV("critical_finding,scan_failed"); err != nil {
		t.Fatal(err)
	}
	if err := ValidateEventCSV("not_real"); err == nil {
		t.Fatal("expected invalid event error")
	}
}

func TestResolveEffectiveRepoOverride(t *testing.T) {
	global := testConfig()
	global.Enabled = true
	disabled := false
	sev := "critical"
	settings := store.RepoSettings{
		NotificationsEnabled:    &disabled,
		NotificationMinSeverity: &sev,
	}
	eff := ResolveEffective(global, settings)
	if eff.Enabled {
		t.Fatal("repo disabled notifications")
	}
	if eff.MinSeverity != "critical" {
		t.Fatalf("expected critical, got %s", eff.MinSeverity)
	}
}

func TestLowFindingNotInDefaultEvents(t *testing.T) {
	events := parseEventSet(DefaultEnabledEvents)
	if events[EventHighFinding] && events["low_finding"] {
		t.Fatal("low finding should not be default")
	}
}
