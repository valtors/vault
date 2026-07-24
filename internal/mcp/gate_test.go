package mcp

import (
	"encoding/json"
	"testing"
)

func TestJSONRPC_Unmarshal(t *testing.T) {
	raw := `{"jsonrpc":"2.0","id":"1","method":"initialize","params":{}}`
	var msg jsonRPC
	if err := json.Unmarshal([]byte(raw), &msg); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if msg.JSONRPC != "2.0" {
		t.Errorf("expected 2.0, got %s", msg.JSONRPC)
	}
	if msg.ID.String() != "1" {
		t.Errorf("expected 1, got %s", msg.ID)
	}
	if msg.Method != "initialize" {
		t.Errorf("expected initialize, got %s", msg.Method)
	}
}

func TestJSONRPC_UnmarshalResult(t *testing.T) {
	raw := `{"jsonrpc":"2.0","id":"2","result":{"tools":[{"name":"read","description":"read a file"}]}}`
	var msg jsonRPC
	if err := json.Unmarshal([]byte(raw), &msg); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if msg.Result == nil {
		t.Fatal("expected non-nil result")
	}
	var tools ToolsListResult
	if err := json.Unmarshal(msg.Result, &tools); err != nil {
		t.Fatalf("unmarshal tools: %v", err)
	}
	if len(tools.Tools) != 1 {
		t.Fatalf("expected 1 tool, got %d", len(tools.Tools))
	}
	if tools.Tools[0].Name != "read" {
		t.Errorf("expected read, got %s", tools.Tools[0].Name)
	}
}

func TestJSONRPC_UnmarshalError(t *testing.T) {
	raw := `{"jsonrpc":"2.0","id":"1","error":{"code":-32601,"message":"method not found"}}`
	var msg jsonRPC
	if err := json.Unmarshal([]byte(raw), &msg); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if msg.Error == nil {
		t.Fatal("expected error")
	}
	if msg.Error.Code != -32601 {
		t.Errorf("expected -32601, got %d", msg.Error.Code)
	}
	if msg.Error.Message != "method not found" {
		t.Errorf("expected 'method not found', got %s", msg.Error.Message)
	}
}

func TestRPCError(t *testing.T) {
	e := rpcError{Code: -1, Message: "test"}
	if e.Code != -1 || e.Message != "test" {
		t.Error("rpcError fields wrong")
	}
}

func TestTool(t *testing.T) {
	tool := Tool{Name: "read_file", Description: "reads a file"}
	if tool.Name != "read_file" || tool.Description != "reads a file" {
		t.Error("tool fields wrong")
	}
}

func TestToolsListResult_Empty(t *testing.T) {
	raw := `{"tools":[]}`
	var result ToolsListResult
	if err := json.Unmarshal([]byte(raw), &result); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(result.Tools) != 0 {
		t.Errorf("expected 0 tools, got %d", len(result.Tools))
	}
}

func TestJSONRPC_Method(t *testing.T) {
	tests := []struct {
		method string
		want   string
	}{
		{"initialize", "initialize"},
		{"tools/list", "tools/list"},
		{"notifications/initialized", "notifications/initialized"},
	}
	for _, tt := range tests {
		raw, _ := json.Marshal(jsonRPC{JSONRPC: "2.0", Method: tt.method})
		var msg jsonRPC
		json.Unmarshal(raw, &msg)
		if msg.Method != tt.want {
			t.Errorf("expected %s, got %s", tt.want, msg.Method)
		}
	}
}
