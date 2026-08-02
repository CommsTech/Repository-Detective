package mcpbridge_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"git.commsnet.org/commstech/repository-detective/mcpbridge"
)

func TestMCPInitializeAndHealth(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/health":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"status":"ok"}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	in := strings.NewReader(strings.Join([]string{
		`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"t","version":"0"}}}`,
		`{"jsonrpc":"2.0","method":"notifications/initialized"}`,
		`{"jsonrpc":"2.0","id":2,"method":"tools/list"}`,
		`{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"rd_health","arguments":{}}}`,
	}, "\n") + "\n")
	var out bytes.Buffer
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	cfg := mcpbridge.Config{BaseURL: srv.URL, APIKey: "test", Client: srv.Client()}
	if err := mcpbridge.Serve(ctx, cfg, in, &out); err != nil && err != context.Canceled {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSpace(out.String()), "\n")
	if len(lines) < 3 {
		t.Fatalf("expected >=3 responses, got %d: %s", len(lines), out.String())
	}
	var initResp map[string]any
	if err := json.Unmarshal([]byte(lines[0]), &initResp); err != nil {
		t.Fatal(err)
	}
	result, _ := initResp["result"].(map[string]any)
	if result["protocolVersion"] != "2024-11-05" {
		t.Fatalf("protocol: %+v", result)
	}
	var callResp map[string]any
	if err := json.Unmarshal([]byte(lines[len(lines)-1]), &callResp); err != nil {
		t.Fatal(err)
	}
	callResult, _ := callResp["result"].(map[string]any)
	if callResult["isError"] == true {
		t.Fatalf("health tool error: %+v", callResult)
	}
}
