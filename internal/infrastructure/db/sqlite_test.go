package db

import (
	"database/sql"
	"os"
	"path/filepath"
	"testing"

	_ "modernc.org/sqlite"
)

func TestOpen_CreatesDBFile(t *testing.T) {
	dir := t.TempDir()
	conn, err := Open(dir)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer conn.Close()

	// Ping to force the driver to create the file.
	if err := conn.Ping(); err != nil {
		t.Fatalf("Ping: %v", err)
	}

	dbPath := filepath.Join(dir, "tork.db")
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		t.Error("expected tork.db to exist")
	}
}

func TestOpen_CreatesDataDir(t *testing.T) {
	parent := t.TempDir()
	dir := filepath.Join(parent, "subdir", "nested")
	conn, err := Open(dir)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer conn.Close()

	if _, err := os.Stat(dir); os.IsNotExist(err) {
		t.Error("expected nested data dir to exist")
	}
}

func TestOpen_PingSucceeds(t *testing.T) {
	dir := t.TempDir()
	conn, err := Open(dir)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer conn.Close()

	if err := conn.Ping(); err != nil {
		t.Errorf("Ping failed: %v", err)
	}
}

func TestMigrate_Success(t *testing.T) {
	conn, err := sql.Open("sqlite", ":memory:?_foreign_keys=on")
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	if err := Migrate(conn); err != nil {
		t.Fatalf("Migrate: %v", err)
	}

	// Verify tables exist.
	for _, table := range []string{"task_lists", "tasks", "task_updates", "tasks_fts"} {
		var name string
		err := conn.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name=?", table).Scan(&name)
		if err != nil {
			t.Errorf("table %q not found after Migrate: %v", table, err)
		}
	}
}

func TestMigrate_Idempotent(t *testing.T) {
	conn, err := sql.Open("sqlite", ":memory:?_foreign_keys=on")
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	// Run migrate twice — should not error.
	if err := Migrate(conn); err != nil {
		t.Fatalf("first Migrate: %v", err)
	}
	if err := Migrate(conn); err != nil {
		t.Fatalf("second Migrate: %v", err)
	}
}

func TestMigrate_NumIDBackfill(t *testing.T) {
	conn, err := sql.Open("sqlite", ":memory:?_foreign_keys=on")
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	if err := Migrate(conn); err != nil {
		t.Fatal(err)
	}

	// Insert a list and a task with NULL num_id.
	conn.Exec("INSERT INTO task_lists (id, name, schema_json, created_at) VALUES ('l1', 'Test', '{}', datetime('now'))")
	conn.Exec(`INSERT INTO tasks (id, list_id, title, description, status, priority, tags, custom_fields, depends_on, created_at, updated_at)
		VALUES ('t1', 'l1', 'Task', '', 'todo', 2, '[]', '{}', '[]', datetime('now'), datetime('now'))`)

	// Run migrate again to trigger backfill.
	if err := Migrate(conn); err != nil {
		t.Fatal(err)
	}

	var numID sql.NullInt64
	conn.QueryRow("SELECT num_id FROM tasks WHERE id='t1'").Scan(&numID)
	if !numID.Valid {
		t.Error("expected num_id to be backfilled")
	}
}
