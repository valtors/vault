package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os/exec"
	"sync"
	"time"

	"github.com/valtors/vault/internal/inject"
	"github.com/valtors/vault/internal/store"
)

type Server struct {
	Command string   `json:"command"`
	Args    []string `json:"args"`
	Tools   []Tool   `json:"tools"`
}

type Tool struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

type Gate struct {
	db     *store.DB
	mu     sync.RWMutex
	servers map[string]*Server
}

func NewGate(db *store.DB) *Gate {
	return &Gate{
		db:     db,
		servers: make(map[string]*Server),
	}
}

func (g *Gate) Register(name, command string, args []string) error {
	g.mu.Lock()
	defer g.mu.Unlock()

	srv := &Server{
		Command: command,
		Args:    args,
	}

	if err := g.probe(srv); err != nil {
		return fmt.Errorf("probe %s: %w", name, err)
	}

	for i, tool := range srv.Tools {
		findings := inject.Scan(tool.Description, tool.Name)
		for _, f := range findings {
			srv.Tools[i].Description = inject.Strip(tool.Description)
			g.db.Log("mcp_gate", "injection_blocked", fmt.Sprintf("server=%s tool=%s pattern=%s severity=%s", name, f.Tool, f.Pattern, f.Severity))
		}
	}

	g.servers[name] = srv
	g.db.Log("mcp_gate", "register", fmt.Sprintf("server=%s tools=%d", name, len(srv.Tools)))

	return nil
}

func (g *Gate) probe(srv *Server) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, srv.Command, srv.Args...)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return err
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}

	if err := cmd.Start(); err != nil {
		return err
	}

	initReq := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "initialize",
		"params": map[string]interface{}{
			"protocolVersion": "2024-11-05",
			"capabilities":    map[string]interface{}{},
			"clientInfo": map[string]interface{}{
				"name":    "vault",
				"version": "0.1.0",
			},
		},
	}

	encoder := json.NewEncoder(stdin)
	if err := encoder.Encode(initReq); err != nil {
		cmd.Process.Kill()
		return err
	}

	listReq := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      2,
		"method":  "tools/list",
		"params":  map[string]interface{}{},
	}
	if err := encoder.Encode(listReq); err != nil {
		cmd.Process.Kill()
		return err
	}

	decoder := json.NewDecoder(stdout)
	var listResp struct {
		Result struct {
			Tools []Tool `json:"tools"`
		} `json:"result"`
	}

	for {
		var msg map[string]json.RawMessage
		if err := decoder.Decode(&msg); err != nil {
			if err == io.EOF {
				break
			}
			continue
		}

		if id, ok := msg["id"]; ok {
			var idVal float64
			json.Unmarshal(id, &idVal)
			if idVal == 2 {
				if result, ok := msg["result"]; ok {
					var lr struct {
						Tools []Tool `json:"tools"`
					}
					json.Unmarshal(result, &lr)
					srv.Tools = lr.Tools
					cmd.Process.Kill()
					return nil
				}
			}
		}
	}

	cmd.Process.Kill()
	return fmt.Errorf("no tools/list response")
}

func (g *Gate) List() []Server {
	g.mu.RLock()
	defer g.mu.RUnlock()

	var list []Server
	for _, srv := range g.servers {
		list = append(list, *srv)
	}
	return list
}

func (g *Gate) Remove(name string) {
	g.mu.Lock()
	defer g.mu.Unlock()
	delete(g.servers, name)
	g.db.Log("mcp_gate", "remove", fmt.Sprintf("server=%s", name))
}
