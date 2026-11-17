package web

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestSPAHandler(t *testing.T) {
	handler := SPAHandler()

	// 1. Root route serves index.html
	req := httptest.NewRequest("GET", "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("Expected 200 OK for root, got %d", rec.Code)
	}

	body := rec.Body.String()
	if !strings.Contains(body, "BountyBoard") || !strings.Contains(body, `<div id="root">`) {
		t.Errorf("Expected index.html content, got: %s", body)
	}

	// 2. Client-side route fallback also serves index.html
	req = httptest.NewRequest("GET", "/guilds/alchemy/quests/42", nil)
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("Expected 200 OK for client route fallback, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), `<div id="root">`) {
		t.Errorf("Expected fallback to index.html for deep client route")
	}
}
