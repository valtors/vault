package store

import (
	"database/sql"
	"fmt"
	"os"
	"sync"
	"time"

	_ "modernc.org/sqlite"
)

type DB struct {
	db  *sql.DB
	mu  sync.Mutex
}

type Entry struct {
	ID        int64     `json:"id"`
	Timestamp time.Time `json:"timestamp"`
	Category  string    `json:"category"`
	Action    string    `json:"action"`
	Detail    string    `json:"detail"`
}

func New(path string) (*DB, error) {
	dsn := fmt.Sprintf("file:%s?_journal_mode=WAL&_busy_timeout=5000&_synchronous=NORMAL", path)
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("open %s: %w", path, err)
	}

	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("ping %s: %w", path, err)
	}

	schema := `
	CREATE TABLE IF NOT EXISTS audit (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		timestamp DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		category TEXT NOT NULL,
		action TEXT NOT NULL,
		detail TEXT
	);
	CREATE INDEX IF NOT EXISTS idx_audit_category ON audit(category);
	CREATE INDEX IF NOT EXISTS idx_audit_timestamp ON audit(timestamp);
	`
	if _, err := db.Exec(schema); err != nil {
		db.Close()
		return nil, fmt.Errorf("create schema: %w", err)
	}

	return &DB{db: db}, nil
}

func (d *DB) Log(category, action, detail string) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	_, err := d.db.Exec(
		"INSERT INTO audit (category, action, detail) VALUES (?, ?, ?)",
		category, action, detail,
	)
	return err
}

func (d *DB) Query(category string, limit int) ([]Entry, error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	if limit <= 0 || limit > 10000 {
		limit = 100
	}

	var rows *sql.Rows
	var err error
	if category == "" {
		rows, err = d.db.Query(
			"SELECT id, timestamp, category, action, detail FROM audit ORDER BY id DESC LIMIT ?",
			limit,
		)
	} else {
		rows, err = d.db.Query(
			"SELECT id, timestamp, category, action, detail FROM audit WHERE category = ? ORDER BY id DESC LIMIT ?",
			category, limit,
		)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entries []Entry
	for rows.Next() {
		var e Entry
		if err := rows.Scan(&e.ID, &e.Timestamp, &e.Category, &e.Action, &e.Detail); err != nil {
			return nil, err
		}
		entries = append(entries, e)
	}
	return entries, rows.Err()
}

func (d *DB) Count() (int, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	var count int
	err := d.db.QueryRow("SELECT COUNT(*) FROM audit").Scan(&count)
	return count, err
}

func (d *DB) Close() error {
	return d.db.Close()
}

func TempDB(t interface{ Cleanup() }) (*DB, error) {
	_ = t
	f, err := os.CreateTemp("", "vault-test-*.db")
	if err != nil {
		return nil, err
	}
	path := f.Name()
	f.Close()
	db, err := New(path)
	if err != nil {
		os.Remove(path)
		return nil, err
	}
	return db, nil
}
