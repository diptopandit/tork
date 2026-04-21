package application

import (
	"fmt"
	"testing"
	"time"

	"github.com/diptopandit/tork/internal/domain"
)

// ---- mock repo ---------------------------------------------------------------

type mockTaskRepo struct {
	tasks map[string]*domain.Task
}

func newMockTaskRepo() *mockTaskRepo {
	return &mockTaskRepo{tasks: map[string]*domain.Task{}}
}

func (m *mockTaskRepo) Create(t *domain.Task) error {
	m.tasks[t.ID] = t
	return nil
}

func (m *mockTaskRepo) Update(t *domain.Task) error {
	if _, ok := m.tasks[t.ID]; !ok {
		return domain.ErrNotFound(t.ID)
	}
	m.tasks[t.ID] = t
	return nil
}

func (m *mockTaskRepo) Delete(id string) error {
	delete(m.tasks, id)
	return nil
}

func (m *mockTaskRepo) GetByID(id string) (*domain.Task, error) {
	if t, ok := m.tasks[id]; ok {
		copy := *t
		return &copy, nil
	}
	return nil, domain.ErrNotFound(id)
}

func (m *mockTaskRepo) List(filter domain.TaskFilter) ([]domain.Task, error) {
	var out []domain.Task
	for _, t := range m.tasks {
		out = append(out, *t)
	}
	return out, nil
}

func (m *mockTaskRepo) GetByNumID(numID int) (*domain.Task, error) {
	for _, t := range m.tasks {
		if t.NumID == numID {
			copy := *t
			return &copy, nil
		}
	}
	return nil, domain.ErrNotFound("num_id")
}

func (m *mockTaskRepo) NextNumID() (int, error) {
	max := 0
	for _, t := range m.tasks {
		if t.NumID > max {
			max = t.NumID
		}
	}
	return max + 1, nil
}

// ---- mock update repo --------------------------------------------------------

type mockUpdateRepo struct {
	updates []domain.Update
}

func (m *mockUpdateRepo) AddUpdate(u *domain.Update) error {
	m.updates = append(m.updates, *u)
	return nil
}

func (m *mockUpdateRepo) ListByTaskID(taskID string) ([]domain.Update, error) {
	var out []domain.Update
	for _, u := range m.updates {
		if u.TaskID == taskID {
			out = append(out, u)
		}
	}
	return out, nil
}

// ---- mock search -------------------------------------------------------------

type mockSearch struct{}

func (s *mockSearch) Index(*domain.Task) error        { return nil }
func (s *mockSearch) Delete(string) error             { return nil }
func (s *mockSearch) Search(string) ([]string, error) { return nil, nil }

// ---- tests ------------------------------------------------------------------

func newTestSvc() *TaskService {
	return NewTaskService(newMockTaskRepo(), &mockUpdateRepo{}, &mockSearch{}, nil)
}

func TestCreateTask_Valid(t *testing.T) {
	svc := newTestSvc()
	// We need a list ID first — use a placeholder.
	task, err := svc.CreateTask(CreateTaskInput{
		ListID: "list-1",
		Title:  "Buy milk",
	})
	if err != nil {
		t.Fatal(err)
	}
	if task.ID == "" {
		t.Error("expected non-empty ID")
	}
	if task.Title != "Buy milk" {
		t.Errorf("title = %q, want %q", task.Title, "Buy milk")
	}
	if task.Status != domain.StatusTodo {
		t.Errorf("status = %q, want todo", task.Status)
	}
	if task.Priority != domain.PriorityMedium {
		t.Errorf("priority = %d, want medium", task.Priority)
	}
}

func TestCreateTask_EmptyTitle(t *testing.T) {
	svc := newTestSvc()
	_, err := svc.CreateTask(CreateTaskInput{ListID: "list-1", Title: ""})
	if err == nil {
		t.Error("expected error for empty title")
	}
}

