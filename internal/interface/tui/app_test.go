package tui

import (
	"fmt"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/diptopandit/tork/internal/application"
	"github.com/diptopandit/tork/internal/domain"
	"github.com/diptopandit/tork/internal/infrastructure/bootstrap"
	"github.com/diptopandit/tork/internal/infrastructure/config"
	"github.com/diptopandit/tork/internal/interface/tui/views"
)

// ---- stubs ------------------------------------------------------------------

type stubTaskRepo struct {
	tasks map[string]*domain.Task
	numID int
}

func newStubTaskRepoWithData() *stubTaskRepo {
	now := time.Now()
	return &stubTaskRepo{
		tasks: map[string]*domain.Task{
			"t1": {ID: "t1", NumID: 1, ListID: "l1", Title: "Task One", Status: domain.StatusTodo, Priority: domain.PriorityMedium, Tags: []string{"go"}, CreatedAt: now, UpdatedAt: now},
			"t2": {ID: "t2", NumID: 2, ListID: "l1", Title: "Task Two", Status: domain.StatusInProgress, Priority: domain.PriorityHigh, CreatedAt: now, UpdatedAt: now},
		},
		numID: 2,
	}
}

func (s *stubTaskRepo) Create(t *domain.Task) error { s.tasks[t.ID] = t; return nil }
func (s *stubTaskRepo) Update(t *domain.Task) error { s.tasks[t.ID] = t; return nil }
func (s *stubTaskRepo) Delete(id string) error      { delete(s.tasks, id); return nil }
func (s *stubTaskRepo) GetByID(id string) (*domain.Task, error) {
	if t, ok := s.tasks[id]; ok {
		c := *t
		return &c, nil
	}
	return nil, domain.ErrNotFound(id)
}
func (s *stubTaskRepo) GetByNumID(numID int) (*domain.Task, error) {
	for _, t := range s.tasks {
		if t.NumID == numID {
			c := *t
			return &c, nil
		}
	}
	return nil, domain.ErrNotFound("num")
}
func (s *stubTaskRepo) List(domain.TaskFilter) ([]domain.Task, error) {
	out := make([]domain.Task, 0)
	for _, t := range s.tasks {
		out = append(out, *t)
	}
	return out, nil
}
func (s *stubTaskRepo) NextNumID() (int, error) { s.numID++; return s.numID, nil }

type stubUpdateRepo struct{ updates map[string][]domain.Update }

func newStubUpdateRepo() *stubUpdateRepo {
	return &stubUpdateRepo{updates: map[string][]domain.Update{}}
}
func (s *stubUpdateRepo) AddUpdate(u *domain.Update) error {
	s.updates[u.TaskID] = append(s.updates[u.TaskID], *u)
	return nil
}
func (s *stubUpdateRepo) ListByTaskID(id string) ([]domain.Update, error) {
	return s.updates[id], nil
}

type stubSearch struct{}

func (s *stubSearch) Index(*domain.Task) error        { return nil }
func (s *stubSearch) Delete(string) error             { return nil }
func (s *stubSearch) Search(string) ([]string, error) { return nil, nil }

type stubListRepo struct {
	lists map[string]*domain.TaskList
}

func newStubListRepoWithData() *stubListRepo {
	return &stubListRepo{lists: map[string]*domain.TaskList{
		"l1": {ID: "l1", Name: "Default", CreatedAt: time.Now()},
	}}
}

func (s *stubListRepo) Create(l *domain.TaskList) error { s.lists[l.ID] = l; return nil }
func (s *stubListRepo) Update(l *domain.TaskList) error { s.lists[l.ID] = l; return nil }
func (s *stubListRepo) GetByID(id string) (*domain.TaskList, error) {
	if l, ok := s.lists[id]; ok {
		c := *l
		return &c, nil
	}
	return nil, domain.ErrNotFound(id)
}
func (s *stubListRepo) GetAll() ([]domain.TaskList, error) {
	out := make([]domain.TaskList, 0)
	for _, l := range s.lists {
		out = append(out, *l)
	}
	return out, nil
}
func (s *stubListRepo) Delete(id string) error { delete(s.lists, id); return nil }

func testCfg() *config.Config {
	return &config.Config{
		Keybindings: config.KeyMap{
			Up: "k", Down: "j", Left: "h", Right: "l",
			Select: "enter", Quit: "q", Help: "?",
			New: "n", Edit: "e", Delete: "d", Search: "/",
			Done: "x", Status: "s",
		},
		Display: config.DisplayConfig{
			Columns:        []string{"title", "status", "priority", "due_date"},
			DateFormat:     "02-01-2006",
			ShowTimestamps: true,
			TabOrder:       []string{"todo", "in_progress", "done", "all"},
			DefaultTab:     "todo",
		},
		ThemeName: "default",
		Theme: config.ThemeConfig{
			Primary: "#7C3AED", Secondary: "#6B7280",
			Active: "#7C3AED", Inactive: "#374151",
			Success: "#10B981", Warning: "#F59E0B", Danger: "#EF4444",
			Text: "#E5E7EB", TextMuted: "#9CA3AF",
			TextBright: "#FFFFFF", Accent: "#60A5FA",
			Border:         "rounded",
			StatusColors:   []string{"#F59E0B", "#60A5FA", "#10B981", "#EF4444"},
			PriorityColors: []string{"#6B7280", "#F59E0B", "#FB923C", "#EF4444"},
		},
		Statuses: []config.StatusDef{
			{Name: "todo", Label: "Todo"},
			{Name: "in_progress", Label: "In Progress"},
			{Name: "done", Label: "Done"},
			{Name: "cancelled", Label: "Cancelled"},
		},
		Priorities: []config.PriorityDef{
			{Name: "low", Value: 1, Label: "Low"},
			{Name: "medium", Value: 2, Label: "Medium"},
			{Name: "high", Value: 3, Label: "High"},
			{Name: "urgent", Value: 4, Label: "Urgent"},
		},
	}
}

