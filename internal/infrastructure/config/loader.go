package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// defaults returns a Config pre-populated with sensible values.
func defaults() Config {
	return Config{
		Keybindings: KeyMap{
			Up:     "k",
			Down:   "j",
			Left:   "h",
			Right:  "l",
			Select: "enter",
			Quit:   "q",
			Help:   "?",
			New:    "n",
			Edit:   "e",
			Delete: "d",
			Search: "/",
			Done:   "x",
			Status: "s",
		},
		Display: DisplayConfig{
			Columns:        []string{"title", "status", "priority", "due_date"},
			DateFormat:     "02-01-2006",
			ShowTimestamps: true,
		},
		Theme: ThemeConfig{
			Primary:    "#7C3AED",
			Secondary:  "#6B7280",
			Active:     "#7C3AED",
			Inactive:   "#374151",
			Success:    "#10B981",
			Warning:    "#F59E0B",
			Danger:     "#EF4444",
			Text:       "#E5E7EB",
			TextMuted:  "#9CA3AF",
			TextBright: "#FFFFFF",
			Accent:     "#60A5FA",
		},
	}
}

// Load reads ~/.tork/config.json and merges it over the defaults.
// If the file does not exist the defaults are returned without error.
func Load() (*Config, error) {
	cfg := defaults()

	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("config: resolve home: %w", err)
	}
	path := filepath.Join(home, ".tork", "config.json")

	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return &cfg, nil
	}
	if err != nil {
		return nil, fmt.Errorf("config: read file: %w", err)
	}

	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("config: parse JSON: %w", err)
	}
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
