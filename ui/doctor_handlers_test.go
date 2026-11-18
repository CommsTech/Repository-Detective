package ui_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestDoctorPageHasNoInlineScript(t *testing.T) {
	r, _ := testUIWithAPIKeyAuth(t, "doctor-key")

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/ui/doctor", nil)
	req.Header.Set("X-Repository-Detective-API-Key", "doctor-key")
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%q", w.Code, w.Body.String())
	}
	body := w.Body.String()
	if !strings.Contains(body, `id="rd-doctor-run"`) {
		t.Fatal("missing Run doctor button")
	}
	if !strings.Contains(body, `data-report-url="/ui/doctor/report"`) {
		t.Fatal("missing data-report-url for CSP-safe fetch")
	}
	// CSP is script-src 'self' — inline handlers must not appear on this page.
	if strings.Contains(body, "addEventListener('click'") || strings.Contains(body, "fetch('/api/v1/doctor'") {
		t.Fatal("doctor page still embeds inline run script (blocked by CSP)")
	}
	if !strings.Contains(body, `/ui/static/app.js`) {
		t.Fatal("expected app.js for doctor click handler")
	}
}

func TestDoctorReportUsesUIAuthCookie(t *testing.T) {
	r, h := testUIWithAPIKeyAuth(t, "doctor-key")
	h.SetDoctorFns(
		func(ctx context.Context, owner, repo string) any {
			return map[string]any{
				"overall": "HEALTHY",
				"summary": "ok",
				"checks": []map[string]any{
					{"id": "core.health", "state": "PASS", "summary": "up"},
				},
			}
		},
		func(ctx context.Context, owner, repo string) any {
			return map[string]any{"bundle": true}
		},
	)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/ui/doctor/report", nil)
	req.AddCookie(&http.Cookie{Name: "rd_ui_sess", Value: "doctor-key", Path: "/ui"})
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%q", w.Code, w.Body.String())
	}
	var payload map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &payload); err != nil {
		t.Fatalf("json: %v body=%q", err, w.Body.String())
	}
	if payload["overall"] != "HEALTHY" {
		t.Fatalf("unexpected payload: %#v", payload)
	}
}

func TestDoctorReportUnauthorizedWithoutKey(t *testing.T) {
	r, h := testUIWithAPIKeyAuth(t, "doctor-key")
	h.SetDoctorFns(
		func(ctx context.Context, owner, repo string) any { return map[string]any{"overall": "HEALTHY"} },
		nil,
	)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/ui/doctor/report", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}
