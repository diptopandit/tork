package tui

import (
	"github.com/diptopandit/tork/internal/domain"
	"github.com/diptopandit/tork/internal/infrastructure/config"
)

// Pane identifies which pane has keyboard focus.
type Pane int

const (
	PaneList   Pane = iota // left: task list table
	PaneDetail             // right: task detail viewport
)

// DetailFocus identifies which section of the right pane has scroll focus.
type DetailFocus int

const (
	FocusDetails DetailFocus = iota // task detail section (top)
	FocusUpdates                    // task updates section (middle)
)

// TabLabel returns a display label for a tab name using the config statuses.
// "all" is always labelled "All".
func TabLabel(defs []config.StatusDef, s string) string {
	if s == "all" {
		return "All"
	}
	return config.StatusLabel(defs, s)
}

// TabStatus returns the domain.Status filter for a tab string.
// Returns nil for "all" (no filter).
func TabStatus(s string) []domain.Status {
	if s == "all" {
		return nil
	}
	return []domain.Status{domain.Status(s)}
}

// ValidTab reports whether s is "all" or a defined status name.
func ValidTab(defs []config.StatusDef, s string) bool {
	if s == "all" {
		return true
	}
	return config.ValidStatus(defs, s)
}

// Overlay identifies a modal that floats above the panes.
type Overlay int

const (
	OverlayNone           Overlay = iota
	OverlayEdit                   // create / edit task form
	OverlayFilter                 // filter panel
	OverlayHelp                   // keybinding help
	OverlayListSwitch             // list switcher
	OverlayThemePicker            // theme picker with live preview
	OverlaySort                   // sort picker
	OverlayRemotePicker           // remote database picker
	OverlayPasswordPrompt         // password input for remote connection
)

// AppState holds all mutable UI state in one place.
type AppState struct {
	ActivePane    Pane
	DetailFocus   DetailFocus // which right-pane section has scroll focus
	ActiveTab     int         // index into Model.tabOrder
	ActiveOverlay Overlay
	ActiveListID  string // currently active task list filter
	SelectedTask  *domain.Task
	Tasks         []domain.Task
	TaskLists     []domain.TaskList
	Filter        domain.TaskFilter
	StatusMsg     string
}
