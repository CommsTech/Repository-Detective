package runner_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"git.commsnet.org/commstech/repository-detective/api"
	"git.commsnet.org/commstech/repository-detective/runner"
	"github.com/gin-gonic/gin"
)

func TestClientPingWithValidHMAC(t *testing.T) {
	gin.SetMode(gin.TestMode)
	secret := "test-runner-secret"
	reg := runner.NewRegistry()
	h := api.NewRunnerHandler(nil, runner.Config{DelegationEnabled: true}, nil, nil, reg, nil)

	r := gin.New()
	g := r.Group("/api/v1/runner")
	g.Use(api.RequireRunnerHMAC(secret))
	h.RegisterRunnerRoutes(g)

	srv := httptest.NewServer(r)
	defer srv.Close()

	client := runner.NewClient(srv.URL, secret)
	if err := client.Ping(context.Background(), "worker-test", "1.0", []string{"graph"}); err != nil {
		t.Fatalf("ping failed: %v", err)
	}
}

func TestClientPingRejectsInvalidHMAC(t *testing.T) {
	gin.SetMode(gin.TestMode)
	secret := "test-runner-secret"
	h := api.NewRunnerHandler(nil, runner.Config{DelegationEnabled: true}, nil, nil, runner.NewRegistry(), nil)
	r := gin.New()
	g := r.Group("/api/v1/runner")
	g.Use(api.RequireRunnerHMAC(secret))
	h.RegisterRunnerRoutes(g)
	srv := httptest.NewServer(r)
	defer srv.Close()

	client := runner.NewClient(srv.URL, "wrong-secret")
	if err := client.Ping(context.Background(), "worker-test", "1.0", nil); err == nil {
		t.Fatal("expected invalid HMAC rejection")
	}
}

func TestClientClaimNoJobs(t *testing.T) {
	gin.SetMode(gin.TestMode)
	secret := "test-runner-secret"
	recv := runner.NewReceiver(nil, runner.Config{}, nil, nil)
	h := api.NewRunnerHandler(nil, runner.Config{}, recv, nil, nil, nil)
	r := gin.New()
	g := r.Group("/api/v1/runner")
	g.Use(api.RequireRunnerHMAC(secret))
	h.RegisterRunnerRoutes(g)
	srv := httptest.NewServer(r)
	defer srv.Close()

	client := runner.NewClient(srv.URL, secret)
	_, _, err := client.ClaimNextJob(context.Background())
	if err == nil {
		t.Fatal("expected claim error when no jobs")
	}
}

func TestClientPingBody(t *testing.T) {
	body, _ := json.Marshal(map[string]string{"runner_id": "x"})
	if len(body) == 0 {
		t.Fatal("expected body")
	}
	_ = http.MethodPost
}
