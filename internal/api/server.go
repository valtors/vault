package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/valtors/vault/internal/sandbox"
	"github.com/valtors/vault/internal/store"
)

type Server struct {
	mu        sync.Mutex
	sandboxes map[int64]*sandbox.Sandbox
	httpSrv   *http.Server
	port      int
}

func NewServer(port int) *Server {
	return &Server{
		sandboxes: make(map[int64]*sandbox.Sandbox),
		port:      port,
	}
}

func (s *Server) Start() error {
	mux := http.NewServeMux()
	mux.HandleFunc("/health", s.handleHealth)
	mux.HandleFunc("/sandboxes", s.handleSandboxes)
	mux.HandleFunc("/sandboxes/", s.handleSandboxByID)

	s.httpSrv = &http.Server{
		Addr:    fmt.Sprintf(":%d", s.port),
		Handler: mux,
	}

	return s.httpSrv.ListenAndServe()
}

func (s *Server) Stop() error {
	if s.httpSrv != nil {
		return s.httpSrv.Close()
	}
	return nil
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "up"})
}

type createRequest struct {
	Command    string   `json:"command"`
	Args       []string `json:"args"`
	Timeout    int      `json:"timeout"`
	AllowedDirs []string `json:"allowed_dirs"`
}

func (s *Server) handleSandboxes(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		s.createSandbox(w, r)
		return
	}
	if r.Method == http.MethodGet {
		s.listSandboxes(w, r)
		return
	}
	http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
}

func (s *Server) createSandbox(w http.ResponseWriter, r *http.Request) {
	var req createRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	cfg := sandbox.Config{
		Command:     req.Command,
		Args:        req.Args,
		TimeoutSecs: req.Timeout,
		AllowedDirs: req.AllowedDirs,
	}

	sb, err := sandbox.New(cfg)
	if err != nil {
		http.Error(w, fmt.Sprintf("create: %v", err), http.StatusInternalServerError)
		return
	}

	s.mu.Lock()
	s.sandboxes[sb.ID()] = sb
	s.mu.Unlock()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"id":   sb.ID(),
		"home": sb.Home(),
	})
}

func (s *Server) listSandboxes(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	defer s.mu.Unlock()

	type sbInfo struct {
		ID   int64  `json:"id"`
		Home string `json:"home"`
		Done bool   `json:"done"`
	}

	var list []sbInfo
	for _, sb := range s.sandboxes {
		list = append(list, sbInfo{
			ID:   sb.ID(),
			Home: sb.Home(),
			Done: sb.IsDone(),
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(list)
}

func (s *Server) handleSandboxByID(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path[len("/sandboxes/"):]
	if path == "" {
		http.Error(w, "missing id", http.StatusBadRequest)
		return
	}

	parts := splitPath(path)
	if len(parts) == 0 {
		http.Error(w, "invalid path", http.StatusBadRequest)
		return
	}

	id, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	s.mu.Lock()
	sb, ok := s.sandboxes[id]
	s.mu.Unlock()

	if !ok {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}

	if len(parts) == 1 {
		info := map[string]interface{}{
			"id":   sb.ID(),
			"home": sb.Home(),
			"done": sb.IsDone(),
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(info)
		return
	}

	if len(parts) == 2 && parts[1] == "logs" {
		category := r.URL.Query().Get("category")
		limitStr := r.URL.Query().Get("limit")
		limit, _ := strconv.Atoi(limitStr)
		if limit == 0 {
			limit = 100
		}

		entries, err := sb.AuditLog(category, limit)
		if err != nil {
			http.Error(w, fmt.Sprintf("query: %v", err), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"count":   len(entries),
			"entries": entries,
		})
		return
	}

	if len(parts) == 2 && parts[1] == "kill" && r.Method == http.MethodPost {
		if err := sb.Kill(); err != nil {
			http.Error(w, fmt.Sprintf("kill: %v", err), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusNoContent)
		return
	}

	if len(parts) == 2 && parts[1] == "cleanup" && r.Method == http.MethodPost {
		if err := sb.Cleanup(); err != nil {
			http.Error(w, fmt.Sprintf("cleanup: %v", err), http.StatusInternalServerError)
			return
		}
		s.mu.Lock()
		delete(s.sandboxes, id)
		s.mu.Unlock()
		w.WriteHeader(http.StatusNoContent)
		return
	}

	http.Error(w, "not found", http.StatusNotFound)
}

func (s *Server) handleAudit(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	defer s.mu.Unlock()

	type auditEntry struct {
		SandboxID int64        `json:"sandbox_id"`
		Entries   []store.Entry `json:"entries"`
	}

	var all []auditEntry
	for _, sb := range s.sandboxes {
		entries, _ := sb.AuditLog("", 100)
		if len(entries) > 0 {
			all = append(all, auditEntry{SandboxID: sb.ID(), Entries: entries})
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"count":  len(all),
		"audits": all,
	})
}

func splitPath(p string) []string {
	var parts []string
	current := ""
	for i := 0; i < len(p); i++ {
		if p[i] == '/' {
			if current != "" {
				parts = append(parts, current)
				current = ""
			}
		} else {
			current += string(p[i])
		}
	}
	if current != "" {
		parts = append(parts, current)
	}
	return parts
}

func (s *Server) uptime() time.Duration {
	return time.Since(time.Now())
}
