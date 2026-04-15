package config

import (
	"encoding/json"
	"testing"
)

func TestConfig_Remotes_MarshalRoundtrip(t *testing.T) {
	cfg := defaults()
	cfg.Remotes = map[string]*RemoteConfig{
		"work": {
			Driver:   "mysql",
			Host:     "db.example.com",
			Port:     3307,
			Database: "mytork",
			Username: "alice",
		},
	}
	cfg.LastRemote = "work"

	data, err := json.Marshal(cfg)
	if err != nil {
		t.Fatal(err)
	}

	var got Config
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatal(err)
	}

	rc := got.ActiveRemote("work")
	if rc == nil {
		t.Fatal("remote 'work' is nil after roundtrip")
	}
	if rc.Driver != "mysql" {
		t.Errorf("driver = %q, want mysql", rc.Driver)
	}
	if rc.Host != "db.example.com" {
		t.Errorf("host = %q", rc.Host)
	}
	if rc.Port != 3307 {
		t.Errorf("port = %d, want 3307", rc.Port)
	}
	if rc.Database != "mytork" {
		t.Errorf("database = %q", rc.Database)
	}
	if rc.Username != "alice" {
		t.Errorf("username = %q", rc.Username)
	}
	if got.LastRemote != "work" {
		t.Errorf("last_remote = %q, want work", got.LastRemote)
	}

	// Verify BuildDSN.
	dsn := rc.BuildDSN("secret")
	if dsn != "alice:secret@tcp(db.example.com:3307)/mytork" {
		t.Errorf("BuildDSN = %q", dsn)
	}
}

func TestConfig_RemoteConfig_Defaults(t *testing.T) {
	rc := &RemoteConfig{Driver: "mysql", Host: "localhost", Username: "root"}
	if rc.EffectivePort() != 3306 {
		t.Errorf("default port = %d, want 3306", rc.EffectivePort())
	}
	if rc.EffectiveDatabase() != "tork" {
		t.Errorf("default database = %q, want tork", rc.EffectiveDatabase())
	}
	dsn := rc.BuildDSN("pw")
	if dsn != "root:pw@tcp(localhost:3306)/tork" {
		t.Errorf("BuildDSN = %q", dsn)
	}
	label := rc.Label()
	if label != "root@localhost:3306/tork" {
		t.Errorf("Label = %q", label)
	}
}

func TestConfig_Remotes_OmitEmpty(t *testing.T) {
	cfg := defaults()
	// No Remotes set — should be omitted from JSON.
	data, err := json.Marshal(cfg)
	if err != nil {
		t.Fatal(err)
	}

	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatal(err)
	}
	if _, ok := raw["remotes"]; ok {
		t.Error("remotes should be omitted when nil")
	}
	if _, ok := raw["last_remote"]; ok {
		t.Error("last_remote should be omitted when empty")
	}
}

func TestConfig_ActiveRemote(t *testing.T) {
	cfg := defaults()
	// No remotes — ActiveRemote should return nil.
	if cfg.ActiveRemote("anything") != nil {
		t.Error("expected nil for empty remotes")
	}
	if cfg.ActiveRemote("local") != nil {
		t.Error("expected nil for 'local'")
	}
	if cfg.ActiveRemote("") != nil {
		t.Error("expected nil for empty string")
	}

	// With remote config.
	cfg.Remotes = map[string]*RemoteConfig{
		"work": {Driver: "mysql", Host: "localhost", Username: "root"},
	}
	if rc := cfg.ActiveRemote("work"); rc == nil {
		t.Error("expected non-nil for 'work'")
	}
	if rc := cfg.ActiveRemote("missing"); rc != nil {
		t.Error("expected nil for unknown remote")
	}
}

func TestConfig_ResolveRemote(t *testing.T) {
	cfg := defaults()
	cfg.Remotes = map[string]*RemoteConfig{
		"work":     {Driver: "mysql", Host: "work-host", Username: "alice"},
		"personal": {Driver: "mysql", Host: "personal-host", Username: "alice"},
	}

	// Flag --local wins.
	name, picker := cfg.ResolveRemote("", true)
	if name != "local" || picker {
		t.Errorf("--local: got %q, picker=%v", name, picker)
	}

	// Flag --remote wins.
	name, picker = cfg.ResolveRemote("work", false)
	if name != "work" || picker {
		t.Errorf("--remote work: got %q, picker=%v", name, picker)
	}

	// LastRemote set.
	cfg.LastRemote = "personal"
	name, picker = cfg.ResolveRemote("", false)
	if name != "personal" || picker {
		t.Errorf("last_remote: got %q, picker=%v", name, picker)
	}

	// Stale LastRemote → needs picker.
	cfg.LastRemote = "deleted-remote"
	name, picker = cfg.ResolveRemote("", false)
	if !picker {
		t.Error("stale last_remote should need picker")
	}

	// No preference, has remotes → needs picker.
	cfg.LastRemote = ""
	name, picker = cfg.ResolveRemote("", false)
	if !picker {
		t.Error("no preference with remotes should need picker")
	}

	// No remotes → local.
	cfg.Remotes = nil
	cfg.LastRemote = ""
	name, picker = cfg.ResolveRemote("", false)
	if name != "local" || picker {
		t.Errorf("no remotes: got %q, picker=%v", name, picker)
	}
}
