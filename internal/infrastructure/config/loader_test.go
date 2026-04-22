package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestSave_CreatesFile(t *testing.T) {
	dir := t.TempDir()
	cfg := defaults()
	cfg.DataDir = dir
	cfg.ThemeName = "dracula"

	if err := Save(&cfg); err != nil {
		t.Fatal(err)
	}

	path := filepath.Join(dir, "config.json")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read saved config: %v", err)
	}

	var loaded Config
	if err := json.Unmarshal(data, &loaded); err != nil {
		t.Fatalf("parse saved config: %v", err)
	}
	if loaded.ThemeName != "dracula" {
		t.Errorf("theme = %q, want dracula", loaded.ThemeName)
	}
}

func TestSave_Roundtrip(t *testing.T) {
	dir := t.TempDir()
	cfg := defaults()
	cfg.DataDir = dir
	cfg.LogLevel = "debug"
	cfg.LastList = "list-123"
	cfg.Remotes = map[string]*RemoteConfig{
		"work": {Driver: "mysql", Host: "db.example.com", Port: 3307, Database: "tork", Username: "alice"},
	}
	cfg.LastRemote = "work"

	if err := Save(&cfg); err != nil {
		t.Fatal(err)
	}

	data, _ := os.ReadFile(filepath.Join(dir, "config.json"))
	var loaded Config
	json.Unmarshal(data, &loaded)

	if loaded.LogLevel != "debug" {
		t.Errorf("log_level = %q", loaded.LogLevel)
	}
	if loaded.LastList != "list-123" {
		t.Errorf("last_list = %q", loaded.LastList)
	}
	if loaded.LastRemote != "work" {
		t.Errorf("last_remote = %q", loaded.LastRemote)
	}
	if loaded.Remotes["work"] == nil {
		t.Fatal("remote 'work' is nil")
	}
	if loaded.Remotes["work"].Port != 3307 {
		t.Errorf("port = %d", loaded.Remotes["work"].Port)
	}
}

func TestSave_BadDir(t *testing.T) {
	cfg := defaults()
	cfg.DataDir = "/nonexistent/deeply/nested/path"
	// MkdirAll on /nonexistent should fail only on read-only FS, but
	// we test that Save doesn't panic.
	_ = Save(&cfg) // may or may not error depending on permissions
}

func TestResolveDataDir_CustomDir(t *testing.T) {
	dir := t.TempDir()
	cfg := &Config{DataDir: dir}
	got, err := cfg.ResolveDataDir()
	if err != nil {
		t.Fatal(err)
	}
	if got != dir {
		t.Errorf("data dir = %q, want %q", got, dir)
	}
}

func TestResolveDataDir_Default(t *testing.T) {
	cfg := &Config{}
	got, err := cfg.ResolveDataDir()
	if err != nil {
		t.Fatal(err)
	}
	home, _ := os.UserHomeDir()
	want := filepath.Join(home, ".tork")
	if got != want {
		t.Errorf("data dir = %q, want %q", got, want)
	}
}

func TestRemoteNames_Sorted(t *testing.T) {
	cfg := &Config{
		Remotes: map[string]*RemoteConfig{
			"zeta":  {Driver: "mysql", Host: "z"},
			"alpha": {Driver: "mysql", Host: "a"},
			"mid":   {Driver: "mysql", Host: "m"},
		},
	}
	names := cfg.RemoteNames()
	if len(names) != 3 {
		t.Fatalf("got %d names", len(names))
	}
	if names[0] != "alpha" || names[1] != "mid" || names[2] != "zeta" {
		t.Errorf("names = %v, want [alpha mid zeta]", names)
	}
}

func TestRemoteNames_Empty(t *testing.T) {
	cfg := &Config{}
	names := cfg.RemoteNames()
	if len(names) != 0 {
		t.Errorf("expected 0 names, got %d", len(names))
	}
}

func TestDefaults(t *testing.T) {
	cfg := defaults()
	if cfg.ThemeName != "default" {
		t.Errorf("theme = %q", cfg.ThemeName)
	}
	if len(cfg.Statuses) != 4 {
		t.Errorf("statuses count = %d, want 4", len(cfg.Statuses))
	}
	if len(cfg.Priorities) != 4 {
		t.Errorf("priorities count = %d, want 4", len(cfg.Priorities))
	}
	if cfg.Display.DateFormat != "02-01-2006" {
		t.Errorf("date format = %q", cfg.Display.DateFormat)
	}
	if cfg.Keybindings.Quit != "q" {
		t.Errorf("quit key = %q", cfg.Keybindings.Quit)
	}
}

