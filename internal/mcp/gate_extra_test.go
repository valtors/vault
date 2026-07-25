package mcp

import (
	"bytes"
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/valtors/vault/internal/store"
)

func TestJSONRPC_Serialization(t *testing.T) {
	rpc := jsonRPC{
		JSONRPC: "2.0",
		ID:      "1",
		Method:  "tools/list",
	}
	data, err := json.Marshal(rpc)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	var parsed jsonRPC
	json.Unmarshal(data, &parsed)
	if parsed.Method != "tools/list" {
		t.Errorf("expected tools/list, got %s", parsed.Method)
	}
}

func TestJSONRPC_WithError(t *testing.T) {
	rpc := jsonRPC{
		JSONRPC: "2.0",
		ID:      "2",
		Error:   &rpcError{Code: -32601, Message: "method not found"},
	}
	data, _ := json.Marshal(rpc)
	var parsed jsonRPC
	json.Unmarshal(data, &parsed)
	if parsed.Error == nil {
		t.Fatal("expected error")
	}
	if parsed.Error.Code != -32601 {
		t.Errorf("expected -32601, got %d", parsed.Error.Code)
	}
}

func TestRPCError_Serialization(t *testing.T) {
	e := rpcError{Code: -32700, Message: "parse error"}
	data, _ := json.Marshal(e)
	var parsed rpcError
	json.Unmarshal(data, &parsed)
	if parsed.Code != -32700 {
		t.Errorf("expected -32700, got %d", parsed.Code)
	}
}

func TestTool_Serialization(t *testing.T) {
	tool := Tool{Name: "test_tool", Description: "a test tool"}
	data, _ := json.Marshal(tool)
	var parsed Tool
	json.Unmarshal(data, &parsed)
	if parsed.Name != "test_tool" {
		t.Errorf("expected test_tool, got %s", parsed.Name)
	}
}

func TestNewServer_EchoCommand(t *testing.T) {
	db, err := store.New(":memory:")
	if err != nil {
		t.Fatalf("store.New: %v", err)
	}
	defer db.Close()

	srv, err := NewServer(context.Background(), "echo", []string{}, db)
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}
	defer srv.Close()
	if srv == nil {
		t.Fatal("expected non-nil server")
	}
}

func TestNewServer_InvalidCommand(t *testing.T) {
	db, _ := store.New(":memory:")
	defer db.Close()

	_, err := NewServer(context.Background(), "nonexistent-cmd-12345", []string{}, db)
	if err == nil {
		t.Error("expected error for invalid command")
	}
}

func TestServer_Close(t *testing.T) {
	db, _ := store.New(":memory:")
	defer db.Close()

	srv, err := NewServer(context.Background(), "cat", []string{}, db)
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}
	if err := srv.Close(); err != nil {
		t.Errorf("Close: %v", err)
	}
}

func TestJSONRPC_ParamsParsing(t *testing.T) {
	rpc := jsonRPC{
		JSONRPC: "2.0",
		ID:      "4",
		Method:  "tools/call",
		Params:  json.RawMessage(`{"name":"test","arguments":{}}`),
	}
	data, _ := json.Marshal(rpc)
	var parsed jsonRPC
	json.Unmarshal(data, &parsed)
	if string(parsed.Params) == "" {
		t.Error("expected non-empty params")
	}
}

func TestServer_ScanTools_WithCat(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	dir := t.TempDir()
	db, err := store.New(dir + "/test.db")
	if err != nil {
		t.Fatalf("store.New: %v", err)
	}
	defer db.Close()

	srv, err := NewServer(ctx, "cat", nil, db)
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}
	defer srv.Close()

	_, err = srv.ScanTools(ctx)
	if err != nil {
		t.Log("ScanTools error (expected - cat doesn.t respond to tools/list)")
	}
}

func TestServer_HandleToolsList(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	dir := t.TempDir()
	db, err := store.New(dir + "/test.db")
	if err != nil {
		t.Fatalf("store.New: %v", err)
	}
	defer db.Close()

	srv, err := NewServer(ctx, "cat", nil, db)
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}
	defer srv.Close()

	line := []byte(`{"jsonrpc":"2.0","id":1,"method":"tools/list"}`)
	out := &bytes.Buffer{}
	srv.handleToolsList(line, out)
	if out.Len() == 0 {
		t.Error("expected output from handleToolsList")
	}
}
