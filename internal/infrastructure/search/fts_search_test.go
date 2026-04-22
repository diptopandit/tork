package search

import (
	"database/sql"
	"testing"
	"time"

	_ "modernc.org/sqlite"

	"github.com/diptopandit/tork/internal/domain"
	"github.com/diptopandit/tork/internal/infrastructure/db"
)

func openTestDB(t *testing.T) *sql.DB {
	t.Helper()
	conn, err := sql.Open("sqlite", ":memory:?_foreign_keys=on")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := conn.Exec("PRAGMA foreign_keys = ON"); err != nil {
		t.Fatal(err)
	}
	if err := db.Migrate(conn); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { conn.Close() })
	return conn
}

func seedTask(t *testing.T, conn *sql.DB, id, title, desc string) {
	t.Helper()
	now := time.Now().UTC()
	// Seed a list first.
	conn.Exec(`INSERT OR IGNORE INTO task_lists (id, name, schema_json, created_at) VALUES ('list-1', 'Test', '{}', ?)`, now)
	// Insert the task.
	conn.Exec(`INSERT INTO tasks (id, num_id, list_id, title, description, status, priority, tags, custom_fields, depends_on, created_at, updated_at)
		VALUES (?, NULL, 'list-1', ?, ?, 'todo', 2, '[]', '{}', '[]', ?, ?)`,
		id, title, desc, now, now)
}

func TestFTSSearch_NewFTSSearch(t *testing.T) {
	conn := openTestDB(t)
	s := NewFTSSearch(conn)
	if s == nil {
		t.Fatal("expected non-nil")
	}
}

func TestFTSSearch_SearchEmpty(t *testing.T) {
	conn := openTestDB(t)
	s := NewFTSSearch(conn)
	ids, err := s.Search("nothing")
	if err != nil {
		t.Fatal(err)
	}
	if len(ids) != 0 {
		t.Errorf("expected 0 results, got %d", len(ids))
	}
}

func TestFTSSearch_SearchFindsTask(t *testing.T) {
	conn := openTestDB(t)
	seedTask(t, conn, "task-1", "Deploy the server", "Production deploy")
	seedTask(t, conn, "task-2", "Write unit tests", "Cover the API")

	s := NewFTSSearch(conn)

	ids, err := s.Search("deploy")
	if err != nil {
		t.Fatal(err)
	}
	if len(ids) == 0 {
		t.Fatal("expected at least 1 result for 'deploy'")
	}
	found := false
	for _, id := range ids {
		if id == "task-1" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected task-1 in results, got %v", ids)
	}
}

func TestFTSSearch_SearchByDescription(t *testing.T) {
	conn := openTestDB(t)
	seedTask(t, conn, "task-3", "Some task", "Kubernetes orchestration")

	s := NewFTSSearch(conn)
	ids, err := s.Search("kubernetes")
	if err != nil {
		t.Fatal(err)
	}
	if len(ids) == 0 {
		t.Fatal("expected result for 'kubernetes' in description")
	}
}

func TestFTSSearch_Index(t *testing.T) {
	conn := openTestDB(t)
	seedTask(t, conn, "task-idx", "Original title", "Original desc")

	s := NewFTSSearch(conn)

	// Re-index with updated content.
	err := s.Index(&domain.Task{
		ID:          "task-idx",
		Title:       "Updated searchable title",
		Description: "Updated desc",
	})
	if err != nil {
		t.Fatal(err)
	}

	ids, err := s.Search("searchable")
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, id := range ids {
		if id == "task-idx" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected task-idx after re-index, got %v", ids)
	}
}

func TestFTSSearch_Delete(t *testing.T) {
	conn := openTestDB(t)
	seedTask(t, conn, "task-del", "Deletable item", "Remove me")

	s := NewFTSSearch(conn)

	// Verify it's searchable first.
	ids, err := s.Search("deletable")
	if err != nil {
		t.Fatal(err)
	}
	if len(ids) == 0 {
		t.Fatal("expected result before delete")
	}

	// Delete from FTS index.
	if err := s.Delete("task-del"); err != nil {
		t.Fatal(err)
	}

	ids, err = s.Search("deletable")
	if err != nil {
		t.Fatal(err)
	}
	if len(ids) != 0 {
		t.Errorf("expected 0 results after delete, got %d", len(ids))
	}
}

func TestFTSSearch_DeleteNonExistent(t *testing.T) {
	conn := openTestDB(t)
	s := NewFTSSearch(conn)
	// Should not error.
	if err := s.Delete("nonexistent"); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}