func TestParseDSN(t *testing.T) {
	tests := []struct {
		dsn          string
		fallbackUser string
		wantHost     string
		wantPort     int
		wantDB       string
		wantUser     string
	}{
		{
			dsn:          "alice:secret@tcp(db.example.com:3307)/mydb",
			fallbackUser: "fallback",
			wantHost:     "db.example.com",
			wantPort:     3307,
			wantDB:       "mydb",
			wantUser:     "alice",
		},
		{
			dsn:          "root:pw@tcp(localhost:3306)/tork?parseTime=true",
			fallbackUser: "",
			wantHost:     "localhost",
			wantPort:     3306,
			wantDB:       "tork",
			wantUser:     "root",
		},
		{
			dsn:          "user@tcp(host)/db",
			fallbackUser: "",
			wantHost:     "host",
			wantPort:     3306,
			wantDB:       "db",
			wantUser:     "user",
		},
		{
			dsn:          "",
			fallbackUser: "fallback",
			wantHost:     "localhost",
			wantPort:     3306,
			wantDB:       "tork",
			wantUser:     "fallback",
		},
	}

	for _, tt := range tests {
		host, port, db, user := parseDSN(tt.dsn, tt.fallbackUser)
		if host != tt.wantHost {
			t.Errorf("parseDSN(%q) host = %q, want %q", tt.dsn, host, tt.wantHost)
		}
		if port != tt.wantPort {
			t.Errorf("parseDSN(%q) port = %d, want %d", tt.dsn, port, tt.wantPort)
		}
		if db != tt.wantDB {
			t.Errorf("parseDSN(%q) db = %q, want %q", tt.dsn, db, tt.wantDB)
		}
		if user != tt.wantUser {
			t.Errorf("parseDSN(%q) user = %q, want %q", tt.dsn, user, tt.wantUser)
		}
	}
}

func TestStatusHelpers(t *testing.T) {
	defs := []StatusDef{
		{Name: "todo", Label: "Todo"},
		{Name: "done", Label: "Done"},
	}

	if StatusLabel(defs, "todo") != "Todo" {
		t.Error("StatusLabel(todo) wrong")
	}
	if StatusLabel(defs, "unknown") != "unknown" {
		t.Error("StatusLabel fallback wrong")
	}
	if DefaultStatus(defs) != "todo" {
		t.Error("DefaultStatus wrong")
	}
	if DefaultStatus(nil) != "todo" {
		t.Error("DefaultStatus nil fallback wrong")
	}
	if !ValidStatus(defs, "done") {
		t.Error("ValidStatus(done) should be true")
	}
	if ValidStatus(defs, "nope") {
		t.Error("ValidStatus(nope) should be false")
	}
	names := StatusNames(defs)
	if len(names) != 2 || names[0] != "todo" || names[1] != "done" {
		t.Errorf("StatusNames = %v", names)
	}
}

func TestPriorityHelpers(t *testing.T) {
	defs := []PriorityDef{
		{Name: "low", Value: 1, Label: "Low"},
		{Name: "medium", Value: 2, Label: "Medium"},
		{Name: "high", Value: 3, Label: "High"},
	}

	if PriorityLabel(defs, 2) != "Medium" {
		t.Error("PriorityLabel(2) wrong")
	}
	if PriorityLabel(defs, 99) != "99" {
		t.Error("PriorityLabel fallback wrong")
	}
	if DefaultPriority(defs) != 2 {
		t.Errorf("DefaultPriority = %d, want 2", DefaultPriority(defs))
	}

	v, ok := PriorityByName(defs, "high")
	if !ok || v != 3 {
		t.Errorf("PriorityByName(high) = %d, %v", v, ok)
	}
	v, ok = PriorityByName(defs, "1") // position-based
	if !ok || v != 1 {
		t.Errorf("PriorityByName(1) = %d, %v", v, ok)
	}
	_, ok = PriorityByName(defs, "unknown")
	if ok {
		t.Error("PriorityByName(unknown) should return false")
	}
}

func TestApplyTheme(t *testing.T) {
	cfg := defaults()
	tf := BuiltinThemes["dracula"]
	ApplyTheme(&cfg, tf)

	if cfg.Theme.Primary != tf.Colors.Primary {
		t.Errorf("primary = %q, want %q", cfg.Theme.Primary, tf.Colors.Primary)
	}
	if cfg.Theme.Border != tf.Border {
		t.Errorf("border = %q, want %q", cfg.Theme.Border, tf.Border)
	}
}

