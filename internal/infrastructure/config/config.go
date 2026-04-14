package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/charmbracelet/lipgloss"
)

// StatusDef defines a task status with its internal name and display label.
type StatusDef struct {
	Name  string `json:"name"`
	Label string `json:"label"`
}

// PriorityDef defines a priority level with a display name, numeric value, and label.
type PriorityDef struct {
	Name  string `json:"name"`
	Value int    `json:"value"`
	Label string `json:"label"`
}

// StatusLabel returns the display label for a status name, falling back to the name itself.
func StatusLabel(defs []StatusDef, name string) string {
	for _, d := range defs {
		if d.Name == name {
			if d.Label != "" {
				return d.Label
			}
			return d.Name
		}
	}
	return name
}

// PriorityLabel returns the display label for a priority value.
func PriorityLabel(defs []PriorityDef, value int) string {
	for _, d := range defs {
		if d.Value == value {
			if d.Label != "" {
				return d.Label
			}
			return d.Name
		}
	}
	return fmt.Sprintf("%d", value)
}

// PriorityByName resolves a priority name or numeric string to a value.
func PriorityByName(defs []PriorityDef, s string) (int, bool) {
	for i, d := range defs {
		if d.Name == s || fmt.Sprintf("%d", i+1) == s || fmt.Sprintf("%d", d.Value) == s {
			return d.Value, true
		}
	}
	return 0, false
}

// StatusNames returns the ordered list of status names.
func StatusNames(defs []StatusDef) []string {
	names := make([]string, len(defs))
	for i, d := range defs {
		names[i] = d.Name
	}
	return names
}

// ValidStatus reports whether name is a defined status.
func ValidStatus(defs []StatusDef, name string) bool {
	for _, d := range defs {
		if d.Name == name {
			return true
		}
	}
	return false
}

// DefaultStatus returns the first status name, or "todo" as fallback.
func DefaultStatus(defs []StatusDef) string {
	if len(defs) > 0 {
		return defs[0].Name
	}
	return "todo"
}

// DefaultPriority returns the value of the priority named "medium",
// or the middle priority, or 1.
func DefaultPriority(defs []PriorityDef) int {
	for _, d := range defs {
		if d.Name == "medium" {
			return d.Value
		}
	}
	if len(defs) > 0 {
		return defs[len(defs)/2].Value
	}
	return 1
}

// KeyMap holds vim-style key bindings used by the TUI.
type KeyMap struct {
	Up          string `json:"up"`
	Down        string `json:"down"`
	Left        string `json:"left"`
	Right       string `json:"right"`
	Select      string `json:"select"`
	Quit        string `json:"quit"`
	Help        string `json:"help"`
	New         string `json:"new"`
	Edit        string `json:"edit"`
	Delete      string `json:"delete"`
	Search      string `json:"search"`
	Done        string `json:"done"`
	Status      string `json:"status"`
	ThemePicker string `json:"theme_picker"`
	ListSwitch  string `json:"list_switch"`
	Sort        string `json:"sort"`
}

// DisplayConfig controls how the task list table is rendered.
type DisplayConfig struct {
	Columns        []string `json:"columns"`
	DateFormat     string   `json:"date_format"`
	ShowTimestamps bool     `json:"show_timestamps"` // show created/updated in detail view
	TabOrder       []string `json:"tab_order"`       // ordered tab names matching status names + "all"
	DefaultTab     string   `json:"default_tab"`     // tab selected on startup
}

// ThemeConfig holds resolved lipgloss color references used at runtime.
type ThemeConfig struct {
	Primary        string   `json:"primary"`
	Secondary      string   `json:"secondary"`
	Active         string   `json:"active"`
	Inactive       string   `json:"inactive"`
	Success        string   `json:"success"`
	Warning        string   `json:"warning"`
	Danger         string   `json:"danger"`
	Text           string   `json:"text"`            // primary text color (light)
	TextMuted      string   `json:"text_muted"`      // muted/placeholder text
	TextBright     string   `json:"text_bright"`     // high-contrast bright text
	Accent         string   `json:"accent"`          // accent for timestamps, highlights
	Border         string   `json:"border"`          // "rounded", "normal", "double", "hidden"
	StatusColors   []string `json:"status_colors"`   // positional colors for statuses
	PriorityColors []string `json:"priority_colors"` // positional colors for priorities
}

// StatusColor returns the color for a status at the given index, falling back to Primary.
func (t *ThemeConfig) StatusColor(index int) string {
	if index >= 0 && index < len(t.StatusColors) {
		return t.StatusColors[index]
	}
	return t.Primary
}

// PriorityColor returns the color for a priority at the given index, falling back to Primary.
func (t *ThemeConfig) PriorityColor(index int) string {
	if index >= 0 && index < len(t.PriorityColors) {
		return t.PriorityColors[index]
	}
	return t.Primary
}

// BorderStyle returns the lipgloss border matching the configured border name.
func (t *ThemeConfig) BorderStyle() lipgloss.Border {
	switch t.Border {
	case "normal":
		return lipgloss.NormalBorder()
	case "double":
		return lipgloss.DoubleBorder()
	case "hidden":
		return lipgloss.HiddenBorder()
	default:
		return lipgloss.RoundedBorder()
	}
}

// ThemeFile represents a complete theme definition stored as a JSON file
// or embedded as a built-in preset.
type ThemeFile struct {
	Name           string      `json:"name"`
	Description    string      `json:"description"`
	Author         string      `json:"author"`
	Colors         ThemeConfig `json:"colors"` // only the 11 semantic color fields are used
	Border         string      `json:"border"`
	StatusColors   []string    `json:"status_colors"`
	PriorityColors []string    `json:"priority_colors"`
}

// Config is the top-level application configuration.
type Config struct {
	DataDir     string          `json:"data_dir"` // directory for db, logs, config (default: ~/.tork)
	Keybindings KeyMap          `json:"keybindings"`
	Display     DisplayConfig   `json:"display"`
	ThemeName   string          `json:"theme"`               // name of the active theme
	Theme       ThemeConfig     `json:"-"`                   // resolved at load time, not serialized
	Statuses    []StatusDef     `json:"statuses"`            // ordered task statuses
	Priorities  []PriorityDef   `json:"priorities"`          // ordered priority levels
	DefaultList string          `json:"default_list"`        // list name or ID to open on startup
	LastList    string          `json:"last_list"`           // persisted by TUI on exit
	RemoteDB    *RemoteDBConfig `json:"remote_db,omitempty"` // optional remote MySQL backend
	UserID      string          `json:"user_id,omitempty"`   // auto-generated UUID for remote multi-user
	Username    string          `json:"username,omitempty"`  // display name for remote multi-user
}

// RemoteDBConfig holds connection details for an optional remote SQL backend.
// When configured, tork uses this instead of the local SQLite database.
type RemoteDBConfig struct {
	Driver string `json:"driver"` // "mysql"
	DSN    string `json:"dsn"`    // e.g. "user:pass@tcp(host:3306)/tork"
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