func newTestModel() Model {
	cfg := testCfg()
	taskSvc := application.NewTaskService(newStubTaskRepoWithData(), newStubUpdateRepo(), &stubSearch{}, nil)
	listSvc := application.NewListService(newStubListRepoWithData(), nil)
	return NewModel(taskSvc, listSvc, cfg, nil, nil)
}

// sizedModel returns a Model with layout already computed.
func sizedModel() Model {
	m := newTestModel()
	m.width = 120
	m.height = 40
	m.recalcLayout()
	return m
}

// keyMsg builds a KeyMsg from a string.
func keyMsg(s string) tea.KeyMsg {
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(s)}
}

// specialKey builds a KeyMsg for special keys like Tab, Esc, Enter.
func specialKey(t tea.KeyType) tea.KeyMsg {
	return tea.KeyMsg{Type: t}
}

// ---- tests ------------------------------------------------------------------

func TestInitialState(t *testing.T) {
	m := newTestModel()
	if m.state.ActivePane != PaneList {
		t.Errorf("initial pane = %d, want PaneList", m.state.ActivePane)
	}
	if m.state.ActiveTab != 0 {
		t.Errorf("initial tab = %d, want 0 (todo)", m.state.ActiveTab)
	}
}

func TestQuitKeyTransition(t *testing.T) {
	m := newTestModel()
	newM, cmd := m.Update(keyMsg("q"))
	_ = newM
	if cmd == nil {
		t.Error("expected quit cmd from q key")
	}
}

func TestCtrlCQuit(t *testing.T) {
	m := sizedModel()
	newM, cmd := m.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
	nm := newM.(Model)
	if !nm.quitting {
		t.Error("expected quitting = true")
	}
	if cmd == nil {
		t.Error("expected quit cmd")
	}
}

func TestNewTaskKeyNavigation(t *testing.T) {
	m := newTestModel()
	newModel, _ := m.Update(keyMsg("n"))
	nm := newModel.(Model)
	if nm.state.ActiveOverlay != OverlayEdit {
		t.Errorf("active overlay = %d, want OverlayEdit", nm.state.ActiveOverlay)
	}
}

func TestFilterKeyNavigation(t *testing.T) {
	m := newTestModel()
	newModel, _ := m.Update(keyMsg("/"))
	nm := newModel.(Model)
	if nm.state.ActiveOverlay != OverlayFilter {
		t.Errorf("active overlay = %d, want OverlayFilter", nm.state.ActiveOverlay)
	}
}

func TestWindowResize(t *testing.T) {
	m := newTestModel()
	newModel, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	nm := newModel.(Model)
	if nm.width != 120 || nm.height != 40 {
		t.Errorf("size = %dx%d, want 120x40", nm.width, nm.height)
	}
}

func TestTabCycleForward(t *testing.T) {
	m := sizedModel()
	newModel, _ := m.Update(specialKey(tea.KeyTab))
	nm := newModel.(Model)
	if nm.state.ActiveTab != 1 {
		t.Errorf("tab = %d, want 1 (in_progress) after first Tab press", nm.state.ActiveTab)
	}
}

func TestTabCycleBackward(t *testing.T) {
	m := sizedModel()
	newModel, _ := m.Update(specialKey(tea.KeyShiftTab))
	nm := newModel.(Model)
	if nm.state.ActiveTab != 3 {
		t.Errorf("tab = %d, want 3 (all) after Shift+Tab from todo", nm.state.ActiveTab)
	}
}

func TestListSwitchOverlayOpen(t *testing.T) {
	m := sizedModel()
	newModel, _ := m.Update(keyMsg("L"))
	nm := newModel.(Model)
	if nm.state.ActiveOverlay != OverlayListSwitch {
		t.Errorf("overlay = %d, want OverlayListSwitch", nm.state.ActiveOverlay)
	}
}

func TestListSwitchCreateMode(t *testing.T) {
	m := sizedModel()
	m.state.ActiveOverlay = OverlayListSwitch

	newModel, _ := m.Update(keyMsg("n"))
	nm := newModel.(Model)
	if nm.listSwitchMode != 1 {
		t.Errorf("listSwitchMode = %d, want 1 (create)", nm.listSwitchMode)
	}
}

func TestListSwitchDeleteConfirmation(t *testing.T) {
	m := sizedModel()
	m.state.ActiveOverlay = OverlayListSwitch
	m.state.TaskLists = []domain.TaskList{{ID: "l1", Name: "Test"}}

	// Press d to start delete
	newModel, _ := m.Update(keyMsg("d"))
	nm := newModel.(Model)
	if nm.listSwitchMode != 3 {
		t.Errorf("listSwitchMode = %d, want 3 (first confirm)", nm.listSwitchMode)
	}

	// Press y for first confirmation
	newModel2, _ := nm.Update(keyMsg("y"))
	nm2 := newModel2.(Model)
	if nm2.listSwitchMode != 4 {
		t.Errorf("listSwitchMode = %d, want 4 (second confirm)", nm2.listSwitchMode)
	}

	// Press something other than y to cancel
	newModel3, _ := nm2.Update(keyMsg("n"))
	nm3 := newModel3.(Model)
	if nm3.listSwitchMode != 0 {
		t.Errorf("listSwitchMode = %d, want 0 (cancelled)", nm3.listSwitchMode)
	}
}

