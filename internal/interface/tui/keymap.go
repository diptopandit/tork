package tui

import (
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/diptopandit/tork/internal/infrastructure/config"
)

// KeyMap holds all key bindings used throughout the TUI.
type KeyMap struct {
	Up          key.Binding
	Down        key.Binding
	Left        key.Binding
	Right       key.Binding
	Select      key.Binding
	Quit        key.Binding
	Help        key.Binding
	New         key.Binding
	Edit        key.Binding
	Delete      key.Binding
	Search      key.Binding
	Done        key.Binding
	Status      key.Binding
	Tab         key.Binding
	ShiftTab    key.Binding
	ListSwitch  key.Binding
	Comment     key.Binding
	ThemePicker key.Binding
	Sort        key.Binding
}

// ShortHelp returns bindings shown in the compact help bar.
func (k KeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Up, k.Down, k.Left, k.Right, k.Tab, k.New, k.Comment, k.Done, k.Status, k.Help, k.Quit}
}

// FullHelp returns all bindings for the full help view.
func (k KeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Up, k.Down, k.Left, k.Right},
		{k.New, k.Edit, k.Done, k.Status, k.Delete, k.Comment},
		{k.Search, k.Tab, k.ShiftTab, k.ListSwitch, k.ThemePicker, k.Sort, k.Help, k.Quit},
	}
}

// NewKeyMap constructs a KeyMap from a config.KeyMap.
func NewKeyMap(c config.KeyMap) KeyMap {
	tpKey := c.ThemePicker
	if tpKey == "" {
		tpKey = "T"
	}
	lsKey := c.ListSwitch
	if lsKey == "" {
		lsKey = "L"
	}
	sortKey := c.Sort
	if sortKey == "" {
		sortKey = "S"
	}
	return KeyMap{
		Up:          key.NewBinding(key.WithKeys(c.Up, "up"), key.WithHelp(c.Up+"/↑", "up")),
		Down:        key.NewBinding(key.WithKeys(c.Down, "down"), key.WithHelp(c.Down+"/↓", "down")),
		Left:        key.NewBinding(key.WithKeys(c.Left, "left"), key.WithHelp(c.Left+"/←", "focus left")),
		Right:       key.NewBinding(key.WithKeys(c.Right, "right"), key.WithHelp(c.Right+"/→", "focus right")),
		Select:      key.NewBinding(key.WithKeys(c.Select), key.WithHelp(c.Select, "select")),
		Quit:        key.NewBinding(key.WithKeys(c.Quit, "ctrl+c"), key.WithHelp(c.Quit+"/ctrl+c", "quit")),
		Help:        key.NewBinding(key.WithKeys(c.Help), key.WithHelp(c.Help, "help")),
		New:         key.NewBinding(key.WithKeys(c.New), key.WithHelp(c.New, "new task")),
		Edit:        key.NewBinding(key.WithKeys(c.Edit), key.WithHelp(c.Edit, "edit")),
		Delete:      key.NewBinding(key.WithKeys(c.Delete), key.WithHelp(c.Delete, "delete")),
		Search:      key.NewBinding(key.WithKeys(c.Search), key.WithHelp(c.Search, "search/filter")),
		Done:        key.NewBinding(key.WithKeys(c.Done), key.WithHelp(c.Done, "mark done")),
		Status:      key.NewBinding(key.WithKeys(c.Status), key.WithHelp(c.Status, "cycle status")),
		Tab:         key.NewBinding(key.WithKeys("tab"), key.WithHelp("tab", "next tab")),
		ShiftTab:    key.NewBinding(key.WithKeys("shift+tab"), key.WithHelp("shift+tab", "prev tab")),
		ListSwitch:  key.NewBinding(key.WithKeys(lsKey), key.WithHelp(lsKey, "switch list")),
		Comment:     key.NewBinding(key.WithKeys("u"), key.WithHelp("u", "add update")),
		ThemePicker: key.NewBinding(key.WithKeys(tpKey), key.WithHelp(tpKey, "theme picker")),
		Sort:        key.NewBinding(key.WithKeys(sortKey), key.WithHelp(sortKey, "sort tasks")),
	}
}

// isKey checks whether a KeyMsg matches a binding.
func isKey(msg tea.KeyMsg, b key.Binding) bool {
	return key.Matches(msg, b)
}
