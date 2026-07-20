package store

import (
	"os"
	"testing"
)

func TestNewAndLog(t *testing.T) {
	path := "/tmp/vault-test-audit.db"
	os.Remove(path)

	db, err := New(path)
	if err != nil {
		t.Fatalf("create db: %v", err)
	}
	defer db.Close()

	db.Log("sandbox", "start", "cmd=test")
	db.Log("mcp_gate", "register", "server=fs tools=14")
	db.Log("net", "blocked", "host=evil.com")

	entries, err := db.Query("", "", 100)
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	if len(entries) != 3 {
		t.Fatalf("expected 3 entries, got %d", len(entries))
	}
}

func TestQueryBySource(t *testing.T) {
	path := "/tmp/vault-test-audit2.db"
	os.Remove(path)

	db, _ := New(path)
	defer db.Close()

	db.Log("sandbox", "start", "a")
	db.Log("net", "blocked", "b")
	db.Log("sandbox", "stop", "c")

	entries, err := db.Query("sandbox", "", 100)
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("expected 2 sandbox entries, got %d", len(entries))
	}
}

func TestQueryByEvent(t *testing.T) {
	path := "/tmp/vault-test-audit3.db"
	os.Remove(path)

	db, _ := New(path)
	defer db.Close()

	db.Log("sandbox", "start", "a")
	db.Log("sandbox", "stop", "b")

	entries, err := db.Query("", "start", 100)
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 start entry, got %d", len(entries))
	}
	if entries[0].Event != "start" {
		t.Fatalf("expected start event, got %s", entries[0].Event)
	}
}

func TestQueryLimit(t *testing.T) {
	path := "/tmp/vault-test-audit4.db"
	os.Remove(path)

	db, _ := New(path)
	defer db.Close()

	for i := 0; i < 50; i++ {
		db.Log("test", "event", "x")
	}

	entries, _ := db.Query("", "", 10)
	if len(entries) != 10 {
		t.Fatalf("expected 10 entries with limit, got %d", len(entries))
	}
}

func TestQueryReturnsNewestFirst(t *testing.T) {
	path := "/tmp/vault-test-audit5.db"
	os.Remove(path)

	db, _ := New(path)
	defer db.Close()

	db.Log("test", "event", "first")
	db.Log("test", "event", "second")

	entries, _ := db.Query("", "", 2)
	if len(entries) != 2 {
		t.Fatalf("expected 2, got %d", len(entries))
	}
	if entries[0].Detail != "second" {
		t.Fatalf("expected second first (newest), got %s", entries[0].Detail)
	}
}