func TestStatusCycleKey(t *testing.T) {
	m := sizedModel()

	// Status key 's' in list pane without a selected task should be a no-op
	newModel, cmd := m.Update(keyMsg("s"))
	nm := newModel.(Model)
	if cmd != nil {
		t.Error("expected nil cmd when no task selected for status cycle")
	}
	_ = nm
}

func TestPaneSwitchKeys(t *testing.T) {
	m := sizedModel()

	// Press 'l' to go right
	newModel, _ := m.Update(keyMsg("l"))
	nm := newModel.(Model)
	if nm.state.ActivePane != PaneDetail {
		t.Errorf("pane = %d, want PaneDetail after 'l'", nm.state.ActivePane)
	}

	// Press 'h' to go left
	newModel2, _ := nm.Update(keyMsg("h"))
	nm2 := newModel2.(Model)
	if nm2.state.ActivePane != PaneList {
		t.Errorf("pane = %d, want PaneList after 'h'", nm2.state.ActivePane)
	}
}

func TestHelpOverlayToggle(t *testing.T) {
	m := sizedModel()

	// Open help
	newModel, _ := m.Update(keyMsg("?"))
	nm := newModel.(Model)
	if nm.state.ActiveOverlay != OverlayHelp {
		t.Errorf("overlay = %d, want OverlayHelp", nm.state.ActiveOverlay)
	}

	// Close help with Esc
	newModel2, _ := nm.Update(specialKey(tea.KeyEsc))
	nm2 := newModel2.(Model)
	if nm2.state.ActiveOverlay != OverlayNone {
		t.Errorf("overlay = %d, want OverlayNone after Esc", nm2.state.ActiveOverlay)
	}
}

// ---- async message tests ----------------------------------------------------

func TestTasksLoadedMsg_Success(t *testing.T) {
	m := sizedModel()
	now := time.Now()
	tasks := []domain.Task{
		{ID: "t1", NumID: 1, Title: "Task One", Status: domain.StatusTodo, CreatedAt: now, UpdatedAt: now},
		{ID: "t2", NumID: 2, Title: "Task Two", Status: domain.StatusDone, CreatedAt: now, UpdatedAt: now},
	}

	newModel, _ := m.Update(tasksLoadedMsg{tasks: tasks})
	nm := newModel.(Model)
	if len(nm.state.Tasks) != 2 {
		t.Errorf("tasks = %d, want 2", len(nm.state.Tasks))
	}
}

func TestTasksLoadedMsg_Error(t *testing.T) {
	m := sizedModel()
	newModel, _ := m.Update(tasksLoadedMsg{err: fmt.Errorf("db error")})
	nm := newModel.(Model)
	if !strings.Contains(nm.state.StatusMsg, "db error") {
		t.Errorf("status = %q, want to contain 'db error'", nm.state.StatusMsg)
	}
}

func TestListsLoadedMsg_Success(t *testing.T) {
	m := sizedModel()
	lists := []domain.TaskList{{ID: "l1", Name: "Work"}}

	newModel, _ := m.Update(listsLoadedMsg{lists: lists})
	nm := newModel.(Model)
	if len(nm.state.TaskLists) != 1 {
		t.Errorf("lists = %d, want 1", len(nm.state.TaskLists))
	}
}

func TestListsLoadedMsg_Empty_CreatesDefault(t *testing.T) {
	m := sizedModel()
	_, cmd := m.Update(listsLoadedMsg{lists: nil})
	if cmd == nil {
		t.Error("expected createDefaultList cmd when lists are empty")
	}
}

func TestListsLoadedMsg_Error(t *testing.T) {
	m := sizedModel()
	newModel, _ := m.Update(listsLoadedMsg{err: fmt.Errorf("fail")})
	nm := newModel.(Model)
	if !strings.Contains(nm.state.StatusMsg, "fail") {
		t.Errorf("status = %q", nm.state.StatusMsg)
	}
}

func TestDefaultListCreatedMsg_Success(t *testing.T) {
	m := sizedModel()
	list := &domain.TaskList{ID: "new-l", Name: "Default"}
	newModel, _ := m.Update(defaultListCreatedMsg{list: list})
	nm := newModel.(Model)
	if len(nm.state.TaskLists) != 1 || nm.state.TaskLists[0].Name != "Default" {
		t.Errorf("lists = %v", nm.state.TaskLists)
	}
}

func TestDefaultListCreatedMsg_Error(t *testing.T) {
	m := sizedModel()
	newModel, _ := m.Update(defaultListCreatedMsg{err: fmt.Errorf("fail")})
	nm := newModel.(Model)
	if !strings.Contains(nm.state.StatusMsg, "fail") {
		t.Errorf("status = %q", nm.state.StatusMsg)
	}
}

func TestTaskCreatedMsg_Success(t *testing.T) {
	m := sizedModel()
	m.state.ActiveOverlay = OverlayEdit

	task := &domain.Task{ID: "new-t", Title: "New"}
	newModel, cmd := m.Update(taskCreatedMsg{task: task})
	nm := newModel.(Model)
	if nm.state.ActiveOverlay != OverlayNone {
		t.Error("overlay should close after task creation")
	}
	if nm.state.StatusMsg != "Task created" {
		t.Errorf("status = %q", nm.state.StatusMsg)
	}
	if cmd == nil {
		t.Error("expected loadTasks cmd")
	}
}

func TestTaskCreatedMsg_Error(t *testing.T) {
	m := sizedModel()
	newModel, _ := m.Update(taskCreatedMsg{err: fmt.Errorf("dup")})
	nm := newModel.(Model)
	if !strings.Contains(nm.state.StatusMsg, "dup") {
		t.Errorf("status = %q", nm.state.StatusMsg)
	}
}

