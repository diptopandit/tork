package config

import (
	"os"
	"path/filepath"
)

// KeyMap holds vim-style key bindings used by the TUI.
type KeyMap struct {
	Up     string `json:"up"`
	Down   string `json:"down"`
	Left   string `json:"left"`
	Right  string `json:"right"`
	Select string `json:"select"`
	Quit   string `json:"quit"`
	Help   string `json:"help"`
	New    string `json:"new"`
	Edit   string `json:"edit"`
	Delete string `json:"delete"`
	Search string `json:"search"`
	Done   string `json:"done"`
	Status string `json:"status"`
}

// DisplayConfig controls how the task list table is rendered.
type DisplayConfig struct {
	Columns        []string `json:"columns"`
	DateFormat     string   `json:"date_format"`
	ShowTimestamps bool     `json:"show_timestamps"` // show created/updated in detail view
}

// ThemeConfig holds lipgloss color references.
type ThemeConfig struct {
	Primary    string `json:"primary"`
	Secondary  string `json:"secondary"`
	Active     string `json:"active"`
	Inactive   string `json:"inactive"`
	Success    string `json:"success"`
	Warning    string `json:"warning"`
	Danger     string `json:"danger"`
	Text       string `json:"text"`        // primary text color (light)
	TextMuted  string `json:"text_muted"`  // muted/placeholder text
	TextBright string `json:"text_bright"` // high-contrast bright text
	Accent     string `json:"accent"`      // accent for timestamps, highlights
}

// Config is the top-level application configuration.
type Config struct {
	DataDir     string        `json:"data_dir"` // directory for db, logs, config (default: ~/.tork)
	Keybindings KeyMap        `json:"keybindings"`
	Display     DisplayConfig `json:"display"`
	Theme       ThemeConfig   `json:"theme"`
	DefaultList string        `json:"default_list"` // list name or ID to open on startup
	LastList    string        `json:"last_list"`    // persisted by TUI on exit
}

// ResolveDataDir returns the effective data directory, falling back to ~/.tork.
func (c *Config) ResolveDataDir() (string, error) {
	if c.DataDir != "" {
		return c.DataDir, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".tork"), nil
}
