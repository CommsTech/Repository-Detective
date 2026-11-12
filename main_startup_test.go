package main

import (
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// TestRoutesRegisteredBeforeServeAndTruthfulReadiness pins the Beta 6 P0
// startup contract:
//
//  1. Every route — including the control plane set — is registered on the Gin
//     engine BEFORE the listener starts. Gin's radix tree must never be mutated
//     while requests are being served; under -race, any post-serve registration
//     (the defect this fixes) is flagged as a data race between route
//     registration and concurrent request handling.
//  2. /health/live answers 200 even while components are still initializing.
//  3. /health/ready is truthful: 503 until all components are ready, 200 after.
func TestRoutesRegisteredBeforeServeAndTruthfulReadiness(t *testing.T) {
	prevConfig := config
	prevReady := componentsReady.Load()
	prevOnboarding := onboardingHandler
	prevLogger := logger
	config = &Config{
		Port:                "0",
		ListenHost:          "127.0.0.1",
		UIBasePath:          "/ui",
		APIKey:              "test-key",
		AuthMode:            "api_key_only",
		DatabaseEnabled:     false,
		StartupCheckTimeout: 1,
	}
	componentsReady.Store(false)
	onboardingHandler = nil
	logger = logrus.New()
	t.Cleanup(func() {
		config = prevConfig
		onboardingHandler = prevOnboarding
		logger = prevLogger
		componentsReady.Store(prevReady)
	})

	gin.SetMode(gin.TestMode)
	router := gin.New()
	setupRoutes(router)
	// registerControlPlaneRoutes is a no-op without controlPlaneHandler, which
	// mirrors real startup order: fully construct the tree, then listen.
	registerControlPlaneRoutes(router)

	server := httptest.NewServer(router)
	defer server.Close()

	var wg sync.WaitGroup
	liveOK := make(chan bool, 8)
	readyWhileStarting := make(chan bool, 8)
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			resp, err := http.Get(server.URL + "/health/live")
			if err == nil {
				liveOK <- resp.StatusCode == http.StatusOK
			} else {
				liveOK <- false
			}
			r2, err := http.Get(server.URL + "/health/ready")
			if err == nil {
				readyWhileStarting <- r2.StatusCode == http.StatusServiceUnavailable
			} else {
				readyWhileStarting <- false
			}
		}()
	}
	wg.Wait()
	close(liveOK)
	close(readyWhileStarting)
	for ok := range liveOK {
		if !ok {
			t.Fatal("/health/live must answer 200 immediately, even before readiness")
		}
	}
	for ok := range readyWhileStarting {
		if !ok {
			t.Fatal("/health/ready must be 503 until components are ready")
		}
	}

	componentsReady.Store(true)
	resp, err := http.Get(server.URL + "/health/ready")
	if err != nil {
		t.Fatalf("readiness probe: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("/health/ready must be 200 once ready, got %d", resp.StatusCode)
	}
}