func TestTaskUpdatedMsg_Success(t *testing.T) {
	m := sizedModel()
	m.state.ActiveOverlay = OverlayEdit

	task := &domain.Task{ID: "t1", Title: "Updated"}
	newModel, cmd := m.Update(taskUpdatedMsg{task: task})
	nm := newModel.(Model)
	if nm.state.ActiveOverlay != OverlayNone {
		t.Error("overlay should close")
	}
	if nm.state.StatusMsg != "Task updated" {
		t.Errorf("status = %q", nm.state.StatusMsg)
	}
	if cmd == nil {
		t.Error("expected loadTasks cmd")
	}
}

func TestTaskUpdatedMsg_Error(t *testing.T) {
	m := sizedModel()
	newModel, _ := m.Update(taskUpdatedMsg{err: fmt.Errorf("fail")})
	nm := newModel.(Model)
	if !strings.Contains(nm.state.StatusMsg, "fail") {
		t.Errorf("status = %q", nm.state.StatusMsg)
	}
}

func TestTaskDeletedMsg_Success(t *testing.T) {
	m := sizedModel()
	newModel, cmd := m.Update(taskDeletedMsg{id: "t1"})
	nm := newModel.(Model)
	if nm.state.StatusMsg != "Task deleted" {
		t.Errorf("status = %q", nm.state.StatusMsg)
	}
	if cmd == nil {
		t.Error("expected loadTasks cmd")
	}
}

func TestTaskDeletedMsg_Error(t *testing.T) {
	m := sizedModel()
	newModel, _ := m.Update(taskDeletedMsg{err: fmt.Errorf("oops")})
	nm := newModel.(Model)
	if !strings.Contains(nm.state.StatusMsg, "oops") {
		t.Errorf("status = %q", nm.state.StatusMsg)
	}
}

func TestUpdateAddedMsg_Success(t *testing.T) {
	m := sizedModel()
	now := time.Now()
	m.state.SelectedTask = &domain.Task{ID: "t1"}

	u := &domain.Update{ID: "u1", TaskID: "t1", Body: "note", CreatedAt: now}
	_, cmd := m.Update(updateAddedMsg{update: u})
	if cmd == nil {
		t.Error("expected loadUpdates cmd")
	}
}

func TestUpdateAddedMsg_NoSelectedTask(t *testing.T) {
	m := sizedModel()
	m.state.SelectedTask = nil

	u := &domain.Update{ID: "u1", TaskID: "t1"}
	_, cmd := m.Update(updateAddedMsg{update: u})
	if cmd != nil {
		t.Error("expected nil cmd when no task selected")
	}
}

func TestUpdateAddedMsg_Error(t *testing.T) {
	m := sizedModel()
	newModel, _ := m.Update(updateAddedMsg{err: fmt.Errorf("fail")})
	nm := newModel.(Model)
	if !strings.Contains(nm.state.StatusMsg, "fail") {
		t.Errorf("status = %q", nm.state.StatusMsg)
	}
}

func TestUpdatesLoadedMsg_Success(t *testing.T) {
	m := sizedModel()
	m.state.SelectedTask = &domain.Task{ID: "t1", Title: "Test"}

	updates := []domain.Update{{ID: "u1", TaskID: "t1", Body: "note"}}
	newModel, _ := m.Update(updatesLoadedMsg{updates: updates, taskID: "t1"})
	nm := newModel.(Model)
	if nm.state.SelectedTask.Updates == nil || len(nm.state.SelectedTask.Updates) != 1 {
		t.Error("updates not attached to selected task")
	}
}

func TestUpdatesLoadedMsg_DifferentTask(t *testing.T) {
	m := sizedModel()
	m.state.SelectedTask = &domain.Task{ID: "t1"}

	// Updates for a different task — should be ignored
	newModel, _ := m.Update(updatesLoadedMsg{updates: []domain.Update{{ID: "u1", TaskID: "t2"}}, taskID: "t2"})
	nm := newModel.(Model)
	if len(nm.state.SelectedTask.Updates) != 0 {
		t.Error("should not apply updates for different task")
	}
}

func TestUpdatesLoadedMsg_Error(t *testing.T) {
	m := sizedModel()
	newModel, _ := m.Update(updatesLoadedMsg{err: fmt.Errorf("err")})
	nm := newModel.(Model)
	if !strings.Contains(nm.state.StatusMsg, "err") {
		t.Errorf("status = %q", nm.state.StatusMsg)
	}
}

func TestListCreatedMsg_Success(t *testing.T) {
	m := sizedModel()
	m.state.ActiveOverlay = OverlayListSwitch

	list := &domain.TaskList{ID: "new-l", Name: "Work"}
	newModel, cmd := m.Update(listCreatedMsg{list: list})
	nm := newModel.(Model)
	if nm.state.ActiveOverlay != OverlayNone {
		t.Error("overlay should close")
	}
	if nm.state.ActiveListID != "new-l" {
		t.Errorf("active list = %q, want new-l", nm.state.ActiveListID)
	}
	if !strings.Contains(nm.state.StatusMsg, "Work") {
		t.Errorf("status = %q", nm.state.StatusMsg)
	}
	if cmd == nil {
		t.Error("expected batch cmd")
	}
}

func TestListCreatedMsg_Error(t *testing.T) {
	m := sizedModel()
	newModel, _ := m.Update(listCreatedMsg{err: fmt.Errorf("dup")})
	nm := newModel.(Model)
	if !strings.Contains(nm.state.StatusMsg, "dup") {
		t.Errorf("status = %q", nm.state.StatusMsg)
	}
}

