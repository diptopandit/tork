package config

import (
	"encoding/json"
	"testing"
)

func TestConfig_RemoteDB_MarshalRoundtrip(t *testing.T) {
	cfg := defaults()
	cfg.RemoteDB = &RemoteDBConfig{
		Driver: "mysql",
		DSN:    "user:pass@tcp(localhost:3306)/tork",
	}
	cfg.UserID = "test-uuid-1234"
	cfg.Username = "alice"

	data, err := json.Marshal(cfg)
	if err != nil {
		t.Fatal(err)
	}

	var got Config
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatal(err)
	}

	if got.RemoteDB == nil {
		t.Fatal("RemoteDB is nil after roundtrip")
	}
	if got.RemoteDB.Driver != "mysql" {
		t.Errorf("driver = %q, want mysql", got.RemoteDB.Driver)
	}
	if got.RemoteDB.DSN != "user:pass@tcp(localhost:3306)/tork" {
		t.Errorf("dsn = %q", got.RemoteDB.DSN)
	}
	if got.UserID != "test-uuid-1234" {
		t.Errorf("user_id = %q", got.UserID)
	}
	if got.Username != "alice" {
		t.Errorf("username = %q", got.Username)
	}
}

func TestConfig_RemoteDB_OmitEmpty(t *testing.T) {
	cfg := defaults()
	// No RemoteDB set — should be omitted from JSON.
	data, err := json.Marshal(cfg)
	if err != nil {
		t.Fatal(err)
	}

	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatal(err)
	}
	if _, ok := raw["remote_db"]; ok {
		t.Error("remote_db should be omitted when nil")
	}
	if _, ok := raw["user_id"]; ok {
		t.Error("user_id should be omitted when empty")
	}
}

func TestConfig_RemoteDB_IsRemote(t *testing.T) {
	// No remote config — should use SQLite path.
	cfg := defaults()
	if cfg.RemoteDB != nil {
		t.Error("default config should have nil RemoteDB")
	}

	// With remote config.
	cfg.RemoteDB = &RemoteDBConfig{Driver: "mysql", DSN: "test"}
	if cfg.RemoteDB == nil || cfg.RemoteDB.DSN == "" {
		t.Error("expected non-empty RemoteDB")
	}
}
