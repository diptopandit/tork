package config

import (
	"fmt"
	"os"
	"path/filepath"
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
	TabOrder       []string `json:"tab_order"`       // ordered tab names matching status names + "all"
	DefaultTab     string   `json:"default_tab"`     // tab selected on startup
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
	Statuses    []StatusDef   `json:"statuses"`     // ordered task statuses
	Priorities  []PriorityDef `json:"priorities"`   // ordered priority levels
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