func TestListUpdatedMsg_Success(t *testing.T) {
	m := sizedModel()
	m.state.ActiveOverlay = OverlayListSwitch

	list := &domain.TaskList{ID: "l1", Name: "Renamed"}
	newModel, cmd := m.Update(listUpdatedMsg{list: list})
	nm := newModel.(Model)
	if nm.state.ActiveOverlay != OverlayNone {
		t.Error("overlay should close")
	}
	if !strings.Contains(nm.state.StatusMsg, "Renamed") {
		t.Errorf("status = %q", nm.state.StatusMsg)
	}
	if cmd == nil {
		t.Error("expected loadLists cmd")
	}
}

func TestListDeletedMsg_Success(t *testing.T) {
	m := sizedModel()
	m.state.ActiveOverlay = OverlayListSwitch
	m.state.ActiveListID = "l1"

	newModel, cmd := m.Update(listDeletedMsg{id: "l1"})
	nm := newModel.(Model)
	if nm.state.ActiveOverlay != OverlayNone {
		t.Error("overlay should close")
	}
	if nm.state.ActiveListID != "" {
		t.Errorf("active list should be cleared, got %q", nm.state.ActiveListID)
	}
	if cmd == nil {
		t.Error("expected batch cmd")
	}
}

func TestListDeletedMsg_DifferentList(t *testing.T) {
	m := sizedModel()
	m.state.ActiveListID = "l1"

	newModel, _ := m.Update(listDeletedMsg{id: "l2"})
	nm := newModel.(Model)
	if nm.state.ActiveListID != "l1" {
		t.Errorf("active list should not change when deleting different list")
	}
}

func TestPromptPasswordMsg(t *testing.T) {
	m := sizedModel()
	newModel, _ := m.Update(promptPasswordMsg{remoteName: "work", label: "work@db.example.com"})
	nm := newModel.(Model)
	if nm.state.ActiveOverlay != OverlayPasswordPrompt {
		t.Errorf("overlay = %d, want OverlayPasswordPrompt", nm.state.ActiveOverlay)
	}
	if nm.pendingRemoteName != "work" {
		t.Errorf("pending remote = %q", nm.pendingRemoteName)
	}
}

// ---- View rendering tests ---------------------------------------------------

func TestView_QuittingEmpty(t *testing.T) {
	m := sizedModel()
	m.quitting = true
	if out := m.View(); out != "" {
		t.Errorf("quitting view should be empty, got %q", out)
	}
}

func TestView_TooSmallTerminal(t *testing.T) {
	m := newTestModel()
	m.width = 30
	m.height = 10
	m.recalcLayout()
	out := m.View()
	if !strings.Contains(out, "too small") {
		t.Errorf("expected 'too small' message, got %q", out)
	}
}

func TestView_NormalRender(t *testing.T) {
	m := sizedModel()
	out := m.View()
	if out == "" {
		t.Error("View should not be empty for sized model")
	}
	// Should contain tab labels
	if !strings.Contains(out, "Todo") && !strings.Contains(out, "todo") {
		t.Error("View should contain tab labels")
	}
}

func TestView_WithOverlays(t *testing.T) {
	overlays := []struct {
		name    string
		overlay Overlay
	}{
		{"edit", OverlayEdit},
		{"filter", OverlayFilter},
		{"help", OverlayHelp},
		{"list-switch", OverlayListSwitch},
		{"sort", OverlaySort},
	}
	for _, tt := range overlays {
		t.Run(tt.name, func(t *testing.T) {
			m := sizedModel()
			m.state.ActiveOverlay = tt.overlay
			if tt.overlay == OverlayHelp {
				m.initHelpViewport()
			}
			if tt.overlay == OverlaySort {
				m.sortView = views.NewSortView(m.styles)
			}
			out := m.View()
			if out == "" {
				t.Error("View should not be empty")
			}
		})
	}
}

// ---- overlay key handler tests ----------------------------------------------

func TestEditOverlay_EscCloses(t *testing.T) {
	m := sizedModel()
	m.state.ActiveOverlay = OverlayEdit
	newModel, _ := m.Update(specialKey(tea.KeyEsc))
	nm := newModel.(Model)
	if nm.state.ActiveOverlay != OverlayNone {
		t.Error("Esc should close edit overlay")
	}
}

func TestFilterOverlay_EscCloses(t *testing.T) {
	m := sizedModel()
	m.state.ActiveOverlay = OverlayFilter
	newModel, _ := m.Update(specialKey(tea.KeyEsc))
	nm := newModel.(Model)
	if nm.state.ActiveOverlay != OverlayNone {
		t.Error("Esc should close filter overlay")
	}
}

func TestHelpOverlay_QCloses(t *testing.T) {
	m := sizedModel()
	m.state.ActiveOverlay = OverlayHelp
	m.initHelpViewport()
	newModel, _ := m.Update(keyMsg("q"))
	nm := newModel.(Model)
	if nm.state.ActiveOverlay != OverlayNone {
		t.Error("q should close help overlay")
	}
}

func TestHelpOverlay_QuestionMarkCloses(t *testing.T) {
	m := sizedModel()
	m.state.ActiveOverlay = OverlayHelp
	m.initHelpViewport()
	newModel, _ := m.Update(keyMsg("?"))
	nm := newModel.(Model)
	if nm.state.ActiveOverlay != OverlayNone {
		t.Error("? should toggle help overlay off")
	}
}

func TestListSwitchOverlay_EscCloses(t *testing.T) {
	m := sizedModel()
	m.state.ActiveOverlay = OverlayListSwitch
	newModel, _ := m.Update(specialKey(tea.KeyEsc))
	nm := newModel.(Model)
	if nm.state.ActiveOverlay != OverlayNone {
		t.Error("Esc should close list switch overlay")
	}
}

