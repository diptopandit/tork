package tui

import "github.com/diptopandit/tork/internal/domain"

// Pane identifies which pane has keyboard focus.
type Pane int

const (
	PaneList   Pane = iota // left: task list table
	PaneDetail             // right: task detail viewport
)

// Tab identifies which filter tab is active in the list pane.
type Tab int

const (
	TabAll        Tab = iota // all tasks
	TabTodo                  // status = todo
	TabInProgress            // status = in_progress
	TabDone                  // status = done
)

func (t Tab) String() string {
	switch t {
	case TabAll:
		return "All"
	case TabTodo:
		return "Todo"
	case TabInProgress:
		return "In Progress"
	case TabDone:
		return "Done"
	}
	return ""
}

// TabCount is the number of defined tabs.
const TabCount = 4

// Overlay identifies a modal that floats above the panes.
type Overlay int

const (
	OverlayNone       Overlay = iota
	OverlayEdit               // create / edit task form
	OverlayFilter             // filter panel
	OverlayHelp               // keybinding help
	OverlayListSwitch         // list switcher
)

// AppState holds all mutable UI state in one place.
type AppState struct {
	ActivePane    Pane
	ActiveTab     Tab
	ActiveOverlay Overlay
	ActiveListID  string // currently active task list filter
	SelectedTask  *domain.Task
	Tasks         []domain.Task
	TaskLists     []domain.TaskList
	Filter        domain.TaskFilter
	StatusMsg     string
}
