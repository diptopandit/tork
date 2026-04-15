package bootstrap

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/diptopandit/tork/internal/infrastructure/config"
)

func TestInit_SQLitePath(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &config.Config{
		DataDir: tmpDir,
	}
	// No Remotes — should use SQLite.
	svc, err := Init(cfg, "local", "", nil)
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

	// Verify the SQLite file was created.
	dbPath := filepath.Join(tmpDir, "tork.db")
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		t.Error("expected tork.db to be created")
	}
}

func TestInit_SQLitePath_CreatesList(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &config.Config{
		DataDir: tmpDir,
	}
	svc, err := Init(cfg, "local", "", nil)
	if err != nil {
		t.Fatal(err)
	}
	defer svc.DB.Close()

	// Should be able to create a list and task.
	list, err := svc.ListSvc.CreateList("Test", nil)
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

func TestInit_RemoteDB_FailsWithBadDSN(t *testing.T) {
	cfg := &config.Config{
		DataDir: t.TempDir(),
		Remotes: map[string]*config.RemoteConfig{
			"bad": {
				Driver:   "mysql",
				Host:     "127.0.0.1",
				Port:     9999,
				Database: "nonexistent",
				Username: "invalid",
			},
		},
	}
	_, err := Init(cfg, "bad", "invalid", nil)
	if err == nil {
		t.Error("expected error for invalid MySQL DSN")
	}
}