func TestListSwitchOverlay_Navigation(t *testing.T) {
	m := sizedModel()
	m.state.ActiveOverlay = OverlayListSwitch
	m.state.TaskLists = []domain.TaskList{
		{ID: "l1", Name: "List1"},
		{ID: "l2", Name: "List2"},
	}

	// Move down
	newModel, _ := m.Update(keyMsg("j"))
	nm := newModel.(Model)
	if nm.listSwitchCursor != 1 {
		t.Errorf("cursor = %d, want 1", nm.listSwitchCursor)
	}

	// Move up
	newModel2, _ := nm.Update(keyMsg("k"))
	nm2 := newModel2.(Model)
	if nm2.listSwitchCursor != 0 {
		t.Errorf("cursor = %d, want 0", nm2.listSwitchCursor)
	}
}

func TestListSwitchOverlay_Select(t *testing.T) {
	m := sizedModel()
	m.state.ActiveOverlay = OverlayListSwitch
	m.state.TaskLists = []domain.TaskList{{ID: "l1", Name: "Work"}}
	m.cfg.DataDir = t.TempDir()

	newModel, cmd := m.Update(specialKey(tea.KeyEnter))
	nm := newModel.(Model)
	if nm.state.ActiveOverlay != OverlayNone {
		t.Error("overlay should close after selection")
	}
	if nm.state.ActiveListID != "l1" {
		t.Errorf("active list = %q, want l1", nm.state.ActiveListID)
	}
	if cmd == nil {
		t.Error("expected loadTasks cmd")
	}
}

func TestListSwitchOverlay_RenameMode(t *testing.T) {
	m := sizedModel()
	m.state.ActiveOverlay = OverlayListSwitch
	m.state.TaskLists = []domain.TaskList{{ID: "l1", Name: "Old"}}

	newModel, _ := m.Update(keyMsg("r"))
	nm := newModel.(Model)
	if nm.listSwitchMode != 2 {
		t.Errorf("listSwitchMode = %d, want 2 (rename)", nm.listSwitchMode)
	}
}

func TestListSwitchOverlay_CreateModeEsc(t *testing.T) {
	m := sizedModel()
	m.state.ActiveOverlay = OverlayListSwitch
	m.listSwitchMode = 1

	newModel, _ := m.Update(specialKey(tea.KeyEsc))
	nm := newModel.(Model)
	if nm.listSwitchMode != 0 {
		t.Errorf("listSwitchMode = %d, want 0 after Esc", nm.listSwitchMode)
	}
}

func TestListSwitchOverlay_RenameModeEsc(t *testing.T) {
	m := sizedModel()
	m.state.ActiveOverlay = OverlayListSwitch
	m.listSwitchMode = 2

	newModel, _ := m.Update(specialKey(tea.KeyEsc))
	nm := newModel.(Model)
	if nm.listSwitchMode != 0 {
		t.Errorf("listSwitchMode = %d, want 0 after Esc", nm.listSwitchMode)
	}
}

func TestListSwitchOverlay_DeleteFullFlow(t *testing.T) {
	m := sizedModel()
	m.state.ActiveOverlay = OverlayListSwitch
	m.state.TaskLists = []domain.TaskList{{ID: "l1", Name: "Target"}}

	// d -> confirm1(y) -> confirm2(y) -> should fire delete cmd
	m1, _ := m.Update(keyMsg("d"))
	nm1 := m1.(Model)
	if nm1.listSwitchMode != 3 {
		t.Fatalf("mode = %d, want 3", nm1.listSwitchMode)
	}

	m2, _ := nm1.Update(keyMsg("y"))
	nm2 := m2.(Model)
	if nm2.listSwitchMode != 4 {
		t.Fatalf("mode = %d, want 4", nm2.listSwitchMode)
	}

	m3, cmd := nm2.Update(keyMsg("y"))
	nm3 := m3.(Model)
	_ = nm3
	if cmd == nil {
		t.Error("expected deleteList cmd after double confirm")
	}
}

// ---- sort overlay tests -----------------------------------------------------

func TestSortOverlay_OpenAndClose(t *testing.T) {
	m := sizedModel()

	// Open sort
	newModel, _ := m.Update(keyMsg("S"))
	nm := newModel.(Model)
	if nm.state.ActiveOverlay != OverlaySort {
		t.Errorf("overlay = %d, want OverlaySort", nm.state.ActiveOverlay)
	}

	// Close with Esc
	newModel2, _ := nm.Update(specialKey(tea.KeyEsc))
	nm2 := newModel2.(Model)
	if nm2.state.ActiveOverlay != OverlayNone {
		t.Error("Esc should close sort overlay")
	}
}

func TestSortOverlay_Navigation(t *testing.T) {
	m := sizedModel()
	m2, _ := m.Update(keyMsg("S"))
	nm := m2.(Model)

	// Navigate down
	m3, _ := nm.Update(keyMsg("j"))
	nm2 := m3.(Model)
	_ = nm2

	// Navigate up
	m4, _ := nm2.Update(keyMsg("k"))
	_ = m4
}

func TestSortOverlay_Select(t *testing.T) {
	m := sizedModel()
	m2, _ := m.Update(keyMsg("S"))
	nm := m2.(Model)

	m3, cmd := nm.Update(specialKey(tea.KeyEnter))
	nm2 := m3.(Model)
	if nm2.state.ActiveOverlay != OverlayNone {
		t.Error("overlay should close after sort selection")
	}
	if !strings.Contains(nm2.state.StatusMsg, "Sorted") {
		t.Errorf("status = %q, want 'Sorted by...'", nm2.state.StatusMsg)
	}
	if cmd == nil {
		t.Error("expected loadTasks cmd")
	}
}

