package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
)

func TestCreateSandbox_BadJSON(t *testing.T) {
	s := NewServer(0)
	req := httptest.NewRequest("POST", "/sandboxes", strings.NewReader("bad"))
	w := httptest.NewRecorder()
	s.handleSandboxes(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestCreateSandbox_Valid(t *testing.T) {
	s := NewServer(0)
	body := `{"command": "echo", "args": ["hi"], "timeout": 5}`
	req := httptest.NewRequest("POST", "/sandboxes", strings.NewReader(body))
	w := httptest.NewRecorder()
	s.handleSandboxes(w, req)
	if w.Code != http.StatusCreated {
		t.Errorf("expected 201, got %d", w.Code)
	}
	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)
	if resp["id"] == nil {
		t.Error("expected id in response")
	}
	if resp["home"] == nil {
		t.Error("expected home in response")
	}
}

func TestListSandboxes_Empty(t *testing.T) {
	s := NewServer(0)
	req := httptest.NewRequest("GET", "/sandboxes", nil)
	w := httptest.NewRecorder()
	s.handleSandboxes(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestListSandboxes_AfterCreate(t *testing.T) {
	s := NewServer(0)
	body := `{"command": "echo", "args": ["hi"]}`
	createReq := httptest.NewRequest("POST", "/sandboxes", strings.NewReader(body))
	createW := httptest.NewRecorder()
	s.handleSandboxes(createW, createReq)

	listReq := httptest.NewRequest("GET", "/sandboxes", nil)
	listW := httptest.NewRecorder()
	s.handleSandboxes(listW, listReq)
	var list []map[string]interface{}
	json.NewDecoder(listW.Body).Decode(&list)
	if len(list) != 1 {
		t.Errorf("expected 1 sandbox, got %d", len(list))
	}
}

func TestHandleSandboxByID_MissingID(t *testing.T) {
	s := NewServer(0)
	req := httptest.NewRequest("GET", "/sandboxes/", nil)
	w := httptest.NewRecorder()
	s.handleSandboxByID(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestHandleSandboxByID_InvalidID(t *testing.T) {
	s := NewServer(0)
	req := httptest.NewRequest("GET", "/sandboxes/abc", nil)
	w := httptest.NewRecorder()
	s.handleSandboxByID(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestHandleSandboxByID_NotFound(t *testing.T) {
	s := NewServer(0)
	req := httptest.NewRequest("GET", "/sandboxes/999", nil)
	w := httptest.NewRecorder()
	s.handleSandboxByID(w, req)
	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestHandleSandboxByID_Get(t *testing.T) {
	s := NewServer(0)
	body := `{"command": "sleep", "args": ["10"], "timeout": 30}`
	createReq := httptest.NewRequest("POST", "/sandboxes", strings.NewReader(body))
	createW := httptest.NewRecorder()
	s.handleSandboxes(createW, createReq)
	var created map[string]interface{}
	json.NewDecoder(createW.Body).Decode(&created)
	id := int64(created["id"].(float64))

	req := httptest.NewRequest("GET", "/sandboxes/"+fmtID(id), nil)
	w := httptest.NewRecorder()
	s.handleSandboxByID(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestSplitPath(t *testing.T) {
	parts := splitPath("/sandboxes/123/exec")
	if len(parts) < 1 {
		t.Fatal("expected at least 1 part")
	}
	if parts[1] != "123" {
		t.Errorf("expected 123, got %s", parts[0])
	}
}

func fmtID(id int64) string {
	return strconv.FormatInt(id, 10)
}

func TestServer_Stop_WithoutStart(t *testing.T) {
	srv := NewServer(0)
	err := srv.Stop()
	if err != nil {
		t.Errorf("Stop without Start should return nil, got %v", err)
	}
}
