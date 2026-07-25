package mcp

import (
	"context"
	"encoding/json"
	"testing"

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