// ---- theme picker tests -----------------------------------------------------

func TestThemePicker_OpenAndEsc(t *testing.T) {
	m := sizedModel()
	m.cfg.DataDir = t.TempDir()

	// Open theme picker
	newModel, _ := m.Update(keyMsg("T"))
	nm := newModel.(Model)
	if nm.state.ActiveOverlay != OverlayThemePicker {
		t.Errorf("overlay = %d, want OverlayThemePicker", nm.state.ActiveOverlay)
	}

	// Esc should revert
	newModel2, _ := nm.Update(specialKey(tea.KeyEsc))
	nm2 := newModel2.(Model)
	if nm2.state.ActiveOverlay != OverlayNone {
		t.Error("Esc should close theme picker")
	}
}

func TestThemePicker_Navigation(t *testing.T) {
	m := sizedModel()
	m.cfg.DataDir = t.TempDir()

	m2, _ := m.Update(keyMsg("T"))
	nm := m2.(Model)

	// Navigate down
	m3, _ := nm.Update(keyMsg("j"))
	nm2 := m3.(Model)
	_ = nm2

	// Navigate up
	m4, _ := nm2.Update(keyMsg("k"))
	_ = m4
}

func TestThemePicker_Select(t *testing.T) {
	m := sizedModel()
	m.cfg.DataDir = t.TempDir()

	m2, _ := m.Update(keyMsg("T"))
	nm := m2.(Model)

	m3, _ := nm.Update(specialKey(tea.KeyEnter))
	nm2 := m3.(Model)
	if nm2.state.ActiveOverlay != OverlayNone {
		t.Error("overlay should close after selection")
	}
}

// ---- password overlay tests -------------------------------------------------

func TestPasswordOverlay_EscCloses(t *testing.T) {
	m := sizedModel()
	m.state.ActiveOverlay = OverlayPasswordPrompt
	m.pendingRemoteName = "work"

	newModel, _ := m.Update(specialKey(tea.KeyEsc))
	nm := newModel.(Model)
	if nm.state.ActiveOverlay != OverlayNone {
		t.Error("Esc should close password overlay")
	}
	if nm.pendingRemoteName != "" {
		t.Errorf("pending remote should be cleared, got %q", nm.pendingRemoteName)
	}
}

// ---- remote picker tests ----------------------------------------------------

func TestRemotePicker_NoRemotesNoop(t *testing.T) {
	m := sizedModel()
	// No remotes configured — RemoteSwitch should be a no-op
	newModel, _ := m.Update(keyMsg("R"))
	nm := newModel.(Model)
	if nm.state.ActiveOverlay != OverlayNone {
		t.Error("remote picker should not open without configured remotes")
	}
}

func TestRemotePicker_WithRemotes(t *testing.T) {
	m := sizedModel()
	m.cfg.Remotes = map[string]*config.RemoteConfig{
		"work": {Driver: "mysql", Host: "db.example.com"},
	}

	newModel, _ := m.Update(keyMsg("R"))
	nm := newModel.(Model)
	if nm.state.ActiveOverlay != OverlayRemotePicker {
		t.Errorf("overlay = %d, want OverlayRemotePicker", nm.state.ActiveOverlay)
	}
	if len(nm.remotePickerChoices) != 2 { // "local" + "work"
		t.Errorf("choices = %d, want 2", len(nm.remotePickerChoices))
	}
}

func TestRemotePicker_EscCloses(t *testing.T) {
	m := sizedModel()
	m.state.ActiveOverlay = OverlayRemotePicker
	m.remotePickerChoices = []string{"local", "work"}
	m.remotePickerLabels = []string{"Local", "Work"}

	newModel, _ := m.Update(specialKey(tea.KeyEsc))
	nm := newModel.(Model)
	if nm.state.ActiveOverlay != OverlayNone {
		t.Error("Esc should close remote picker")
	}
}

func TestRemotePicker_Navigation(t *testing.T) {
	m := sizedModel()
	m.state.ActiveOverlay = OverlayRemotePicker
	m.remotePickerChoices = []string{"local", "work"}
	m.remotePickerLabels = []string{"Local", "Work"}

	m2, _ := m.Update(keyMsg("j"))
	nm := m2.(Model)
	if nm.remotePickerCursor != 1 {
		t.Errorf("cursor = %d, want 1", nm.remotePickerCursor)
	}

	m3, _ := nm.Update(keyMsg("k"))
	nm2 := m3.(Model)
	if nm2.remotePickerCursor != 0 {
		t.Errorf("cursor = %d, want 0", nm2.remotePickerCursor)
	}
}

func TestRemoteConnectedMsg_Success(t *testing.T) {
	m := sizedModel()
	m.cfg.DataDir = t.TempDir()

	taskSvc := application.NewTaskService(newStubTaskRepoWithData(), newStubUpdateRepo(), &stubSearch{}, nil)
	listSvc := application.NewListService(newStubListRepoWithData(), nil)

	// Use a mock closer
	closer := &mockCloser{}
	m.currentDB = closer

	svc := &bootstrap.Services{TaskSvc: taskSvc, ListSvc: listSvc}
	newModel, cmd := m.Update(remoteConnectedMsg{svc: svc, remoteName: "work"})
	nm := newModel.(Model)
	if !strings.Contains(nm.state.StatusMsg, "Connected") {
		t.Errorf("status = %q", nm.state.StatusMsg)
	}
	if cmd == nil {
		t.Error("expected batch load cmd")
	}
	if !closer.closed {
		t.Error("previous DB should be closed")
	}
}

