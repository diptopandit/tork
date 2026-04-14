package bootstrap

import (
	"database/sql"
	"os"
	"testing"

	_ "github.com/go-sql-driver/mysql"

	"github.com/diptopandit/tork/internal/infrastructure/config"
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

func cleanMySQL(t *testing.T, dsn string) {
	t.Helper()
	conn, err := db.OpenMySQL(dsn)
	if err != nil {
		t.Skipf("MySQL not available: %v", err)
	}
	defer conn.Close()
	for _, tbl := range []string{"task_updates", "tasks", "list_members", "task_lists", "users"} {
		conn.Exec("DELETE FROM " + tbl)
	}
}

func TestInit_MySQLPath(t *testing.T) {
	dsn := mysqlDSN(t)

	// Attempt connection to verify MySQL is available.
	testConn, err := sql.Open("mysql", dsn+"?parseTime=true")
	if err != nil {
		t.Skipf("MySQL not available: %v", err)
	}
	if err := testConn.Ping(); err != nil {
		testConn.Close()
		t.Skipf("MySQL not available: %v", err)
	}
	testConn.Close()

	cleanMySQL(t, dsn)

	cfg := &config.Config{
		DataDir: t.TempDir(),
		RemoteDB: &config.RemoteDBConfig{
			Driver: "mysql",
			DSN:    dsn,
		},
		Username: "integration-test-user",
	}

	svc, err := Init(cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer svc.DB.Close()

	if svc.TaskSvc == nil {
		t.Error("TaskSvc is nil")
	}
	if svc.ListSvc == nil {
		t.Error("ListSvc is nil")
	}

	// UserID should have been auto-generated and set.
	if cfg.UserID == "" {
		t.Error("UserID should be auto-generated")
	}

	// Create a list and task through services.
	list, err := svc.ListSvc.CreateList("Integration Test", nil)
	if err != nil {
		t.Fatal(err)
	}
	if list.ID == "" {
		t.Error("list ID is empty")
	}

	lists, err := svc.ListSvc.GetAllLists()
	if err != nil {
		t.Fatal(err)
	}
	if len(lists) != 1 {
		t.Errorf("got %d lists, want 1", len(lists))
	}
}

func TestInit_MySQL_AutoGeneratesUserID(t *testing.T) {
	dsn := mysqlDSN(t)

	testConn, err := sql.Open("mysql", dsn+"?parseTime=true")
	if err != nil {
		t.Skipf("MySQL not available: %v", err)
	}
	if err := testConn.Ping(); err != nil {
		testConn.Close()
		t.Skipf("MySQL not available: %v", err)
	}
	testConn.Close()

	cleanMySQL(t, dsn)

	cfg := &config.Config{
		DataDir: t.TempDir(),
		RemoteDB: &config.RemoteDBConfig{
			Driver: "mysql",
			DSN:    dsn,
		},
		// No UserID, no Username — should be auto-generated.
	}

	svc, err := Init(cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer svc.DB.Close()

	if cfg.UserID == "" {
		t.Error("expected auto-generated UserID")
	}
	if cfg.Username == "" {
		t.Error("expected auto-generated Username")
	}
}