func TestCreateTask_EmptyListID(t *testing.T) {
	svc := newTestSvc()
	_, err := svc.CreateTask(CreateTaskInput{Title: "hello"})
	if err == nil {
		t.Error("expected error for empty listID")
	}
}

func TestUpdateTask(t *testing.T) {
	svc := newTestSvc()
	task, _ := svc.CreateTask(CreateTaskInput{ListID: "l", Title: "Original"})

	newTitle := "Updated"
	updated, err := svc.UpdateTask(UpdateTaskInput{
		ID:    task.ID,
		Title: &newTitle,
	})
	if err != nil {
		t.Fatal(err)
	}
	if updated.Title != "Updated" {
		t.Errorf("title = %q, want Updated", updated.Title)
	}
}

func TestDeleteTask(t *testing.T) {
	svc := newTestSvc()
	task, _ := svc.CreateTask(CreateTaskInput{ListID: "l", Title: "Temp"})
	if err := svc.DeleteTask(task.ID); err != nil {
		t.Fatal(err)
	}
	_, err := svc.GetTask(task.ID)
	if err == nil {
		t.Error("expected error after delete")
	}
}

func TestListTasks_Filter(t *testing.T) {
	svc := newTestSvc()
	svc.CreateTask(CreateTaskInput{ListID: "l", Title: "A"})
	svc.CreateTask(CreateTaskInput{ListID: "l", Title: "B"})

	tasks, err := svc.ListTasks(TaskFilter{})
	if err != nil {
		t.Fatal(err)
	}
	if len(tasks) != 2 {
		t.Errorf("got %d tasks, want 2", len(tasks))
	}
}

func TestAddUpdate(t *testing.T) {
	svc := newTestSvc()
	task, _ := svc.CreateTask(CreateTaskInput{ListID: "l", Title: "Task with updates"})

	u, err := svc.AddUpdate(task.ID, "First update")
	if err != nil {
		t.Fatal(err)
	}
	if u.Body != "First update" {
		t.Errorf("update body = %q, want %q", u.Body, "First update")
	}
	if u.TaskID != task.ID {
		t.Errorf("update taskID = %q, want %q", u.TaskID, task.ID)
	}
}

func TestAddUpdate_EmptyBody(t *testing.T) {
	svc := newTestSvc()
	task, _ := svc.CreateTask(CreateTaskInput{ListID: "l", Title: "T"})

	_, err := svc.AddUpdate(task.ID, "")
	if err == nil {
		t.Error("expected error for empty update body")
	}
}

