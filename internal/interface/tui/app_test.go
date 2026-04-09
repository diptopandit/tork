package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/diptopandit/tork/internal/application"
	"github.com/diptopandit/tork/internal/domain"
	"github.com/diptopandit/tork/internal/infrastructure/config"
)

// ---- stubs ------------------------------------------------------------------

type stubTaskRepo struct{}

func (s *stubTaskRepo) Create(*domain.Task) error                     { return nil }
func (s *stubTaskRepo) Update(*domain.Task) error                     { return nil }
func (s *stubTaskRepo) Delete(string) error                           { return nil }
func (s *stubTaskRepo) GetByID(string) (*domain.Task, error)          { return nil, nil }
func (s *stubTaskRepo) GetByNumID(int) (*domain.Task, error)          { return nil, nil }
func (s *stubTaskRepo) List(domain.TaskFilter) ([]domain.Task, error) { return nil, nil }
func (s *stubTaskRepo) NextNumID() (int, error)                       { return 1, nil }

type stubUpdateRepo struct{}

func (s *stubUpdateRepo) AddUpdate(*domain.Update) error               { return nil }
func (s *stubUpdateRepo) ListByTaskID(string) ([]domain.Update, error) { return nil, nil }

type stubSearch struct{}

func (s *stubSearch) Index(*domain.Task) error        { return nil }
func (s *stubSearch) Delete(string) error             { return nil }
func (s *stubSearch) Search(string) ([]string, error) { return nil, nil }

type stubListRepo struct{}

func (s *stubListRepo) Create(*domain.TaskList) error            { return nil }
func (s *stubListRepo) Update(*domain.TaskList) error            { return nil }
func (s *stubListRepo) GetByID(string) (*domain.TaskList, error) { return nil, nil }
func (s *stubListRepo) GetAll() ([]domain.TaskList, error)       { return nil, nil }
func (s *stubListRepo) Delete(string) error                      { return nil }

func newTestModel() Model {
	cfg, _ := config.Load()
	taskSvc := application.NewTaskService(&stubTaskRepo{}, &stubUpdateRepo{}, &stubSearch{})
	listSvc := application.NewListService(&stubListRepo{})
	return NewModel(taskSvc, listSvc, cfg)
}

// ---- tests ------------------------------------------------------------------

func TestInitialState(t *testing.T) {
	m := newTestModel()
	if m.state.ActivePane != PaneList {
		t.Errorf("initial pane = %d, want PaneList", m.state.ActivePane)
	}
	if m.state.ActiveTab != TabAll {
		t.Errorf("initial tab = %d, want TabAll", m.state.ActiveTab)
	}
}

func TestQuitKeyTransition(t *testing.T) {
	m := newTestModel()
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")}
	newM, cmd := m.Update(msg)
	_ = newM
	if cmd == nil {
		t.Error("expected quit cmd from q key")
	}
}

func TestNewTaskKeyNavigation(t *testing.T) {
	m := newTestModel()
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("n")}
	newModel, _ := m.Update(msg)
	nm := newModel.(Model)
	if nm.state.ActiveOverlay != OverlayEdit {
		t.Errorf("active overlay = %d, want OverlayEdit", nm.state.ActiveOverlay)
	}
}

func TestFilterKeyNavigation(t *testing.T) {
	m := newTestModel()
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("/")}
	newModel, _ := m.Update(msg)
	nm := newModel.(Model)
	if nm.state.ActiveOverlay != OverlayFilter {
		t.Errorf("active overlay = %d, want OverlayFilter", nm.state.ActiveOverlay)
	}
}

func TestWindowResize(t *testing.T) {
	m := newTestModel()
	msg := tea.WindowSizeMsg{Width: 120, Height: 40}
	newModel, _ := m.Update(msg)
	nm := newModel.(Model)
	if nm.width != 120 || nm.height != 40 {
		t.Errorf("size = %dx%d, want 120x40", nm.width, nm.height)
	}
}

func TestTabCycleForward(t *testing.T) {
	m := newTestModel()
	// Simulate a window size so recalcLayout doesn't short-circuit.
	m.width = 120
	m.height = 40
	m.recalcLayout()

	msg := tea.KeyMsg{Type: tea.KeyTab}
	newModel, _ := m.Update(msg)
	nm := newModel.(Model)
	if nm.state.ActiveTab != TabTodo {
		t.Errorf("tab = %d, want TabTodo after first Tab press", nm.state.ActiveTab)
	}
}