func TestApplyTheme_DefaultBorder(t *testing.T) {
	cfg := defaults()
	tf := ThemeFile{Name: "noborder", Colors: ThemeConfig{Primary: "#fff"}}
	ApplyTheme(&cfg, tf)
	if cfg.Theme.Border != "rounded" {
		t.Errorf("border = %q, want rounded", cfg.Theme.Border)
	}
}

func TestLoad_NoConfigFile(t *testing.T) {
	// Point HOME to an empty temp dir so Load finds no config.json.
	dir := t.TempDir()
	t.Setenv("HOME", dir)

	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg == nil {
		t.Fatal("cfg is nil")
	}
	if cfg.ThemeName != "default" {
		t.Errorf("theme = %q, want default", cfg.ThemeName)
	}
	// It should have created the file.
	path := filepath.Join(dir, ".tork", "config.json")
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Error("expected config.json to be created on first run")
	}
}

func TestLoad_WithConfigFile(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)

	configDir := filepath.Join(dir, ".tork")
	os.MkdirAll(configDir, 0o755)

	cfg := defaults()
	cfg.ThemeName = "dracula"
	cfg.LogLevel = "debug"
	data, _ := json.MarshalIndent(cfg, "", "  ")
	os.WriteFile(filepath.Join(configDir, "config.json"), data, 0o644)

	loaded, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if loaded.ThemeName != "dracula" {
		t.Errorf("theme = %q, want dracula", loaded.ThemeName)
	}
	if loaded.LogLevel != "debug" {
		t.Errorf("log_level = %q", loaded.LogLevel)
	}
}

func TestLoad_BadJSON(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)

	configDir := filepath.Join(dir, ".tork")
	os.MkdirAll(configDir, 0o755)
	os.WriteFile(filepath.Join(configDir, "config.json"), []byte("{invalid"), 0o644)

	_, err := Load()
	if err == nil {
		t.Error("expected error for bad JSON")
	}
}

func TestLoad_OldInlineThemeMigration(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)

	configDir := filepath.Join(dir, ".tork")
	os.MkdirAll(configDir, 0o755)

	// Write config with old inline theme object (should be migrated to "default").
	configJSON := `{"theme": {"primary": "#ff0000"}, "log_level": "info"}`
	os.WriteFile(filepath.Join(configDir, "config.json"), []byte(configJSON), 0o644)

	loaded, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if loaded.ThemeName != "default" {
		t.Errorf("theme = %q, want default after migration", loaded.ThemeName)
	}
}

func TestLoad_RemoteDBMigration(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)

	configDir := filepath.Join(dir, ".tork")
	os.MkdirAll(configDir, 0o755)

	configJSON := `{
		"theme": "default",
		"remote_db": {"driver": "mysql", "dsn": "alice:pw@tcp(db.host:3307)/mydb?parseTime=true"},
		"username": "alice"
	}`
	os.WriteFile(filepath.Join(configDir, "config.json"), []byte(configJSON), 0o644)

	loaded, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if loaded.RemoteDB != nil {
		t.Error("remote_db should be nil after migration")
	}
	if loaded.Remotes["default"] == nil {
		t.Fatal("expected 'default' remote after migration")
	}
	if loaded.Remotes["default"].Host != "db.host" {
		t.Errorf("host = %q", loaded.Remotes["default"].Host)
	}
	if loaded.LastRemote != "default" {
		t.Errorf("last_remote = %q", loaded.LastRemote)
	}
}

func TestListThemes_IncludesUserThemes(t *testing.T) {
	dir := t.TempDir()
	themesDir := filepath.Join(dir, "themes")
	os.MkdirAll(themesDir, 0o755)

	// Create a user theme file.
	os.WriteFile(filepath.Join(themesDir, "custom.json"), []byte(`{"name":"custom"}`), 0o644)

	names := ListThemes(dir)
	found := false
	for _, n := range names {
		if n == "custom" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected 'custom' in themes, got %v", names)
	}
}

func TestDefaultPriority_Nil(t *testing.T) {
	if DefaultPriority(nil) != 1 {
		t.Errorf("DefaultPriority(nil) = %d, want 1", DefaultPriority(nil))
	}
}
