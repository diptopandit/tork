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
	DataDir     string                   `json:"data_dir"` // directory for db, logs, config (default: ~/.tork)
	Keybindings KeyMap                   `json:"keybindings"`
	Display     DisplayConfig            `json:"display"`
	ThemeName   string                   `json:"theme"`                 // name of the active theme
	Theme       ThemeConfig              `json:"-"`                     // resolved at load time, not serialized
	Statuses    []StatusDef              `json:"statuses"`              // ordered task statuses
	Priorities  []PriorityDef            `json:"priorities"`            // ordered priority levels
	DefaultList string                   `json:"default_list"`          // list name or ID to open on startup
	LastList    string                   `json:"last_list"`             // persisted by TUI on exit
	Remotes     map[string]*RemoteConfig `json:"remotes,omitempty"`     // named remote database backends
	LastRemote  string                   `json:"last_remote,omitempty"` // "local" or remote name; empty = not chosen
	// Deprecated: use Remotes instead. Kept for backward-compatible migration.
	RemoteDB *RemoteDBConfig `json:"remote_db,omitempty"`
	UserID   string          `json:"user_id,omitempty"`
	Username string          `json:"username,omitempty"`
}

// RemoteConfig holds connection details for one remote backend.
// Password is never stored — it's prompted at connection time.
// Username doubles as the unique user identity for multi-user scoping.
type RemoteConfig struct {
	Driver   string `json:"driver"`             // "mysql"
	Host     string `json:"host"`               // e.g. "db.example.com"
	Port     int    `json:"port,omitempty"`     // default: 3306
	Database string `json:"database,omitempty"` // default: "tork"
	Username string `json:"username"`           // MySQL login user — unique per user on a shared server
}

// EffectivePort returns the configured port or 3306 as default.
func (rc *RemoteConfig) EffectivePort() int {
	if rc.Port > 0 {
		return rc.Port
	}
	return 3306
}

// EffectiveDatabase returns the configured database name or "tork" as default.
func (rc *RemoteConfig) EffectiveDatabase() string {
	if rc.Database != "" {
		return rc.Database
	}
	return "tork"
}

// BuildDSN constructs a MySQL DSN from the structured fields and the given password.
func (rc *RemoteConfig) BuildDSN(password string) string {
	return fmt.Sprintf("%s:%s@tcp(%s:%d)/%s",
		rc.Username, password, rc.Host, rc.EffectivePort(), rc.EffectiveDatabase())
}

// Label returns a human-readable label for this remote.
func (rc *RemoteConfig) Label() string {
	return fmt.Sprintf("%s@%s:%d/%s", rc.Username, rc.Host, rc.EffectivePort(), rc.EffectiveDatabase())
}

// RemoteDBConfig is the deprecated single-remote config. Migrated to Remotes on load.
type RemoteDBConfig struct {
	Driver string `json:"driver"` // "mysql"
	DSN    string `json:"dsn"`    // e.g. "user:pass@tcp(host:3306)/tork"
}

// ActiveRemote returns the RemoteConfig for the given name, or nil if "local" or not found.
func (c *Config) ActiveRemote(name string) *RemoteConfig {
	if name == "" || name == "local" {
		return nil
	}
	if c.Remotes == nil {
		return nil
	}
	return c.Remotes[name]
}

// RemoteNames returns the sorted list of configured remote names.
func (c *Config) RemoteNames() []string {
	names := make([]string, 0, len(c.Remotes))
	for name := range c.Remotes {
		names = append(names, name)
	}
	// Stable order.
	for i := 1; i < len(names); i++ {
		for j := i; j > 0 && names[j] < names[j-1]; j-- {
			names[j], names[j-1] = names[j-1], names[j]
		}
	}
	return names
}

// ResolveRemote determines which remote to use based on flags and persisted state.
// Returns the remote name ("local" for SQLite), and whether a picker should be shown
// (true when there are remotes but no preference yet).
func (c *Config) ResolveRemote(flagRemote string, flagLocal bool) (name string, needsPicker bool) {
	// Explicit flags win.
	if flagLocal {
		return "local", false
	}
	if flagRemote != "" {
		return flagRemote, false
	}
	// Persisted last choice.
	if c.LastRemote != "" {
		// Validate it still exists.
		if c.LastRemote == "local" {
			return "local", false
		}
		if c.ActiveRemote(c.LastRemote) != nil {
			return c.LastRemote, false
		}
		// Stale reference — fall through.
	}
	// No preference. If remotes exist, need picker.
	if len(c.Remotes) > 0 {
		return "", true
	}
	return "local", false
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
