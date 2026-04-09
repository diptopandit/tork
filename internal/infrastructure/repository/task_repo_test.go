package repository

import (
	"database/sql"
	"testing"
	"time"

	_ "modernc.org/sqlite"

	"github.com/diptopandit/tork/internal/domain"
	"github.com/diptopandit/tork/internal/infrastructure/db"
)

func openTestDB(t *testing.T) *sql.DB {
	t.Helper()
	conn, err := sql.Open("sqlite", ":memory:?_foreign_keys=on")
	if err != nil {
		t.Fatal(err)
	}
	// Ensure foreign keys are enabled (some drivers ignore DSN params).
	if _, err := conn.Exec("PRAGMA foreign_keys = ON"); err != nil {
		t.Fatal(err)
	}
	if err := db.Migrate(conn); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { conn.Close() })
	return conn
}

func seedList(t *testing.T, conn *sql.DB) string {
	t.Helper()
	listRepo := NewListRepo(conn)
	l := &domain.TaskList{
		ID:        "test-list-1",
		Name:      "Test",
		Schema:    map[string]domain.FieldDefinition{},
		CreatedAt: time.Now(),
	}
	if err := listRepo.Create(l); err != nil {
		t.Fatal(err)
	}
	return l.ID
}

func TestTaskRepo_CreateAndGet(t *testing.T) {
	conn := openTestDB(t)
	listID := seedList(t, conn)
	repo := NewTaskRepo(conn)

	now := time.Now()
	task := &domain.Task{
		ID:        "task-1",
		ListID:    listID,
		Title:     "Test task",
		Status:    domain.StatusTodo,
		Priority:  domain.PriorityMedium,
		Tags:      []string{"go", "test"},
		DependsOn: []string{},
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := repo.Create(task); err != nil {
		t.Fatal(err)
	}

	got, err := repo.GetByID("task-1")
	if err != nil {
		t.Fatal(err)
	}
	if got.Title != task.Title {
		t.Errorf("title = %q, want %q", got.Title, task.Title)
	}
	if len(got.Tags) != 2 {
		t.Errorf("tags len = %d, want 2", len(got.Tags))
	}
}

func TestTaskRepo_Update(t *testing.T) {
	conn := openTestDB(t)
	listID := seedList(t, conn)
	repo := NewTaskRepo(conn)

	now := time.Now()
	task := &domain.Task{
		ID:        "task-2",
		ListID:    listID,
		Title:     "Original",
		Status:    domain.StatusTodo,
		Priority:  domain.PriorityMedium,
		DependsOn: []string{},
		CreatedAt: now,
		UpdatedAt: now,
	}
	repo.Create(task)

	task.Title = "Updated"
	task.Status = domain.StatusDone
	if err := repo.Update(task); err != nil {
		t.Fatal(err)
	}

	got, _ := repo.GetByID("task-2")
	if got.Title != "Updated" {
		t.Errorf("title = %q", got.Title)
	}
	if got.Status != domain.StatusDone {
		t.Errorf("status = %q", got.Status)
	}
}

func TestTaskRepo_Delete(t *testing.T) {
	conn := openTestDB(t)
	listID := seedList(t, conn)
	repo := NewTaskRepo(conn)

	now := time.Now()
	task := &domain.Task{
		ID:        "task-3",
		ListID:    listID,
		Title:     "Delete me",
		Status:    domain.StatusTodo,
		Priority:  domain.PriorityMedium,
		DependsOn: []string{},
		CreatedAt: now,
		UpdatedAt: now,
	}
	repo.Create(task)
	repo.Delete("task-3")

	_, err := repo.GetByID("task-3")
	if err == nil {
		t.Error("expected error after delete")
	}
}

func TestTaskRepo_List(t *testing.T) {
	conn := openTestDB(t)
	listID := seedList(t, conn)
	repo := NewTaskRepo(conn)

	now := time.Now()
	for i, title := range []string{"T1", "T2", "T3"} {
		repo.Create(&domain.Task{
			ID:        "t" + string(rune('1'+i)),
			NumID:     i + 1,
			ListID:    listID,
			Title:     title,
			Status:    domain.StatusTodo,
			Priority:  domain.PriorityMedium,
			DependsOn: []string{},
			CreatedAt: now,
			UpdatedAt: now,
		})
	}

	tasks, err := repo.List(domain.TaskFilter{})
	if err != nil {
		t.Fatal(err)
	}
	if len(tasks) != 3 {
		t.Errorf("got %d tasks, want 3", len(tasks))
	}
}

func TestTaskRepo_GetByNumID(t *testing.T) {
	conn := openTestDB(t)
	listID := seedList(t, conn)
	repo := NewTaskRepo(conn)

	now := time.Now()
	task := &domain.Task{
		ID:        "task-num",
		NumID:     42,
		ListID:    listID,
		Title:     "Numbered task",
		Status:    domain.StatusTodo,
		Priority:  domain.PriorityMedium,
		DependsOn: []string{},
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := repo.Create(task); err != nil {
		t.Fatal(err)
	}

	got, err := repo.GetByNumID(42)
	if err != nil {
		t.Fatal(err)
	}
	if got.Title != "Numbered task" {
		t.Errorf("title = %q, want %q", got.Title, "Numbered task")
	}
}

func TestTaskRepo_NextNumID(t *testing.T) {
	conn := openTestDB(t)
	listID := seedList(t, conn)
	repo := NewTaskRepo(conn)

	now := time.Now()
	repo.Create(&domain.Task{
		ID: "t1", NumID: 5, ListID: listID, Title: "T1",
		Status: domain.StatusTodo, Priority: domain.PriorityMedium,
		DependsOn: []string{}, CreatedAt: now, UpdatedAt: now,
	})

	next, err := repo.NextNumID()
	if err != nil {
		t.Fatal(err)
	}
	if next != 6 {
		t.Errorf("next num_id = %d, want 6", next)
	}
}

func TestListRepo_CreateAndGet(t *testing.T) {
	conn := openTestDB(t)
	repo := NewListRepo(conn)

	l := &domain.TaskList{
		ID:        "list-1",
		Name:      "Work",
		Schema:    map[string]domain.FieldDefinition{},
		CreatedAt: time.Now(),
	}
	if err := repo.Create(l); err != nil {
		t.Fatal(err)
	}

	got, err := repo.GetByID("list-1")
	if err != nil {
		t.Fatal(err)
	}
	if got.Name != "Work" {
		t.Errorf("name = %q, want %q", got.Name, "Work")
	}
}

func TestListRepo_Update(t *testing.T) {
	conn := openTestDB(t)
	repo := NewListRepo(conn)

	l := &domain.TaskList{
		ID:        "list-upd",
		Name:      "Original",
		Schema:    map[string]domain.FieldDefinition{},
		CreatedAt: time.Now(),
	}
	repo.Create(l)

	l.Name = "Renamed"
	if err := repo.Update(l); err != nil {
		t.Fatal(err)
	}

	got, _ := repo.GetByID("list-upd")
	if got.Name != "Renamed" {
		t.Errorf("name = %q, want %q", got.Name, "Renamed")
	}
}

func TestListRepo_GetAll(t *testing.T) {
	conn := openTestDB(t)
	repo := NewListRepo(conn)

	for _, name := range []string{"A", "B", "C"} {
		repo.Create(&domain.TaskList{
			ID: "l-" + name, Name: name,
			Schema: map[string]domain.FieldDefinition{}, CreatedAt: time.Now(),
		})
	}

	lists, err := repo.GetAll()
	if err != nil {
		t.Fatal(err)
	}
	if len(lists) != 3 {
		t.Errorf("got %d lists, want 3", len(lists))
	}
}

func TestListRepo_DeleteCascade(t *testing.T) {
	conn := openTestDB(t)
	listRepo := NewListRepo(conn)
	taskRepo := NewTaskRepo(conn)

	l := &domain.TaskList{
		ID: "list-del", Name: "Deletable",
		Schema: map[string]domain.FieldDefinition{}, CreatedAt: time.Now(),
	}
	listRepo.Create(l)

	now := time.Now()
	taskRepo.Create(&domain.Task{
		ID: "task-in-del", ListID: "list-del", Title: "Orphan soon",
		Status: domain.StatusTodo, Priority: domain.PriorityMedium,
		DependsOn: []string{}, CreatedAt: now, UpdatedAt: now,
	})

	if err := listRepo.Delete("list-del"); err != nil {
		t.Fatal(err)
	}

	// Task should be cascade-deleted
	_, err := taskRepo.GetByID("task-in-del")
	if err == nil {
		t.Error("expected error: task should be cascade-deleted with list")
	}
}

func TestUpdateRepo_AddAndList(t *testing.T) {
	conn := openTestDB(t)
	listID := seedList(t, conn)
	taskRepo := NewTaskRepo(conn)
	updateRepo := NewUpdateRepo(conn)

	now := time.Now()
	taskRepo.Create(&domain.Task{
		ID: "task-u", ListID: listID, Title: "Has updates",
		Status: domain.StatusTodo, Priority: domain.PriorityMedium,
		DependsOn: []string{}, CreatedAt: now, UpdatedAt: now,
	})

	updateRepo.AddUpdate(&domain.Update{
		ID: "u1", TaskID: "task-u", Body: "First", CreatedAt: now,
	})
	updateRepo.AddUpdate(&domain.Update{
		ID: "u2", TaskID: "task-u", Body: "Second", CreatedAt: now,
	})

	updates, err := updateRepo.ListByTaskID("task-u")
	if err != nil {
		t.Fatal(err)
	}
	if len(updates) != 2 {
		t.Errorf("got %d updates, want 2", len(updates))
	}
}
