package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/valtors/vault/internal/store"
)

type Server struct {
	db   *store.DB
	port int
	mux  *http.ServeMux
}

func NewServer(db *store.DB, port int) *Server {
	s := &Server{
		db:   db,
		port: port,
		mux:  http.NewServeMux(),
	}
	s.routes()
	return s
}

func (s *Server) routes() {
	s.mux.HandleFunc("GET /audit", s.handleAudit)
	s.mux.HandleFunc("GET /health", s.handleHealth)
}

func (s *Server) Start() error {
	return http.ListenAndServe(fmt.Sprintf(":%d", s.port), s.mux)
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{
		"status": "up",
	})
}

func (s *Server) handleAudit(w http.ResponseWriter, r *http.Request) {
	source := r.URL.Query().Get("source")
	event := r.URL.Query().Get("event")
	limit := 100

	if l := r.URL.Query().Get("limit"); l != "" {
		if n, err := strconv.Atoi(l); err == nil && n > 0 && n <= 1000 {
			limit = n
		}
	}

	entries, err := s.db.Query(source, event, limit)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": err.Error(),
		})
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"entries": entries,
		"count":   len(entries),
	})
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}
