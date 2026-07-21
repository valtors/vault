package mcp

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sync"
	"time"

	"github.com/valtors/vault/internal/inject"
	"github.com/valtors/vault/internal/store"
)

type Server struct {
	cmd      *exec.Cmd
	stdin    io.WriteCloser
	stdout   io.Reader
	db       *store.DB
	mu       sync.Mutex
	scanning bool
}

type jsonRPC struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.Number     `json:"id,omitempty"`
	Method  string          `json:"method,omitempty"`
	Params  json.RawMessage `json:"params,omitempty"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *rpcError        `json:"error,omitempty"`
}

type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type Tool struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

type ToolsListResult struct {
	Tools []Tool `json:"tools"`
}

func NewServer(ctx context.Context, command string, args []string, db *store.DB) (*Server, error) {
	cmd := exec.CommandContext(ctx, command, args...)
	cmd.Env = os.Environ()

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("stdin pipe: %w", err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("stdout pipe: %w", err)
	}
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("start command: %w", err)
	}

	if db != nil {
		db.Log("mcp", "START", fmt.Sprintf("started: %s %v", command, args))
	}

	return &Server{
		cmd:    cmd,
		stdin:  stdin,
		stdout: stdout,
		db:     db,
	}, nil
}

func (s *Server) Proxy(in io.Reader, out io.Writer) error {
	go io.Copy(s.stdin, in)

	scanner := bufio.NewScanner(s.stdout)
	scanner.Buffer(make([]byte, 0, 1024*1024), 10*1024*1024)

	for scanner.Scan() {
		line := scanner.Bytes()
		var msg jsonRPC
		if err := json.Unmarshal(line, &msg); err != nil {
			out.Write(line)
			out.Write([]byte("\n"))
			continue
		}

		if msg.Method == "tools/list" {
			s.handleToolsList(line, out)
			continue
		}

		if msg.Method == "tools/call" && s.db != nil {
			var params map[string]interface{}
			if msg.Params != nil {
				json.Unmarshal(msg.Params, &params)
				toolName, _ := params["name"].(string)
				s.db.Log("mcp", "CALL", toolName)
			}
		}

		out.Write(line)
		out.Write([]byte("\n"))
	}

	return scanner.Err()
}

func (s *Server) handleToolsList(line []byte, out io.Writer) {
	var msg jsonRPC
	json.Unmarshal(line, &msg)

	result := ToolsListResult{}
	if msg.Result != nil {
		json.Unmarshal(msg.Result, &result)
	}

	var totalFindings int
	var allFindings []inject.Finding

	for i := range result.Tools {
		findings := inject.Scan(result.Tools[i].Description, result.Tools[i].Name)
		if len(findings) > 0 {
			cleaned, stripped := inject.Strip(result.Tools[i].Description)
			result.Tools[i].Description = cleaned
			allFindings = append(allFindings, stripped...)
			totalFindings += len(stripped)
		}
	}

	if totalFindings > 0 && s.db != nil {
		s.db.Log("mcp", "INJECT", fmt.Sprintf("stripped %d injection patterns from %d tools", totalFindings, len(result.Tools)))
	}

	newResult, _ := json.Marshal(result)
	msg.Result = newResult
	msg.Error = nil

	output, _ := json.Marshal(msg)
	out.Write(output)
	out.Write([]byte("\n"))
}

func (s *Server) Close() error {
	s.stdin.Close()
	done := make(chan error, 1)
	go func() { done <- s.cmd.Wait() }()
	select {
	case err := <-done:
		return err
	case <-time.After(5 * time.Second):
		s.cmd.Process.Kill()
		<-done
		return fmt.Errorf("killed: did not exit in 5s")
	}
}

func (s *Server) ScanTools(ctx context.Context) ([]inject.Result, error) {
	initReq, _ := json.Marshal(jsonRPC{
		JSONRPC: "2.0",
		ID:     json.Number("1"),
		Method: "initialize",
		Params: json.RawMessage(`{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"vault","version":"0.1.0"}}`),
	})
	s.stdin.Write(initReq)
	s.stdin.Write([]byte("\n"))

	notify, _ := json.Marshal(jsonRPC{
		JSONRPC: "2.0",
		Method:  "notifications/initialized",
	})
	s.stdin.Write(notify)
	s.stdin.Write([]byte("\n"))

	listReq, _ := json.Marshal(jsonRPC{
		JSONRPC: "2.0",
		ID:     json.Number("2"),
		Method:  "tools/list",
	})
	s.stdin.Write(listReq)
	s.stdin.Write([]byte("\n"))

	scanner := bufio.NewScanner(s.stdout)
	scanner.Buffer(make([]byte, 0, 1024*1024), 10*1024*1024)

	deadline, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	var results []inject.Result
	for scanner.Scan() {
		select {
		case <-deadline.Done():
			return results, fmt.Errorf("timeout")
		default:
		}

		var msg jsonRPC
		if err := json.Unmarshal(scanner.Bytes(), &msg); err != nil {
			continue
		}
		if msg.Method == "tools/list" || (msg.ID != "" && string(msg.ID) == "2") {
			tools := ToolsListResult{}
			if msg.Result != nil {
				json.Unmarshal(msg.Result, &tools)
			}
			for _, tool := range tools.Tools {
				results = append(results, inject.ScanDescription(tool.Description, tool.Name))
			}
			break
		}
	}

	return results, scanner.Err()
}
