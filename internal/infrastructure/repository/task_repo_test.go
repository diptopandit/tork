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

// ---- List filter / sort / nullable field tests -----------------------------

func TestTaskRepo_List_FilterByStatus(t *testing.T) {
	conn := openTestDB(t)
	listID := seedList(t, conn)
	repo := NewTaskRepo(conn)
	now := time.Now()

	repo.Create(&domain.Task{
		ID: "t-s1", NumID: 1, ListID: listID, Title: "Todo",
		Status: domain.StatusTodo, Priority: domain.PriorityMedium,
		DependsOn: []string{}, CreatedAt: now, UpdatedAt: now,
	})
	repo.Create(&domain.Task{
		ID: "t-s2", NumID: 2, ListID: listID, Title: "Done",
		Status: domain.StatusDone, Priority: domain.PriorityMedium,
		DependsOn: []string{}, CreatedAt: now, UpdatedAt: now,
	})

	tasks, err := repo.List(domain.TaskFilter{Statuses: []domain.Status{domain.StatusTodo}})
	if err != nil {
		t.Fatal(err)
	}
	if len(tasks) != 1 || tasks[0].Title != "Todo" {
		t.Errorf("expected 1 todo task, got %d", len(tasks))
	}
}

func TestTaskRepo_List_FilterByPriority(t *testing.T) {
	conn := openTestDB(t)
	listID := seedList(t, conn)
	repo := NewTaskRepo(conn)
	now := time.Now()

	repo.Create(&domain.Task{
		ID: "t-p1", NumID: 1, ListID: listID, Title: "Low",
		Status: domain.StatusTodo, Priority: domain.PriorityLow,
		DependsOn: []string{}, CreatedAt: now, UpdatedAt: now,
	})
	repo.Create(&domain.Task{
		ID: "t-p2", NumID: 2, ListID: listID, Title: "High",
		Status: domain.StatusTodo, Priority: domain.PriorityHigh,
		DependsOn: []string{}, CreatedAt: now, UpdatedAt: now,
	})

	tasks, err := repo.List(domain.TaskFilter{Priorities: []domain.Priority{domain.PriorityHigh}})
	if err != nil {
		t.Fatal(err)
	}
	if len(tasks) != 1 || tasks[0].Title != "High" {
		t.Errorf("expected 1 high task, got %d", len(tasks))
	}
}

func TestTaskRepo_List_FilterByListID(t *testing.T) {
	conn := openTestDB(t)
	repo := NewTaskRepo(conn)
	listRepo := NewListRepo(conn)
	now := time.Now()

	l1 := &domain.TaskList{ID: "list-a", Name: "A", Schema: map[string]domain.FieldDefinition{}, CreatedAt: now}
	l2 := &domain.TaskList{ID: "list-b", Name: "B", Schema: map[string]domain.FieldDefinition{}, CreatedAt: now}
	listRepo.Create(l1)
	listRepo.Create(l2)

	repo.Create(&domain.Task{
		ID: "t-la", NumID: 1, ListID: "list-a", Title: "In A",
		Status: domain.StatusTodo, Priority: domain.PriorityMedium,
		DependsOn: []string{}, CreatedAt: now, UpdatedAt: now,
	})
	repo.Create(&domain.Task{
		ID: "t-lb", NumID: 2, ListID: "list-b", Title: "In B",
		Status: domain.StatusTodo, Priority: domain.PriorityMedium,
		DependsOn: []string{}, CreatedAt: now, UpdatedAt: now,
	})

	tasks, err := repo.List(domain.TaskFilter{ListIDs: []string{"list-a"}})
	if err != nil {
		t.Fatal(err)
	}
	if len(tasks) != 1 || tasks[0].Title != "In A" {
		t.Errorf("expected 1 task from list-a, got %d", len(tasks))
	}
}

func TestTaskRepo_List_FilterByDueBefore(t *testing.T) {
	conn := openTestDB(t)
	listID := seedList(t, conn)
	repo := NewTaskRepo(conn)
	now := time.Now()
	past := now.Add(-24 * time.Hour)
	future := now.Add(48 * time.Hour)

	repo.Create(&domain.Task{
		ID: "t-d1", NumID: 1, ListID: listID, Title: "Past due",
		Status: domain.StatusTodo, Priority: domain.PriorityMedium,
		DueDate: &past, DependsOn: []string{}, CreatedAt: now, UpdatedAt: now,
	})
	repo.Create(&domain.Task{
		ID: "t-d2", NumID: 2, ListID: listID, Title: "Future due",
		Status: domain.StatusTodo, Priority: domain.PriorityMedium,
		DueDate: &future, DependsOn: []string{}, CreatedAt: now, UpdatedAt: now,
	})

	cutoff := now
	tasks, err := repo.List(domain.TaskFilter{DueBefore: &cutoff})
	if err != nil {
		t.Fatal(err)
	}
	if len(tasks) != 1 || tasks[0].Title != "Past due" {
		t.Errorf("expected 1 past-due task, got %d", len(tasks))
	}
}

func TestTaskRepo_List_SortByPriority(t *testing.T) {
	conn := openTestDB(t)
	listID := seedList(t, conn)
	repo := NewTaskRepo(conn)
	now := time.Now()

	repo.Create(&domain.Task{
		ID: "t-sp1", NumID: 1, ListID: listID, Title: "Low",
		Status: domain.StatusTodo, Priority: domain.PriorityLow,
		DependsOn: []string{}, CreatedAt: now, UpdatedAt: now,
	})
	repo.Create(&domain.Task{
		ID: "t-sp2", NumID: 2, ListID: listID, Title: "High",
		Status: domain.StatusTodo, Priority: domain.PriorityHigh,
		DependsOn: []string{}, CreatedAt: now, UpdatedAt: now,
	})

	tasks, err := repo.List(domain.TaskFilter{SortField: domain.SortByPriority, SortDir: domain.SortAsc})
	if err != nil {
		t.Fatal(err)
	}
	if len(tasks) != 2 {
		t.Fatalf("got %d tasks", len(tasks))
	}
	if tasks[0].Priority > tasks[1].Priority {
		t.Errorf("expected ascending priority, got %d then %d", tasks[0].Priority, tasks[1].Priority)
	}
}

