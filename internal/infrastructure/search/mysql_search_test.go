package search

import (
	"database/sql"
	"os"
	"testing"
	"time"

	_ "github.com/go-sql-driver/mysql"

	"github.com/diptopandit/tork/internal/infrastructure/db"
)

func mysqlDSN(t *testing.T) string {
	t.Helper()
	dsn := os.Getenv("TORK_MYSQL_DSN")
	if dsn == "" {
		dsn = "root:tork@tcp(127.0.0.1:3306)/tork_test"
	}
	return dsn
}

func openMySQLSearchDB(t *testing.T) *sql.DB {
	t.Helper()
	dsn := mysqlDSN(t)
	conn, err := db.OpenMySQL(dsn)
	if err != nil {
		t.Skipf("MySQL not available: %v", err)
	}
	// Clean up.
	for _, tbl := range []string{"task_updates", "tasks", "list_members", "task_lists", "users"} {
		conn.Exec("DELETE FROM " + tbl)
	}
	t.Cleanup(func() { conn.Close() })
	return conn
}

func TestMySQLSearch_Search(t *testing.T) {
	conn := openMySQLSearchDB(t)

	// Seed a user and list.
	now := time.Now().UTC()
	conn.Exec("INSERT INTO users (id, username, created_at) VALUES (?, ?, ?)", "search-user", "searcher", now)
	conn.Exec("INSERT INTO task_lists (id, name, schema_json, owner_id, visibility, created_at) VALUES (?, ?, '{}', ?, 'private', ?)",
		"search-list", "Search List", "search-user", now)

	// Seed tasks with searchable content.
	conn.Exec(`INSERT INTO tasks (id, num_id, list_id, title, description, status, priority, tags, custom_fields, depends_on, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, 'todo', 2, '[]', '{}', '[]', ?, ?)`,
		"st1", 1, "search-list", "Deploy the application server", "Production deploy steps", now, now)
	conn.Exec(`INSERT INTO tasks (id, num_id, list_id, title, description, status, priority, tags, custom_fields, depends_on, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, 'todo', 2, '[]', '{}', '[]', ?, ?)`,
		"st2", 2, "search-list", "Write unit tests", "Cover the new API endpoints", now, now)
	conn.Exec(`INSERT INTO tasks (id, num_id, list_id, title, description, status, priority, tags, custom_fields, depends_on, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, 'todo', 2, '[]', '{}', '[]', ?, ?)`,
		"st3", 3, "search-list", "Fix database migration bug", "The migration fails on MySQL 8", now, now)

	svc := NewMySQLSearch(conn)

	// Search for "deploy" — should find st1.
	ids, err := svc.Search("deploy")
	if err != nil {
		t.Fatal(err)
	}
	if len(ids) == 0 {
		t.Error("expected at least 1 result for 'deploy'")
	}
	found := false
	for _, id := range ids {
		if id == "st1" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected st1 in results, got %v", ids)
	}

	// Search for "migration" — should find st3.
	ids2, err := svc.Search("migration")
	if err != nil {
		t.Fatal(err)
	}
	found2 := false
	for _, id := range ids2 {
		if id == "st3" {
			found2 = true
		}
	}
	if !found2 {
		t.Errorf("expected st3 in results for 'migration', got %v", ids2)
	}
}

func TestMySQLSearch_IndexAndDelete_NoOp(t *testing.T) {
	conn := openMySQLSearchDB(t)
	svc := NewMySQLSearch(conn)

	// Index and Delete are no-ops — should not error.
	if err := svc.Index(nil); err != nil {
		t.Errorf("Index should be a no-op, got: %v", err)
	}
	if err := svc.Delete("nonexistent"); err != nil {
		t.Errorf("Delete should be a no-op, got: %v", err)
	}
}
