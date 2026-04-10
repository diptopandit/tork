package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// defaults returns a Config pre-populated with sensible values.
func defaults() Config {
	return Config{
		ThemeName: "default",
		Keybindings: KeyMap{
			Up:          "k",
			Down:        "j",
			Left:        "h",
			Right:       "l",
			Select:      "enter",
			Quit:        "q",
			Help:        "?",
			New:         "n",
			Edit:        "e",
			Delete:      "d",
			Search:      "/",
			Done:        "x",
			Status:      "s",
			ThemePicker: "T",
		},
		Display: DisplayConfig{
			Columns:        []string{"title", "status", "priority", "due_date"},
			DateFormat:     "02-01-2006",
			ShowTimestamps: true,
			TabOrder:       []string{"todo", "in_progress", "done", "all"},
			DefaultTab:     "todo",
		},
		Statuses: []StatusDef{
			{Name: "todo", Label: "Todo"},
			{Name: "in_progress", Label: "In Progress"},
			{Name: "done", Label: "Done"},
			{Name: "cancelled", Label: "Cancelled"},
		},
		Priorities: []PriorityDef{
			{Name: "low", Value: 1, Label: "Low"},
			{Name: "medium", Value: 2, Label: "Medium"},
			{Name: "high", Value: 3, Label: "High"},
			{Name: "urgent", Value: 4, Label: "Urgent"},
		},
	}
}

// ResolveTheme looks up a theme by name from built-in presets or user files
// in the configDir/themes/ directory. It validates the result before returning.
func ResolveTheme(themeName string, configDir string) (ThemeFile, error) {
	if themeName == "" {
		themeName = "default"
	}

	// Check built-in themes first.
	if t, ok := BuiltinThemes[themeName]; ok {
		return t, nil
	}

	// Check user theme file.
	themePath := filepath.Join(configDir, "themes", themeName+".json")
	data, err := os.ReadFile(themePath)
	if os.IsNotExist(err) {
		names := strings.Join(BuiltinThemeNames(), ", ")
		return ThemeFile{}, fmt.Errorf("theme %q not found (built-in themes: %s; also checked %s)", themeName, names, themePath)
	}
	if err != nil {
		return ThemeFile{}, fmt.Errorf("theme: read %s: %w", themePath, err)
	}

	var tf ThemeFile
	if err := json.Unmarshal(data, &tf); err != nil {
		return ThemeFile{}, fmt.Errorf("theme: parse %s: %w", themePath, err)
	}
	if tf.Name == "" {
		tf.Name = themeName
	}
	if err := ValidateTheme(tf); err != nil {
		return ThemeFile{}, err
	}
	return tf, nil
}

// ApplyTheme populates the runtime Theme field on cfg from a resolved ThemeFile.
func ApplyTheme(cfg *Config, tf ThemeFile) {
	cfg.Theme = tf.Colors
	cfg.Theme.Border = tf.Border
	if cfg.Theme.Border == "" {
		cfg.Theme.Border = "rounded"
	}
	cfg.Theme.StatusColors = tf.StatusColors
	cfg.Theme.PriorityColors = tf.PriorityColors
}

// Load reads ~/.tork/config.json, resolves the named theme, and merges
// everything over the defaults. If the file does not exist the defaults are
// returned without error.
func Load() (*Config, error) {
	cfg := defaults()

	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("config: resolve home: %w", err)
	}
	configDir := filepath.Join(home, ".tork")
	path := filepath.Join(configDir, "config.json")

	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		// No config file — apply default theme and return.
		tf := BuiltinThemes["default"]
		ApplyTheme(&cfg, tf)
		return &cfg, nil
	}
	if err != nil {
		return nil, fmt.Errorf("config: read file: %w", err)
	}

	// Peek at the "theme" field to handle migration from old inline object format.
	// If it's a JSON object (old format), silently treat it as "default".
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err == nil {
		if themeRaw, ok := raw["theme"]; ok && len(themeRaw) > 0 {
			trimmed := strings.TrimSpace(string(themeRaw))
			if strings.HasPrefix(trimmed, "{") {
				// Old inline theme object — replace with "default" for this load.
				raw["theme"] = json.RawMessage(`"default"`)
				data, _ = json.Marshal(raw)
			}
		}
	}

	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("config: parse JSON: %w", err)
	}

	// Ensure themes directory exists.
	themesDir := filepath.Join(configDir, "themes")
	_ = os.MkdirAll(themesDir, 0o755)

	// Resolve and apply theme.
	tf, err := ResolveTheme(cfg.ThemeName, configDir)
	if err != nil {
		return nil, err
	}
	ApplyTheme(&cfg, tf)

	return &cfg, nil
}

// Save writes the configuration to config.json in the config's data directory.
func Save(cfg *Config) error {
	dir, err := cfg.ResolveDataDir()
	if err != nil {
		return fmt.Errorf("config: resolve data dir: %w", err)
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("config: create dir: %w", err)
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("config: marshal: %w", err)
	}
	path := filepath.Join(dir, "config.json")
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("config: write: %w", err)
	}
	return nil
}

// ListThemes returns the names of all available themes (built-in + user files).
func ListThemes(configDir string) []string {
	seen := make(map[string]bool)
	var names []string
	for _, n := range BuiltinThemeNames() {
		names = append(names, n)
		seen[n] = true
	}
	themesDir := filepath.Join(configDir, "themes")
	entries, err := os.ReadDir(themesDir)
	if err != nil {
		return names
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if strings.HasSuffix(name, ".json") {
			base := strings.TrimSuffix(name, ".json")
			if !seen[base] {
				names = append(names, base)
				seen[base] = true
			}
		}
	}
	return names
}
