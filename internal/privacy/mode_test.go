package privacy_test

import (
	"testing"

	"git.commsnet.org/commstech/repository-detective/internal/privacy"
)

func TestClassifyLoopbackAndPrivate(t *testing.T) {
	cases := map[string]string{
		"127.0.0.1":   privacy.ClassLocal,
		"localhost":   privacy.ClassLocal,
		"10.1.2.3":    privacy.ClassLocal,
		"192.168.1.1": privacy.ClassLocal,
		"172.16.0.1":  privacy.ClassLocal,
		"8.8.8.8":     privacy.ClassExternal,
		"::1":         privacy.ClassLocal,
	}
	for host, want := range cases {
		if got := privacy.ClassifyHost(host); got != want {
			t.Fatalf("%s: got %s want %s", host, got, want)
		}
	}
}

func TestLocalOnlyRejectsCloudProvider(t *testing.T) {
	d := privacy.EvaluateAIEgress(privacy.ModeLocalOnly, "openai", "https://api.openai.com/v1", true)
	if d.Allowed {
		t.Fatalf("expected reject: %+v", d)
	}
}

func TestLocalOnlyAllowsPrivateOllama(t *testing.T) {
	d := privacy.EvaluateAIEgress(privacy.ModeLocalOnly, "ollama", "http://10.0.0.5:11434", true)
	if !d.Allowed || d.EndpointClass != privacy.ClassLocal {
		t.Fatalf("expected allow local ollama: %+v", d)
	}
}

func TestLocalOnlyRejectsPublicOllamaURL(t *testing.T) {
	d := privacy.EvaluateAIEgress(privacy.ModeLocalOnly, "ollama", "http://8.8.8.8:11434", true)
	if d.Allowed {
		t.Fatalf("public IP must be rejected in LOCAL_ONLY: %+v", d)
	}
}

func TestProviderNameAloneNotTrusted(t *testing.T) {
	// ollama provider with external IP is still external
	d := privacy.EvaluateAIEgress(privacy.ModeLocalOnly, "ollama", "https://example.com", true)
	if d.Allowed && d.EndpointClass == privacy.ClassLocal {
		t.Fatal("must not classify hostname as local without resolution proof")
	}
}

func TestDisabledAIAllowedInLocalOnly(t *testing.T) {
	d := privacy.EvaluateAIEgress(privacy.ModeLocalOnly, "", "", false)
	if !d.Allowed {
		t.Fatalf("disabled AI must be allowed under LOCAL_ONLY: %+v", d)
	}
}

func TestEvaluateURLEgressLocalOnlyBlocksExternalWebhook(t *testing.T) {
	d := privacy.EvaluateURLEgress(privacy.ModeLocalOnly, "https://hooks.slack.com/services/T00/B00/XXX")
	if d.Allowed {
		t.Fatalf("expected EXTERNAL webhook blocked: %+v", d)
	}
	d2 := privacy.EvaluateURLEgress(privacy.ModeLocalOnly, "http://10.0.0.9:9000/hook")
	if !d2.Allowed || d2.EndpointClass != privacy.ClassLocal {
		t.Fatalf("expected local webhook allowed: %+v", d2)
	}
}

func TestEvaluateURLEgressHybridAllowsExternal(t *testing.T) {
	d := privacy.EvaluateURLEgress(privacy.ModeHybrid, "https://hooks.slack.com/services/T00/B00/XXX")
	if !d.Allowed || d.EndpointClass != privacy.ClassExternal {
		t.Fatalf("hybrid should allow with EXTERNAL class: %+v", d)
	}
}

func TestNormalizeModeDefaultHybrid(t *testing.T) {
	if got := privacy.NormalizeMode(""); got != privacy.ModeHybrid {
		t.Fatalf("empty mode want hybrid got %s", got)
	}
}
