package cli

import (
	"bytes"
	"testing"

	"github.com/diptopandit/tork/internal/application"
	"github.com/diptopandit/tork/internal/domain"
	"github.com/diptopandit/tork/internal/infrastructure/config"
)

// ---- stubs ------------------------------------------------------------------

type stubTaskRepo struct {
	tasks map[string]*domain.Task
	numID int
}

func newStubTaskRepo() *stubTaskRepo {
	return &stubTaskRepo{tasks: map[string]*domain.Task{}, numID: 0}
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

type stubUpdateRepo struct{}

func (s *stubUpdateRepo) AddUpdate(u *domain.Update) error             { return nil }
func (s *stubUpdateRepo) ListByTaskID(string) ([]domain.Update, error) { return nil, nil }

type stubSearch struct{}

func (s *stubSearch) Index(*domain.Task) error        { return nil }
func (s *stubSearch) Delete(string) error             { return nil }
func (s *stubSearch) Search(string) ([]string, error) { return nil, nil }

type stubListRepo struct{ lists map[string]*domain.TaskList }

func newStubListRepo() *stubListRepo {
	return &stubListRepo{lists: map[string]*domain.TaskList{}}
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

// ---- helpers ----------------------------------------------------------------

func testCfg() *config.Config {
	return &config.Config{
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
		Display: config.DisplayConfig{DateFormat: "02-01-2006"},
	}
}

func testServices() (*application.TaskService, *application.ListService) {
	return application.NewTaskService(newStubTaskRepo(), &stubUpdateRepo{}, &stubSearch{}, nil),
		application.NewListService(newStubListRepo(), nil)
}

// ---- parser tests -----------------------------------------------------------

func TestParseDate_Valid(t *testing.T) {
	for _, tt := range []struct{ in, want string }{
		{"15-04-2026", "2026-04-15"},
		{"2026-04-15", "2026-04-15"},
	} {
		d, err := ParseDate(tt.in)
		if err != nil {
			t.Errorf("ParseDate(%q): %v", tt.in, err)
			continue
		}
		if d == nil {
			t.Errorf("ParseDate(%q) = nil", tt.in)
			continue
		}
		if got := d.Format("2006-01-02"); got != tt.want {
			t.Errorf("ParseDate(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestParseDate_Empty(t *testing.T) {
	d, err := ParseDate("")
	if err != nil {
		t.Fatal(err)
	}
	if d != nil {
		t.Error("expected nil")
	}
}

func TestParseDate_Invalid(t *testing.T) {
	for _, s := range []string{"not-a-date", "2026/04/15", "15042026"} {
		if _, err := ParseDate(s); err == nil {
			t.Errorf("ParseDate(%q) should error", s)
		}
	}
}

func TestParsePriority_ByName(t *testing.T) {
	p, err := ParsePriority(testCfg().Priorities, "high")
	if err != nil {
		t.Fatal(err)
	}
	if int(p) != 3 {
		t.Errorf("priority = %d, want 3", p)
	}
}

func TestParsePriority_ByPosition(t *testing.T) {
	p, err := ParsePriority(testCfg().Priorities, "3")
	if err != nil {
		t.Fatal(err)
	}
	if int(p) != 3 {
		t.Errorf("priority = %d, want 3", p)
	}
}

func TestParsePriority_Empty(t *testing.T) {
	p, err := ParsePriority(testCfg().Priorities, "")
	if err != nil {
		t.Fatal(err)
	}
	if int(p) != 2 {
		t.Errorf("priority = %d, want 2", p)
	}
}

func TestParsePriority_Invalid(t *testing.T) {
	if _, err := ParsePriority(testCfg().Priorities, "nonexistent"); err == nil {
		t.Error("expected error")
	}
}

func TestParsePriority_OutOfRange(t *testing.T) {
	if _, err := ParsePriority(testCfg().Priorities, "99"); err == nil {
		t.Error("expected error for out-of-range position")
	}
}

func TestParseStatus_Valid(t *testing.T) {
	s, err := ParseStatus(testCfg().Statuses, "done")
	if err != nil {
		t.Fatal(err)
	}
	if string(s) != "done" {
		t.Errorf("status = %q", s)
	}
}

func TestParseStatus_DashForm(t *testing.T) {
	s, err := ParseStatus(testCfg().Statuses, "in-progress")
	if err != nil {
		t.Fatal(err)
	}
	if string(s) != "in_progress" {
		t.Errorf("status = %q", s)
	}
}

func TestParseStatus_Invalid(t *testing.T) {
	if _, err := ParseStatus(testCfg().Statuses, "bogus"); err == nil {
		t.Error("expected error")
	}
}

// ---- resolveTaskID tests ----------------------------------------------------

func TestResolveTaskID_Numeric(t *testing.T) {
	repo := newStubTaskRepo()
	svc := application.NewTaskService(repo, &stubUpdateRepo{}, &stubSearch{}, nil)
	task, _ := svc.CreateTask(application.CreateTaskInput{
		ListID: "list-1", Title: "Test", Status: domain.StatusTodo, Priority: domain.PriorityMedium,
	})

	id, err := resolveTaskID(svc, "1")
	if err != nil {
		t.Fatal(err)
	}
	if id != task.ID {
		t.Errorf("id = %q, want %q", id, task.ID)
	}
}

func TestResolveTaskID_WithHash(t *testing.T) {
	repo := newStubTaskRepo()
	svc := application.NewTaskService(repo, &stubUpdateRepo{}, &stubSearch{}, nil)
	task, _ := svc.CreateTask(application.CreateTaskInput{
		ListID: "list-1", Title: "Test", Status: domain.StatusTodo, Priority: domain.PriorityMedium,
	})

	id, err := resolveTaskID(svc, "#1")
	if err != nil {
		t.Fatal(err)
	}
	if id != task.ID {
		t.Errorf("id = %q, want %q", id, task.ID)
	}
}

func TestResolveTaskID_UUID(t *testing.T) {
	svc := application.NewTaskService(newStubTaskRepo(), &stubUpdateRepo{}, &stubSearch{}, nil)
	id, err := resolveTaskID(svc, "some-uuid-string")
	if err != nil {
		t.Fatal(err)
	}
	if id != "some-uuid-string" {
		t.Errorf("id = %q", id)
	}
}

func TestResolveTaskID_NumericNotFound(t *testing.T) {
	svc := application.NewTaskService(newStubTaskRepo(), &stubUpdateRepo{}, &stubSearch{}, nil)
	_, err := resolveTaskID(svc, "999")
	if err == nil {
		t.Error("expected error")
	}
}

// ---- cobra command tests ----------------------------------------------------

func TestAddCmd_CreatesTask(t *testing.T) {
	taskSvc, listSvc := testServices()
	root := NewRootCmd(taskSvc, listSvc, testCfg(), "test")
	root.SetOut(new(bytes.Buffer))
	root.SetErr(new(bytes.Buffer))
	root.SetArgs([]string{"add", "Test task", "--priority", "high"})
	if err := root.Execute(); err != nil {
		t.Fatalf("add: %v", err)
	}
}

func TestAddCmd_NoArgs(t *testing.T) {
	taskSvc, listSvc := testServices()
	root := NewRootCmd(taskSvc, listSvc, testCfg(), "test")
	root.SetArgs([]string{"add"})
	root.SilenceErrors = true
	root.SilenceUsage = true
	if err := root.Execute(); err == nil {
		t.Error("expected error for missing title")
	}
}

func TestAddCmd_WithDueDate(t *testing.T) {
	taskSvc, listSvc := testServices()
	root := NewRootCmd(taskSvc, listSvc, testCfg(), "test")
	root.SetOut(new(bytes.Buffer))
	root.SetErr(new(bytes.Buffer))
	root.SetArgs([]string{"add", "Dated", "--due", "15-04-2026"})
	if err := root.Execute(); err != nil {
		t.Fatalf("add with due: %v", err)
	}
}

func TestAddCmd_WithTags(t *testing.T) {
	taskSvc, listSvc := testServices()
	root := NewRootCmd(taskSvc, listSvc, testCfg(), "test")
	root.SetOut(new(bytes.Buffer))
	root.SetErr(new(bytes.Buffer))
	root.SetArgs([]string{"add", "Tagged", "--tags", "go,tui"})
	if err := root.Execute(); err != nil {
		t.Fatalf("add with tags: %v", err)
	}
}

func TestAddCmd_InvalidPriority(t *testing.T) {
	taskSvc, listSvc := testServices()
	root := NewRootCmd(taskSvc, listSvc, testCfg(), "test")
	root.SilenceErrors = true
	root.SilenceUsage = true
	root.SetArgs([]string{"add", "Bad", "--priority", "invalid"})
	if err := root.Execute(); err == nil {
		t.Error("expected error for invalid priority")
	}
}

func TestAddCmd_InvalidDueDate(t *testing.T) {
	taskSvc, listSvc := testServices()
	root := NewRootCmd(taskSvc, listSvc, testCfg(), "test")
	root.SilenceErrors = true
	root.SilenceUsage = true
	root.SetArgs([]string{"add", "Bad", "--due", "not-a-date"})
	if err := root.Execute(); err == nil {
		t.Error("expected error for invalid date")
	}
}

func TestListCmd_Empty(t *testing.T) {
	taskSvc, listSvc := testServices()
	root := NewRootCmd(taskSvc, listSvc, testCfg(), "test")
	root.SetOut(new(bytes.Buffer))
	root.SetArgs([]string{"list"})
	if err := root.Execute(); err != nil {
		t.Fatalf("list: %v", err)
	}
}

func TestListCreateCmd(t *testing.T) {
	taskSvc, listSvc := testServices()
	root := NewRootCmd(taskSvc, listSvc, testCfg(), "test")
	root.SetOut(new(bytes.Buffer))
	root.SetErr(new(bytes.Buffer))
	root.SetArgs([]string{"list-create", "Work"})
	if err := root.Execute(); err != nil {
		t.Fatalf("list-create: %v", err)
	}
}

func TestListsCmd(t *testing.T) {
	taskSvc, listSvc := testServices()
	root := NewRootCmd(taskSvc, listSvc, testCfg(), "test")
	root.SetOut(new(bytes.Buffer))
	root.SetArgs([]string{"lists"})
	if err := root.Execute(); err != nil {
		t.Fatalf("lists: %v", err)
	}
}

func TestDeleteCmd_NoArgs(t *testing.T) {
	taskSvc, listSvc := testServices()
	root := NewRootCmd(taskSvc, listSvc, testCfg(), "test")
	root.SetArgs([]string{"delete"})
	root.SilenceErrors = true
	root.SilenceUsage = true
	if err := root.Execute(); err == nil {
		t.Error("expected error")
	}
}

func TestSearchCmd_NoArgs(t *testing.T) {
	taskSvc, listSvc := testServices()
	root := NewRootCmd(taskSvc, listSvc, testCfg(), "test")
	root.SetArgs([]string{"search"})
	root.SilenceErrors = true
	root.SilenceUsage = true
	if err := root.Execute(); err == nil {
		t.Error("expected error")
	}
}

func TestSearchCmd_NoResults(t *testing.T) {
	taskSvc, listSvc := testServices()
	root := NewRootCmd(taskSvc, listSvc, testCfg(), "test")
	root.SetOut(new(bytes.Buffer))
	root.SetArgs([]string{"search", "nothing"})
	if err := root.Execute(); err != nil {
		t.Fatalf("search: %v", err)
	}
}

func TestListDeleteCmd_NoForce(t *testing.T) {
	taskSvc, listSvc := testServices()
	root := NewRootCmd(taskSvc, listSvc, testCfg(), "test")
	root.SetArgs([]string{"list-delete", "some-id"})
	root.SilenceErrors = true
	root.SilenceUsage = true
	if err := root.Execute(); err == nil {
		t.Error("expected error without --force")
	}
}

func TestVersionOutput(t *testing.T) {
	taskSvc, listSvc := testServices()
	root := NewRootCmd(taskSvc, listSvc, testCfg(), "v1.0.0 (abc123)")
	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetArgs([]string{"--version"})
	if err := root.Execute(); err != nil {
		t.Fatalf("version: %v", err)
	}
	if !bytes.Contains(buf.Bytes(), []byte("v1.0.0")) {
		t.Errorf("output = %q, missing v1.0.0", buf.String())
	}
}