func TestTabCycleBackward(t *testing.T) {
	m := newTestModel()
	m.width = 120
	m.height = 40
	m.recalcLayout()

	msg := tea.KeyMsg{Type: tea.KeyShiftTab}
	newModel, _ := m.Update(msg)
	nm := newModel.(Model)
	if nm.state.ActiveTab != TabDone {
		t.Errorf("tab = %d, want TabDone after Shift+Tab from TabAll", nm.state.ActiveTab)
	}
}

func TestListSwitchOverlayOpen(t *testing.T) {
	m := newTestModel()
	m.width = 120
	m.height = 40
	m.recalcLayout()

	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("L")}
	newModel, _ := m.Update(msg)
	nm := newModel.(Model)
	if nm.state.ActiveOverlay != OverlayListSwitch {
		t.Errorf("overlay = %d, want OverlayListSwitch", nm.state.ActiveOverlay)
	}
}

func TestListSwitchCreateMode(t *testing.T) {
	m := newTestModel()
	m.width = 120
	m.height = 40
	m.recalcLayout()
	m.state.ActiveOverlay = OverlayListSwitch

	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("n")}
	newModel, _ := m.Update(msg)
	nm := newModel.(Model)
	if nm.listSwitchMode != 1 {
		t.Errorf("listSwitchMode = %d, want 1 (create)", nm.listSwitchMode)
	}
}

func TestListSwitchDeleteConfirmation(t *testing.T) {
	m := newTestModel()
	m.width = 120
	m.height = 40
	m.recalcLayout()
	m.state.ActiveOverlay = OverlayListSwitch
	m.state.TaskLists = []domain.TaskList{{ID: "l1", Name: "Test"}}

	// Press d to start delete
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("d")}
	newModel, _ := m.Update(msg)
	nm := newModel.(Model)
	if nm.listSwitchMode != 3 {
		t.Errorf("listSwitchMode = %d, want 3 (first confirm)", nm.listSwitchMode)
	}

	// Press y for first confirmation
	msg2 := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("y")}
	newModel2, _ := nm.Update(msg2)
	nm2 := newModel2.(Model)
	if nm2.listSwitchMode != 4 {
		t.Errorf("listSwitchMode = %d, want 4 (second confirm)", nm2.listSwitchMode)
	}

	// Press something other than y to cancel
	msg3 := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("n")}
	newModel3, _ := nm2.Update(msg3)
	nm3 := newModel3.(Model)
	if nm3.listSwitchMode != 0 {
		t.Errorf("listSwitchMode = %d, want 0 (cancelled)", nm3.listSwitchMode)
	}
}

func TestStatusCycleKey(t *testing.T) {
	m := newTestModel()
	m.width = 120
	m.height = 40
	m.recalcLayout()

	// Status key 's' in list pane without a selected task should be a no-op
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("s")}
	newModel, cmd := m.Update(msg)
	nm := newModel.(Model)
	if cmd != nil {
		t.Error("expected nil cmd when no task selected for status cycle")
	}
	_ = nm
}

func TestPaneSwitchKeys(t *testing.T) {
	m := newTestModel()
	m.width = 120
	m.height = 40
	m.recalcLayout()

	// Press 'l' to go right
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("l")}
	newModel, _ := m.Update(msg)
	nm := newModel.(Model)
	if nm.state.ActivePane != PaneDetail {
		t.Errorf("pane = %d, want PaneDetail after 'l'", nm.state.ActivePane)
	}

	// Press 'h' to go left
	msg2 := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("h")}
	newModel2, _ := nm.Update(msg2)
	nm2 := newModel2.(Model)
	if nm2.state.ActivePane != PaneList {
		t.Errorf("pane = %d, want PaneList after 'h'", nm2.state.ActivePane)
	}
}

func TestHelpOverlayToggle(t *testing.T) {
	m := newTestModel()
	m.width = 120
	m.height = 40
	m.recalcLayout()

	// Open help
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("?")}
	newModel, _ := m.Update(msg)
	nm := newModel.(Model)
	if nm.state.ActiveOverlay != OverlayHelp {
		t.Errorf("overlay = %d, want OverlayHelp", nm.state.ActiveOverlay)
	}

	// Close help
	msg2 := tea.KeyMsg{Type: tea.KeyEsc}
	newModel2, _ := nm.Update(msg2)
	nm2 := newModel2.(Model)
	if nm2.state.ActiveOverlay != OverlayNone {
		t.Errorf("overlay = %d, want OverlayNone after Esc", nm2.state.ActiveOverlay)
	}
}