func TestGetUpdates(t *testing.T) {
	svc := newTestSvc()
	task, _ := svc.CreateTask(CreateTaskInput{ListID: "l", Title: "T"})
	svc.AddUpdate(task.ID, "Update 1")
	svc.AddUpdate(task.ID, "Update 2")

	updates, err := svc.GetUpdates(task.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(updates) != 2 {
		t.Errorf("got %d updates, want 2", len(updates))
	}
}

func TestGetTaskByNum(t *testing.T) {
	svc := newTestSvc()
	task, _ := svc.CreateTask(CreateTaskInput{ListID: "l", Title: "Numbered"})

	got, err := svc.GetTaskByNum(task.NumID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Title != "Numbered" {
		t.Errorf("title = %q, want %q", got.Title, "Numbered")
	}
}

func TestCycleStatus(t *testing.T) {
	svc := newTestSvc()
	task, _ := svc.CreateTask(CreateTaskInput{ListID: "l", Title: "Cycle me"})

	// Initial status is todo
	if task.Status != domain.StatusTodo {
		t.Fatalf("initial status = %q, want todo", task.Status)
	}

	// Cycle to in_progress
	ip := domain.StatusInProgress
	updated, err := svc.UpdateTask(UpdateTaskInput{ID: task.ID, Status: &ip})
	if err != nil {
		t.Fatal(err)
	}
	if updated.Status != domain.StatusInProgress {
		t.Errorf("status = %q, want in_progress", updated.Status)
	}

	// Cycle to done
	done := domain.StatusDone
	updated2, err := svc.UpdateTask(UpdateTaskInput{ID: task.ID, Status: &done})
	if err != nil {
		t.Fatal(err)
	}
	if updated2.Status != domain.StatusDone {
		t.Errorf("status = %q, want done", updated2.Status)
	}

	// Cycle to cancelled
	cancelled := domain.StatusCancelled
	updated3, err := svc.UpdateTask(UpdateTaskInput{ID: task.ID, Status: &cancelled})
	if err != nil {
		t.Fatal(err)
	}
	if updated3.Status != domain.StatusCancelled {
		t.Errorf("status = %q, want cancelled", updated3.Status)
	}
}

func TestNumIDAutoIncrement(t *testing.T) {
	svc := newTestSvc()
	t1, _ := svc.CreateTask(CreateTaskInput{ListID: "l", Title: "First"})
	t2, _ := svc.CreateTask(CreateTaskInput{ListID: "l", Title: "Second"})

	if t2.NumID <= t1.NumID {
		t.Errorf("second NumID (%d) should be greater than first (%d)", t2.NumID, t1.NumID)
	}
}

// ---- error-returning mocks ---------------------------------------------------

type errTaskRepo struct {
	mockTaskRepo
}

func (m *errTaskRepo) Create(*domain.Task) error { return fmt.Errorf("db write error") }
func (m *errTaskRepo) Update(*domain.Task) error { return fmt.Errorf("db write error") }
func (m *errTaskRepo) Delete(string) error       { return fmt.Errorf("db write error") }
func (m *errTaskRepo) NextNumID() (int, error)   { return 0, fmt.Errorf("db read error") }

type errSearch struct{}

func (s *errSearch) Index(*domain.Task) error        { return nil }
func (s *errSearch) Delete(string) error             { return nil }
func (s *errSearch) Search(string) ([]string, error) { return nil, fmt.Errorf("search error") }

// searchWithResults returns predefined IDs from Search.
type searchWithResults struct {
	ids []string
}

func (s *searchWithResults) Index(*domain.Task) error { return nil }
func (s *searchWithResults) Delete(string) error      { return nil }
func (s *searchWithResults) Search(string) ([]string, error) {
	return s.ids, nil
}

// ---- additional tests -------------------------------------------------------

func TestCreateTask_WithCustomStatus(t *testing.T) {
	svc := newTestSvc()
	task, err := svc.CreateTask(CreateTaskInput{
		ListID: "l", Title: "Custom status", Status: domain.StatusInProgress,
	})
	if err != nil {
		t.Fatal(err)
	}
	if task.Status != domain.StatusInProgress {
		t.Errorf("status = %q, want in_progress", task.Status)
	}
}

func TestCreateTask_WithPriority(t *testing.T) {
	svc := newTestSvc()
	task, err := svc.CreateTask(CreateTaskInput{
		ListID: "l", Title: "High priority", Priority: domain.PriorityHigh,
	})
	if err != nil {
		t.Fatal(err)
	}
	if task.Priority != domain.PriorityHigh {
		t.Errorf("priority = %d, want %d", task.Priority, domain.PriorityHigh)
	}
}

func TestCreateTask_WithTags(t *testing.T) {
	svc := newTestSvc()
	task, err := svc.CreateTask(CreateTaskInput{
		ListID: "l", Title: "Tagged", Tags: []string{"go", "tui"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(task.Tags) != 2 {
		t.Errorf("tags count = %d, want 2", len(task.Tags))
	}
}

func TestCreateTask_NilTagsDefaulted(t *testing.T) {
	svc := newTestSvc()
	task, _ := svc.CreateTask(CreateTaskInput{ListID: "l", Title: "No tags"})
	if task.Tags == nil {
		t.Error("tags should be [] not nil")
	}
}

func TestCreateTask_RepoError(t *testing.T) {
	svc := NewTaskService(&errTaskRepo{mockTaskRepo: *newMockTaskRepo()}, &mockUpdateRepo{}, &mockSearch{}, nil)
	_, err := svc.CreateTask(CreateTaskInput{ListID: "l", Title: "Fail"})
	if err == nil {
		t.Error("expected error from repo")
	}
}

func TestCreateTask_NextNumIDError(t *testing.T) {
	repo := &errTaskRepo{mockTaskRepo: *newMockTaskRepo()}
	svc := NewTaskService(repo, &mockUpdateRepo{}, &mockSearch{}, nil)
	_, err := svc.CreateTask(CreateTaskInput{ListID: "l", Title: "Fail"})
	if err == nil {
		t.Error("expected error from NextNumID")
	}
}

func TestUpdateTask_NotFound(t *testing.T) {
	svc := newTestSvc()
	title := "new"
	_, err := svc.UpdateTask(UpdateTaskInput{ID: "nonexistent", Title: &title})
	if err == nil {
		t.Error("expected error for nonexistent task")
	}
}

func TestUpdateTask_EmptyTitle(t *testing.T) {
	svc := newTestSvc()
	task, _ := svc.CreateTask(CreateTaskInput{ListID: "l", Title: "Original"})
	empty := ""
	_, err := svc.UpdateTask(UpdateTaskInput{ID: task.ID, Title: &empty})
	if err == nil {
		t.Error("expected error for empty title")
	}
}

func TestUpdateTask_AllFields(t *testing.T) {
	svc := newTestSvc()
	task, _ := svc.CreateTask(CreateTaskInput{ListID: "l", Title: "Original"})

	newTitle := "New Title"
	newDesc := "New Desc"
	newStatus := domain.StatusDone
	newPri := domain.PriorityUrgent
	now := time.Now()
	newTags := []string{"tag1", "tag2"}

	updated, err := svc.UpdateTask(UpdateTaskInput{
		ID:          task.ID,
		Title:       &newTitle,
		Description: &newDesc,
		Status:      &newStatus,
		Priority:    &newPri,
		DueDate:     &now,
		Tags:        newTags,
	})
	if err != nil {
		t.Fatal(err)
	}
	if updated.Title != newTitle {
		t.Errorf("title = %q", updated.Title)
	}
	if updated.Description != newDesc {
		t.Errorf("description = %q", updated.Description)
	}
	if updated.Status != newStatus {
		t.Errorf("status = %q", updated.Status)
	}
	if updated.Priority != newPri {
		t.Errorf("priority = %d", updated.Priority)
	}
	if updated.DueDate == nil {
		t.Error("due date should be set")
	}
	if len(updated.Tags) != 2 {
		t.Errorf("tags = %v", updated.Tags)
	}
}

func TestDeleteTask_NotFound(t *testing.T) {
	// Deleting a nonexistent task — mockTaskRepo.Delete is a no-op (returns nil).
	// But errTaskRepo.Delete returns error.
	errSvc := NewTaskService(&errTaskRepo{mockTaskRepo: *newMockTaskRepo()}, &mockUpdateRepo{}, &mockSearch{}, nil)
	err := errSvc.DeleteTask("nonexistent")
	if err == nil {
		t.Error("expected error from repo")
	}
}

func TestGetTask_NotFound(t *testing.T) {
	svc := newTestSvc()
	_, err := svc.GetTask("nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent task")
	}
}

func TestGetTaskByNum_NotFound(t *testing.T) {
	svc := newTestSvc()
	_, err := svc.GetTaskByNum(999)
	if err == nil {
		t.Error("expected error for nonexistent num_id")
	}
}

func TestAddUpdate_TaskNotFound(t *testing.T) {
	svc := newTestSvc()
	_, err := svc.AddUpdate("nonexistent", "comment")
	if err == nil {
		t.Error("expected error for nonexistent task")
	}
}

func TestGetUpdates_Empty(t *testing.T) {
	svc := newTestSvc()
	task, _ := svc.CreateTask(CreateTaskInput{ListID: "l", Title: "T"})
	updates, err := svc.GetUpdates(task.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(updates) != 0 {
		t.Errorf("expected 0 updates, got %d", len(updates))
	}
}

func TestListTasks_WithSearch(t *testing.T) {
	repo := newMockTaskRepo()
	task := &domain.Task{ID: "abc-123", NumID: 1, Title: "Findable", ListID: "l"}
	repo.tasks[task.ID] = task

	svc := NewTaskService(repo, &mockUpdateRepo{}, &searchWithResults{ids: []string{"abc-123"}}, nil)
	tasks, err := svc.ListTasks(TaskFilter{Search: "Findable"})
	if err != nil {
		t.Fatal(err)
	}
	if len(tasks) != 1 {
		t.Errorf("got %d tasks, want 1", len(tasks))
	}
}

func TestListTasks_SearchNoResults(t *testing.T) {
	svc := NewTaskService(newMockTaskRepo(), &mockUpdateRepo{}, &searchWithResults{ids: nil}, nil)
	tasks, err := svc.ListTasks(TaskFilter{Search: "nothing"})
	if err != nil {
		t.Fatal(err)
	}
	if tasks != nil {
		t.Errorf("expected nil, got %v", tasks)
	}
}

func TestListTasks_SearchError(t *testing.T) {
	svc := NewTaskService(newMockTaskRepo(), &mockUpdateRepo{}, &errSearch{}, nil)
	_, err := svc.ListTasks(TaskFilter{Search: "fail"})
	if err == nil {
		t.Error("expected search error")
	}
}

func TestListTasks_SearchDeletedTask(t *testing.T) {
	// Search returns ID but task was deleted from repo
	repo := newMockTaskRepo()
	svc := NewTaskService(repo, &mockUpdateRepo{}, &searchWithResults{ids: []string{"deleted-id"}}, nil)
	tasks, err := svc.ListTasks(TaskFilter{Search: "ghost"})
	if err != nil {
		t.Fatal(err)
	}
	if len(tasks) != 0 {
		t.Errorf("expected 0 tasks for deleted IDs, got %d", len(tasks))
	}
}

func TestAddDependency(t *testing.T) {
	svc := newTestSvc()
	t1, _ := svc.CreateTask(CreateTaskInput{ListID: "l", Title: "Parent"})
	t2, _ := svc.CreateTask(CreateTaskInput{ListID: "l", Title: "Dep"})

	err := svc.AddDependency(t1.ID, t2.ID)
	if err != nil {
		t.Fatal(err)
	}
	got, _ := svc.GetTask(t1.ID)
	if len(got.DependsOn) != 1 || got.DependsOn[0] != t2.ID {
		t.Errorf("depends_on = %v", got.DependsOn)
	}

	// Adding same dependency again should be a no-op.
	err = svc.AddDependency(t1.ID, t2.ID)
	if err != nil {
		t.Fatal(err)
	}
	got2, _ := svc.GetTask(t1.ID)
	if len(got2.DependsOn) != 1 {
		t.Errorf("duplicate dependency: depends_on = %v", got2.DependsOn)
	}
}

func TestAddDependency_TaskNotFound(t *testing.T) {
	svc := newTestSvc()
	err := svc.AddDependency("nonexistent", "dep")
	if err == nil {
		t.Error("expected error for nonexistent task")
	}
}

func TestNewTaskService_NilLogger(t *testing.T) {
	svc := NewTaskService(newMockTaskRepo(), &mockUpdateRepo{}, &mockSearch{}, nil)
	if svc.log == nil {
		t.Error("logger should be nop, not nil")
	}
}
