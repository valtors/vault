package store

import (
	"testing"
)

func TestStore_Count(t *testing.T) {
	db := setupTestDB(t)
	count, err := db.Count()
	if err != nil {
		t.Fatalf("Count: %v", err)
	}
	if count != 0 {
		t.Errorf("expected 0, got %d", count)
	}

	db.Log("tool", "call", "test detail")
	count, _ = db.Count()
	if count != 1 {
		t.Errorf("expected 1, got %d", count)
	}

	db.Log("tool", "call", "another")
	db.Log("network", "request", "get")
	count, _ = db.Count()
	if count != 3 {
		t.Errorf("expected 3, got %d", count)
	}
}

func TestStore_QueryByCategory(t *testing.T) {
	db := setupTestDB(t)
	db.Log("tool", "call", "a")
	db.Log("network", "request", "b")
	db.Log("tool", "call", "c")

	entries, err := db.Query("tool", 10)
	if err != nil {
		t.Fatalf("Query: %v", err)
	}
	if len(entries) != 2 {
		t.Errorf("expected 2, got %d", len(entries))
	}
	for _, e := range entries {
		if e.Category != "tool" {
			t.Errorf("expected tool, got %s", e.Category)
		}
	}
}

func TestStore_QueryAllCategories(t *testing.T) {
	db := setupTestDB(t)
	db.Log("tool", "call", "a")
	db.Log("network", "request", "b")

	entries, err := db.Query("", 10)
	if err != nil {
		t.Fatalf("Query: %v", err)
	}
	if len(entries) != 2 {
		t.Errorf("expected 2, got %d", len(entries))
	}
}

func TestStore_QueryLimitDefault(t *testing.T) {
	db := setupTestDB(t)
	for i := 0; i < 5; i++ {
		db.Log("tool", "call", "x")
	}
	entries, err := db.Query("tool", 0)
	if err != nil {
		t.Fatalf("Query: %v", err)
	}
	if len(entries) != 5 {
		t.Errorf("expected 5, got %d", len(entries))
	}
}

func TestStore_QueryLimitCapped(t *testing.T) {
	db := setupTestDB(t)
	for i := 0; i < 5; i++ {
		db.Log("tool", "call", "x")
	}
	entries, err := db.Query("tool", 3)
	if err != nil {
		t.Fatalf("Query: %v", err)
	}
	if len(entries) != 3 {
		t.Errorf("expected 3, got %d", len(entries))
	}
}

func TestStore_QueryEmptyDB(t *testing.T) {
	db := setupTestDB(t)
	entries, err := db.Query("", 10)
	if err != nil {
		t.Fatalf("Query: %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("expected 0, got %d", len(entries))
	}
}

func TestStore_LogMultipleCategories(t *testing.T) {
	db := setupTestDB(t)
	categories := []string{"tool", "network", "fs", "env", "mcp"}
	for _, cat := range categories {
		if err := db.Log(cat, "action", "detail"); err != nil {
			t.Fatalf("Log %s: %v", cat, err)
		}
	}
	count, _ := db.Count()
	if count != 5 {
		t.Errorf("expected 5, got %d", count)
	}
}

func TestStore_EntryFields(t *testing.T) {
	db := setupTestDB(t)
	db.Log("tool", "call", "some detail")
	entries, _ := db.Query("", 1)
	if len(entries) != 1 {
		t.Fatal("expected 1 entry")
	}
	e := entries[0]
	if e.Category != "tool" {
		t.Errorf("expected tool, got %s", e.Category)
	}
	if e.Action != "call" {
		t.Errorf("expected call, got %s", e.Action)
	}
	if e.Detail != "some detail" {
		t.Errorf("expected 'some detail', got %s", e.Detail)
	}
	if e.ID <= 0 {
		t.Error("expected positive ID")
	}
	if e.Timestamp.IsZero() {
		t.Error("expected non-zero timestamp")
	}
}

func setupTestDB(t *testing.T) *DB {
	t.Helper()
	db, err := New(":memory:")
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

type cleanupHolder struct {
	fn func()
}

func (c *cleanupHolder) Cleanup() { c.fn() }

func TestStore_TempDB(t *testing.T) {
	ch := &cleanupHolder{fn: func() {}}
	db, err := TempDB(ch)
	if err != nil {
		t.Fatalf("TempDB: %v", err)
	}
	if db == nil {
		t.Fatal("expected non-nil db")
	}
	count, _ := db.Count()
	if count != 0 {
		t.Errorf("expected 0, got %d", count)
	}
	db.Close()
}

func TestStore_CloseReopen(t *testing.T) {
	db := setupTestDB(t)
	db.Log("x", "y", "z")
	if err := db.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
}