func TestRemoteConnectedMsg_Error(t *testing.T) {
	m := sizedModel()
	m.cfg.DataDir = t.TempDir()
	newModel, _ := m.Update(remoteConnectedMsg{err: fmt.Errorf("conn refused")})
	nm := newModel.(Model)
	if !strings.Contains(nm.state.StatusMsg, "conn refused") {
		t.Errorf("status = %q", nm.state.StatusMsg)
	}
}

// ---- detail pane tests ------------------------------------------------------

func TestDetailPane_EscReturnsToList(t *testing.T) {
	m := sizedModel()
	m.state.ActivePane = PaneDetail

	newModel, _ := m.Update(specialKey(tea.KeyEsc))
	nm := newModel.(Model)
	if nm.state.ActivePane != PaneList {
		t.Errorf("pane = %d, want PaneList", nm.state.ActivePane)
	}
}

func TestDetailPane_TabCyclesFocus(t *testing.T) {
	m := sizedModel()
	m.state.ActivePane = PaneDetail
	m.state.DetailFocus = FocusDetails

	newModel, _ := m.Update(specialKey(tea.KeyTab))
	nm := newModel.(Model)
	if nm.state.DetailFocus != FocusUpdates {
		t.Errorf("focus = %d, want FocusUpdates", nm.state.DetailFocus)
	}

	newModel2, _ := nm.Update(specialKey(tea.KeyTab))
	nm2 := newModel2.(Model)
	if nm2.state.DetailFocus != FocusDetails {
		t.Errorf("focus = %d, want FocusDetails", nm2.state.DetailFocus)
	}
}

func TestDetailPane_ShiftTabReversesFocus(t *testing.T) {
	m := sizedModel()
	m.state.ActivePane = PaneDetail
	m.state.DetailFocus = FocusDetails

	newModel, _ := m.Update(specialKey(tea.KeyShiftTab))
	nm := newModel.(Model)
	if nm.state.DetailFocus != FocusUpdates {
		t.Errorf("focus = %d, want FocusUpdates", nm.state.DetailFocus)
	}
}

func TestDetailPane_EditOpensOverlay(t *testing.T) {
	m := sizedModel()
	m.state.ActivePane = PaneDetail
	m.state.SelectedTask = &domain.Task{ID: "t1", Title: "Test"}

	newModel, _ := m.Update(keyMsg("e"))
	nm := newModel.(Model)
	if nm.state.ActiveOverlay != OverlayEdit {
		t.Error("e should open edit overlay from detail pane")
	}
}

func TestDetailPane_QuitStillWorks(t *testing.T) {
	m := sizedModel()
	m.state.ActivePane = PaneDetail

	newModel, cmd := m.Update(keyMsg("q"))
	nm := newModel.(Model)
	if !nm.quitting {
		t.Error("q should quit from detail pane")
	}
	if cmd == nil {
		t.Error("expected quit cmd")
	}
}

func TestDetailPane_HelpFromDetail(t *testing.T) {
	m := sizedModel()
	m.state.ActivePane = PaneDetail

	newModel, _ := m.Update(keyMsg("?"))
	nm := newModel.(Model)
	if nm.state.ActiveOverlay != OverlayHelp {
		t.Error("? should open help from detail pane")
	}
}

// ---- helper functions -------------------------------------------------------

func TestNextTab(t *testing.T) {
	m := sizedModel()
	m.state.ActiveTab = 0
	if got := m.nextTab(); got != 1 {
		t.Errorf("nextTab() = %d, want 1", got)
	}
	m.state.ActiveTab = 3 // last
	if got := m.nextTab(); got != 0 {
		t.Errorf("nextTab() wraps: got %d, want 0", got)
	}
}

func TestPrevTab(t *testing.T) {
	m := sizedModel()
	m.state.ActiveTab = 0
	if got := m.prevTab(); got != 3 {
		t.Errorf("prevTab() wraps: got %d, want 3", got)
	}
	m.state.ActiveTab = 2
	if got := m.prevTab(); got != 1 {
		t.Errorf("prevTab() = %d, want 1", got)
	}
}

// ---- KeyMap tests -----------------------------------------------------------

func TestNewKeyMap_DefaultKeys(t *testing.T) {
	km := NewKeyMap(config.KeyMap{
		Up: "k", Down: "j", Left: "h", Right: "l",
		Quit: "q", Help: "?", New: "n", Edit: "e",
		Delete: "d", Search: "/", Done: "x", Status: "s",
		Select: "enter",
	})
	// ThemePicker defaults to "T" when not set
	if len(km.ThemePicker.Keys()) == 0 {
		t.Error("ThemePicker should have keys")
	}
	// ListSwitch defaults to "L"
	if len(km.ListSwitch.Keys()) == 0 {
		t.Error("ListSwitch should have keys")
	}
}

func TestShortHelp(t *testing.T) {
	km := NewKeyMap(config.KeyMap{
		Up: "k", Down: "j", Left: "h", Right: "l",
		Quit: "q", Help: "?", New: "n",
		Done: "x", Status: "s", Select: "enter",
	})
	bindings := km.ShortHelp()
	if len(bindings) == 0 {
		t.Error("ShortHelp should return bindings")
	}
}

func TestFullHelp(t *testing.T) {
	km := NewKeyMap(config.KeyMap{
		Up: "k", Down: "j", Left: "h", Right: "l",
		Quit: "q", Help: "?", New: "n", Edit: "e",
		Delete: "d", Search: "/", Done: "x", Status: "s",
		Select: "enter",
	})
	groups := km.FullHelp()
	if len(groups) != 3 {
		t.Errorf("FullHelp groups = %d, want 3", len(groups))
	}
}

// ---- mock helpers -----------------------------------------------------------

type mockCloser struct{ closed bool }

func (c *mockCloser) Close() error { c.closed = true; return nil }
