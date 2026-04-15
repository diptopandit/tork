package application

import (
	"testing"

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
