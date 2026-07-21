package store

import (
	"os"
	"testing"
)

func TestNewCreatesDB(t *testing.T) {
	path := tempPath(t)
	defer os.Remove(path)

	db, err := New(path)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer db.Close()
}

func TestLogInsertsEntry(t *testing.T) {
	path := tempPath(t)
	defer os.Remove(path)

	db, err := New(path)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer db.Close()

	if err := db.Log("net", "ALLOWED", "connection to api.openai.com:443"); err != nil {
		t.Fatalf("Log: %v", err)
	}

	count, err := db.Count()
	if err != nil {
		t.Fatalf("Count: %v", err)
	}
	if count != 1 {
		t.Fatalf("count = %d, want 1", count)
	}
}

func TestQueryAll(t *testing.T) {
	path := tempPath(t)
	defer os.Remove(path)

	db, _ := New(path)
	defer db.Close()

	db.Log("net", "ALLOWED", "a")
	db.Log("fs", "WRITE", "b")
	db.Log("mcp", "SCAN", "c")

	entries, err := db.Query("", 10)
	if err != nil {
		t.Fatalf("Query: %v", err)
	}
	if len(entries) != 3 {
		t.Fatalf("got %d entries, want 3", len(entries))
	}
}

func TestQueryByCategory(t *testing.T) {
	path := tempPath(t)
	defer os.Remove(path)

	db, _ := New(path)
	defer db.Close()

	db.Log("net", "ALLOWED", "a")
	db.Log("fs", "WRITE", "b")
	db.Log("net", "BLOCKED", "c")

	entries, err := db.Query("net", 10)
	if err != nil {
		t.Fatalf("Query: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("got %d entries, want 2", len(entries))
	}
	for _, e := range entries {
		if e.Category != "net" {
			t.Fatalf("category = %s, want net", e.Category)
		}
	}
}

func TestQueryLimit(t *testing.T) {
	path := tempPath(t)
	defer os.Remove(path)

	db, _ := New(path)
	defer db.Close()

	for i := 0; i < 50; i++ {
		db.Log("net", "ALLOWED", "entry")
	}

	entries, err := db.Query("", 5)
	if err != nil {
		t.Fatalf("Query: %v", err)
	}
	if len(entries) != 5 {
		t.Fatalf("got %d entries, want 5", len(entries))
	}
}

func TestCount(t *testing.T) {
	path := tempPath(t)
	defer os.Remove(path)

	db, _ := New(path)
	defer db.Close()

	count, err := db.Count()
	if err != nil {
		t.Fatalf("Count: %v", err)
	}
	if count != 0 {
		t.Fatalf("initial count = %d, want 0", count)
	}

	db.Log("test", "action", "detail")
	count, _ = db.Count()
	if count != 1 {
		t.Fatalf("count = %d, want 1", count)
	}
}

func TestQueryEmpty(t *testing.T) {
	path := tempPath(t)
	defer os.Remove(path)

	db, _ := New(path)
	defer db.Close()

	entries, err := db.Query("", 10)
	if err != nil {
		t.Fatalf("Query: %v", err)
	}
	if len(entries) != 0 {
		t.Fatalf("got %d entries, want 0", len(entries))
	}
}

func TestLogMultipleCategories(t *testing.T) {
	path := tempPath(t)
	defer os.Remove(path)

	db, _ := New(path)
	defer db.Close()

	db.Log("fs", "READ", "/etc/passwd")
	db.Log("net", "BLOCKED", "evil.com:443")
	db.Log("mcp", "INJECT", "prompt override detected")
	db.Log("env", "SANITIZE", "stripped 3 vars")

	entries, _ := db.Query("", 100)
	if len(entries) != 4 {
		t.Fatalf("got %d, want 4", len(entries))
	}

	netEntries, _ := db.Query("net", 100)
	if len(netEntries) != 1 {
		t.Fatalf("net entries = %d, want 1", len(netEntries))
	}
}

func TestNewReusesExistingDB(t *testing.T) {
	path := tempPath(t)
	defer os.Remove(path)

	db1, _ := New(path)
	db1.Log("test", "action", "first")
	db1.Close()

	db2, _ := New(path)
	defer db2.Close()

	count, _ := db2.Count()
	if count != 1 {
		t.Fatalf("reopened db count = %d, want 1", count)
	}
}

func tempPath(t *testing.T) string {
	t.Helper()
	f, err := os.CreateTemp("", "vault-test-*.db")
	if err != nil {
		t.Fatal(err)
	}
	path := f.Name()
	f.Close()
	os.Remove(path)
	return path
}