func TestTaskRepo_List_SortByDueDate(t *testing.T) {
	conn := openTestDB(t)
	listID := seedList(t, conn)
	repo := NewTaskRepo(conn)
	now := time.Now()
	d1 := now.Add(24 * time.Hour)
	d2 := now.Add(48 * time.Hour)

	repo.Create(&domain.Task{
		ID: "t-sd1", NumID: 1, ListID: listID, Title: "Later",
		Status: domain.StatusTodo, Priority: domain.PriorityMedium,
		DueDate: &d2, DependsOn: []string{}, CreatedAt: now, UpdatedAt: now,
	})
	repo.Create(&domain.Task{
		ID: "t-sd2", NumID: 2, ListID: listID, Title: "Sooner",
		Status: domain.StatusTodo, Priority: domain.PriorityMedium,
		DueDate: &d1, DependsOn: []string{}, CreatedAt: now, UpdatedAt: now,
	})

	tasks, err := repo.List(domain.TaskFilter{SortField: domain.SortByDueDate, SortDir: domain.SortAsc})
	if err != nil {
		t.Fatal(err)
	}
	if len(tasks) != 2 || tasks[0].Title != "Sooner" {
		t.Errorf("expected Sooner first, got %q", tasks[0].Title)
	}
}

func TestTaskRepo_List_SortByID(t *testing.T) {
	conn := openTestDB(t)
	listID := seedList(t, conn)
	repo := NewTaskRepo(conn)
	now := time.Now()

	repo.Create(&domain.Task{
		ID: "t-si1", NumID: 10, ListID: listID, Title: "Ten",
		Status: domain.StatusTodo, Priority: domain.PriorityMedium,
		DependsOn: []string{}, CreatedAt: now, UpdatedAt: now,
	})
	repo.Create(&domain.Task{
		ID: "t-si2", NumID: 5, ListID: listID, Title: "Five",
		Status: domain.StatusTodo, Priority: domain.PriorityMedium,
		DependsOn: []string{}, CreatedAt: now, UpdatedAt: now,
	})

	tasks, err := repo.List(domain.TaskFilter{SortField: domain.SortByID, SortDir: domain.SortDesc})
	if err != nil {
		t.Fatal(err)
	}
	if len(tasks) != 2 || tasks[0].NumID != 10 {
		t.Errorf("expected NumID=10 first, got %d", tasks[0].NumID)
	}
}

func TestTaskRepo_Create_WithNullableFields(t *testing.T) {
	conn := openTestDB(t)
	listID := seedList(t, conn)
	repo := NewTaskRepo(conn)
	now := time.Now()
	due := now.Add(24 * time.Hour)
	parentID := "parent-1"

	task := &domain.Task{
		ID: "t-nf", NumID: 1, ListID: listID, Title: "With nullables",
		Description: "A description",
		Status:      domain.StatusTodo, Priority: domain.PriorityHigh,
		DueDate: &due, ParentID: &parentID,
		Tags: []string{"tag1"}, CustomFields: map[string]interface{}{"key": "val"},
		DependsOn: []string{"dep1"}, CreatedAt: now, UpdatedAt: now,
	}
	if err := repo.Create(task); err != nil {
		t.Fatal(err)
	}

	got, err := repo.GetByID("t-nf")
	if err != nil {
		t.Fatal(err)
	}
	if got.DueDate == nil {
		t.Error("expected due date")
	}
	if got.ParentID == nil || *got.ParentID != parentID {
		t.Errorf("parentID = %v, want %q", got.ParentID, parentID)
	}
	if got.Description != "A description" {
		t.Errorf("description = %q", got.Description)
	}
	if len(got.CustomFields) == 0 {
		t.Error("expected custom fields")
	}
	if len(got.DependsOn) != 1 {
		t.Error("expected depends_on")
	}
}

func TestTaskRepo_Update_WithNullableFields(t *testing.T) {
	conn := openTestDB(t)
	listID := seedList(t, conn)
	repo := NewTaskRepo(conn)
	now := time.Now()

	task := &domain.Task{
		ID: "t-un", NumID: 1, ListID: listID, Title: "No nullables",
		Status: domain.StatusTodo, Priority: domain.PriorityMedium,
		DependsOn: []string{}, CreatedAt: now, UpdatedAt: now,
	}
	repo.Create(task)

	// Set nullable fields via update.
	due := now.Add(72 * time.Hour)
	pid := "p1"
	task.DueDate = &due
	task.ParentID = &pid
	task.Description = "Now with desc"
	task.CustomFields = map[string]interface{}{"x": 1.0}
	task.Tags = []string{"a", "b"}
	if err := repo.Update(task); err != nil {
		t.Fatal(err)
	}

	got, _ := repo.GetByID("t-un")
	if got.DueDate == nil {
		t.Error("expected due date after update")
	}
	if got.ParentID == nil {
		t.Error("expected parent_id after update")
	}
}

func TestTaskRepo_NextNumID_Empty(t *testing.T) {
	conn := openTestDB(t)
	// Don't seed any tasks.
	repo := NewTaskRepo(conn)

	next, err := repo.NextNumID()
	if err != nil {
		t.Fatal(err)
	}
	if next != 1 {
		t.Errorf("next num_id = %d, want 1 for empty table", next)
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
