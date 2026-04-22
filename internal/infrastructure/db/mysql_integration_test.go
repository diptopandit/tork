package db

import (
	"os"
	"testing"

	_ "github.com/go-sql-driver/mysql"
)

// mysqlDSN returns the DSN for the test MySQL server.
// Defaults to the Docker-based test instance used by make test-integration.
func mysqlDSN(t *testing.T) string {
	t.Helper()
	dsn := os.Getenv("TORK_MYSQL_DSN")
	if dsn == "" {
		dsn = "root:tork@tcp(127.0.0.1:3306)/tork_test"
	}
	return dsn
}

func TestOpenMySQL_Success(t *testing.T) {
	dsn := mysqlDSN(t)
	conn, err := OpenMySQL(dsn)
	if err != nil {
		t.Skipf("MySQL not available: %v", err)
	}
	defer conn.Close()

	if err := conn.Ping(); err != nil {
		t.Fatalf("Ping failed: %v", err)
	}
}

func TestOpenMySQL_BadDSN(t *testing.T) {
	// Use an unreachable host/port to trigger a connect error quickly.
	_, err := OpenMySQL("nobody:nope@tcp(127.0.0.1:1)/nodb")
	if err == nil {
		t.Error("expected error for bad DSN")
	}
}

func TestMigrateMySQL_Success(t *testing.T) {
	dsn := mysqlDSN(t)
	conn, err := OpenMySQL(dsn)
	if err != nil {
		t.Skipf("MySQL not available: %v", err)
	}
	defer conn.Close()

	if err := MigrateMySQL(conn); err != nil {
		t.Fatalf("MigrateMySQL: %v", err)
	}
}

func TestMigrateMySQL_Idempotent(t *testing.T) {
	dsn := mysqlDSN(t)
	conn, err := OpenMySQL(dsn)
	if err != nil {
		t.Skipf("MySQL not available: %v", err)
	}
	defer conn.Close()

	// Run twice — should not error.
	if err := MigrateMySQL(conn); err != nil {
		t.Fatalf("first migrate: %v", err)
	}
	if err := MigrateMySQL(conn); err != nil {
		t.Fatalf("second migrate: %v", err)
	}
}

func TestMigrateMySQL_CreatesAllTables(t *testing.T) {
	dsn := mysqlDSN(t)
	conn, err := OpenMySQL(dsn)
	if err != nil {
		t.Skipf("MySQL not available: %v", err)
	}
	defer conn.Close()

	if err := MigrateMySQL(conn); err != nil {
		t.Fatal(err)
	}

	expected := []string{"users", "task_lists", "list_members", "tasks", "task_updates"}
	for _, tbl := range expected {
		var name string
		err := conn.QueryRow("SELECT TABLE_NAME FROM information_schema.TABLES WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME=?", tbl).Scan(&name)
		if err != nil {
			t.Errorf("table %q not found: %v", tbl, err)
		}
	}
}
