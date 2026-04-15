package bootstrap

import (
	"database/sql"
	"os"
	"testing"

	_ "github.com/go-sql-driver/mysql"

	"github.com/diptopandit/tork/internal/infrastructure/config"
	"github.com/diptopandit/tork/internal/infrastructure/db"
)

// testDSN returns the raw DSN string for connecting to the test MySQL server.
func testDSN(t *testing.T) string {
	t.Helper()
	dsn := os.Getenv("TORK_MYSQL_DSN")
	if dsn == "" {
		dsn = "root:tork@tcp(127.0.0.1:3306)/tork_test"
	}
	return dsn
}

// testRemoteConfig builds a RemoteConfig + password from the test DSN.
// Default: host=127.0.0.1, port=3306, database=tork_test, username=root, password=tork.
func testRemoteConfig(t *testing.T) (*config.RemoteConfig, string) {
	t.Helper()
	return &config.RemoteConfig{
		Driver:   "mysql",
		Host:     "127.0.0.1",
		Port:     3306,
		Database: "tork_test",
		Username: "root",
	}, "tork"
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
	dsn := testDSN(t)

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

	rc, password := testRemoteConfig(t)
	cfg := &config.Config{
		DataDir: t.TempDir(),
		Remotes: map[string]*config.RemoteConfig{"test": rc},
	}

	svc, err := Init(cfg, "test", password)
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

func TestInit_MySQL_UsernameAsIdentity(t *testing.T) {
	dsn := testDSN(t)

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

	rc, password := testRemoteConfig(t)
	cfg := &config.Config{
		DataDir: t.TempDir(),
		Remotes: map[string]*config.RemoteConfig{"test": rc},
	}

	svc, err := Init(cfg, "test", password)
	if err != nil {
		t.Fatal(err)
	}
	defer svc.DB.Close()

	// Username ("root") is used as the user identity — verify a list is scoped to it.
	_, err = svc.ListSvc.CreateList("My List", nil)
	if err != nil {
		t.Fatal(err)
	}

	lists, err := svc.ListSvc.GetAllLists()
	if err != nil {
		t.Fatal(err)
	}
	if len(lists) != 1 {
		t.Errorf("got %d lists, want 1", len(lists))
	}
	if lists[0].OwnerID != "root" {
		t.Errorf("owner_id = %q, want 'root'", lists[0].OwnerID)
	}
}
