package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHandleHealth(t *testing.T) {
	s := NewServer(0)
	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()
	s.handleHealth(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)
	if resp["status"] != "up" {
		t.Errorf("expected ok, got %v", resp["status"])
	}
}

func TestHandleSandboxesGET(t *testing.T) {
	s := NewServer(0)
	req := httptest.NewRequest("GET", "/sandboxes", nil)
	w := httptest.NewRecorder()
	s.handleSandboxes(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestHandleSandboxesMethodNotAllowed(t *testing.T) {
	s := NewServer(0)
	req := httptest.NewRequest("PUT", "/sandboxes", nil)
	w := httptest.NewRecorder()
	s.handleSandboxes(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405, got %d", w.Code)
	}
}

func TestNewServer(t *testing.T) {
	s := NewServer(8080)
	if s.port != 8080 {
		t.Errorf("expected 8080, got %d", s.port)
	}
	if s.sandboxes == nil {
		t.Error("expected non-nil sandboxes map")
	}
}
