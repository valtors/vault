package store

import (
	"database/sql"
	"fmt"
	"os"
	"time"

	_ "modernc.org/sqlite"
)

type DB struct {
	db *sql.DB
}

func New(path string) (*DB, error) {
	if _, err := os.Stat(path); err == nil {
		os.Remove(path)
	}

	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open audit db: %w", err)
	}

	schema := `
	CREATE TABLE IF NOT EXISTS audit (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		ts TEXT NOT NULL,
		source TEXT NOT NULL,
		event TEXT NOT NULL,
		detail TEXT
	);
	CREATE INDEX IF NOT EXISTS idx_audit_ts ON audit(ts);
	CREATE INDEX IF NOT EXISTS idx_audit_source ON audit(source);
	CREATE INDEX IF NOT EXISTS idx_audit_event ON audit(event);
	`

	if _, err := db.Exec(schema); err != nil {
		return nil, fmt.Errorf("init audit schema: %w", err)
	}

	return &DB{db: db}, nil
}

func (d *DB) Log(source, event, detail string) {
	d.db.Exec("INSERT INTO audit (ts, source, event, detail) VALUES (?, ?, ?, ?)",
		time.Now().UTC().Format(time.RFC3339Nano), source, event, detail)
}

func (d *DB) Query(source, event string, limit int) ([]Entry, error) {
	q := "SELECT ts, source, event, detail FROM audit WHERE 1=1"
	args := []interface{}{}

	if source != "" {
		q += " AND source = ?"
		args = append(args, source)
	}
	if event != "" {
		q += " AND event = ?"
		args = append(args, event)
	}
	q += " ORDER BY id DESC LIMIT ?"
	args = append(args, limit)

	rows, err := d.db.Query(q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entries []Entry
	for rows.Next() {
		var e Entry
		var detail sql.NullString
		if err := rows.Scan(&e.Timestamp, &e.Source, &e.Event, &detail); err != nil {
			continue
		}
		e.Detail = detail.String
		entries = append(entries, e)
	}
	return entries, nil
}

func (d *DB) Close() error {
	return d.db.Close()
}

type Entry struct {
	Timestamp string `json:"ts"`
	Source    string `json:"source"`
	Event     string `json:"event"`
	Detail    string `json:"detail"`
}
