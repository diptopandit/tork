package repository

import (
	"database/sql"
	"os"
	"testing"
	"time"

	_ "github.com/go-sql-driver/mysql"

	"github.com/diptopandit/tork/internal/domain"
	"github.com/diptopandit/tork/internal/infrastructure/db"
)

// mysqlDSN reads the test MySQL DSN from the TORK_MYSQL_DSN env var.
// Default: root:tork@tcp(127.0.0.1:3306)/tork_test
func mysqlDSN(t *testing.T) string {
	t.Helper()
	dsn := os.Getenv("TORK_MYSQL_DSN")
	if dsn == "" {
		dsn = "root:tork@tcp(127.0.0.1:3306)/tork_test"
	}
	return dsn
}

func openMySQLTestDB(t *testing.T) *sql.DB {
	t.Helper()
	dsn := mysqlDSN(t)
	conn, err := db.OpenMySQL(dsn)
	if err != nil {
		t.Skipf("MySQL not available: %v", err)
	}
	// Clean up tables in reverse dependency order.
	for _, tbl := range []string{"task_updates", "tasks", "list_members", "task_lists", "users"} {
		conn.Exec("DELETE FROM " + tbl)
	}
	t.Cleanup(func() { conn.Close() })
	return conn
}

func seedMySQLUser(t *testing.T, conn *sql.DB, id, username string) {
	t.Helper()
	repo := NewUserRepoMySQL(conn)
	if err := repo.EnsureUser(&domain.User{
		ID:        id,
		Username:  username,
		CreatedAt: time.Now().UTC(),
	}); err != nil {
		t.Fatal(err)
	}
}

func seedMySQLList(t *testing.T, conn *sql.DB, userID string) string {
	t.Helper()
	repo := NewListRepoMySQL(conn, userID)
	l := &domain.TaskList{
		ID:        "test-list-mysql",
		Name:      "Test MySQL List",
		Schema:    map[string]domain.FieldDefinition{},
		CreatedAt: time.Now().UTC(),
	}
	if err := repo.Create(l); err != nil {
		t.Fatal(err)
	}
	return l.ID
}

// ---- User repo tests --------------------------------------------------------

func TestMySQLUserRepo_EnsureUser(t *testing.T) {
	conn := openMySQLTestDB(t)
	repo := NewUserRepoMySQL(conn)

	u := &domain.User{ID: "u1", Username: "alice", CreatedAt: time.Now().UTC()}
	if err := repo.EnsureUser(u); err != nil {
		t.Fatal(err)
	}

	// Ensure again (should update, not fail).
	u.Username = "alice-updated"
	if err := repo.EnsureUser(u); err != nil {
		t.Fatal(err)
	}

	got, err := repo.GetByID("u1")
	if err != nil {
		t.Fatal(err)
	}
	if got.Username != "alice-updated" {
		t.Errorf("username = %q, want alice-updated", got.Username)
	}
}

func TestMySQLUserRepo_GetByUsername(t *testing.T) {
	conn := openMySQLTestDB(t)
	repo := NewUserRepoMySQL(conn)

	repo.EnsureUser(&domain.User{ID: "u2", Username: "bob", CreatedAt: time.Now().UTC()})

	got, err := repo.GetByUsername("bob")
	if err != nil {
		t.Fatal(err)
	}
	if got.ID != "u2" {
		t.Errorf("id = %q, want u2", got.ID)
	}
}

// ---- Task repo tests --------------------------------------------------------

func TestMySQLTaskRepo_CRUD(t *testing.T) {
	conn := openMySQLTestDB(t)
	userID := "user-task-crud"
	seedMySQLUser(t, conn, userID, "task-user")
	listID := seedMySQLList(t, conn, userID)
	repo := NewTaskRepoMySQL(conn, userID)

	now := time.Now().UTC()
	task := &domain.Task{
		ID:        "task-m1",
		NumID:     1,
		ListID:    listID,
		Title:     "MySQL task",
		Status:    domain.StatusTodo,
		Priority:  domain.PriorityHigh,
		Tags:      []string{"mysql", "test"},
		DependsOn: []string{},
		CreatedAt: now,
		UpdatedAt: now,
	}

	// Create
	if err := repo.Create(task); err != nil {
		t.Fatal(err)
	}

	// GetByID
	got, err := repo.GetByID("task-m1")
	if err != nil {
		t.Fatal(err)
	}
	if got.Title != "MySQL task" {
		t.Errorf("title = %q", got.Title)
	}
	if len(got.Tags) != 2 {
		t.Errorf("tags len = %d, want 2", len(got.Tags))
	}

	// GetByNumID
	got2, err := repo.GetByNumID(1)
	if err != nil {
		t.Fatal(err)
	}
	if got2.ID != "task-m1" {
		t.Errorf("id = %q", got2.ID)
	}

	// Update
	task.Title = "Updated MySQL task"
	task.UpdatedAt = time.Now().UTC()
	if err := repo.Update(task); err != nil {
		t.Fatal(err)
	}
	got3, _ := repo.GetByID("task-m1")
	if got3.Title != "Updated MySQL task" {
		t.Errorf("title after update = %q", got3.Title)
	}

	// NextNumID
	next, err := repo.NextNumID()
	if err != nil {
		t.Fatal(err)
	}
	if next != 2 {
		t.Errorf("next num_id = %d, want 2", next)
	}

	// Delete
	if err := repo.Delete("task-m1"); err != nil {
		t.Fatal(err)
	}
	_, err = repo.GetByID("task-m1")
	if err == nil {
		t.Error("expected error after delete")
	}
}

func TestMySQLTaskRepo_List_UserScoped(t *testing.T) {
	conn := openMySQLTestDB(t)

	// Two users
	seedMySQLUser(t, conn, "alice-id", "alice")
	seedMySQLUser(t, conn, "bob-id", "bob")

	aliceListRepo := NewListRepoMySQL(conn, "alice-id")
	bobListRepo := NewListRepoMySQL(conn, "bob-id")

	now := time.Now().UTC()

	// Alice creates a list
	aliceListRepo.Create(&domain.TaskList{
		ID: "alice-list", Name: "Alice's", Schema: map[string]domain.FieldDefinition{},
		OwnerID: "alice-id", CreatedAt: now,
	})

	// Bob creates a list
	bobListRepo.Create(&domain.TaskList{
		ID: "bob-list", Name: "Bob's", Schema: map[string]domain.FieldDefinition{},
		OwnerID: "bob-id", CreatedAt: now,
	})

	// Alice adds tasks to her list
	aliceTaskRepo := NewTaskRepoMySQL(conn, "alice-id")
	aliceTaskRepo.Create(&domain.Task{
		ID: "a-t1", NumID: 10, ListID: "alice-list", Title: "Alice task",
		Status: domain.StatusTodo, Priority: domain.PriorityMedium,
		DependsOn: []string{}, CreatedAt: now, UpdatedAt: now,
	})

	// Bob adds tasks to his list
	bobTaskRepo := NewTaskRepoMySQL(conn, "bob-id")
	bobTaskRepo.Create(&domain.Task{
		ID: "b-t1", NumID: 20, ListID: "bob-list", Title: "Bob task",
		Status: domain.StatusTodo, Priority: domain.PriorityMedium,
		DependsOn: []string{}, CreatedAt: now, UpdatedAt: now,
	})

	// Alice should only see her task
	aliceTasks, err := aliceTaskRepo.List(domain.TaskFilter{})
	if err != nil {
		t.Fatal(err)
	}
	if len(aliceTasks) != 1 {
		t.Errorf("alice got %d tasks, want 1", len(aliceTasks))
	}

	// Bob should only see his task
	bobTasks, err := bobTaskRepo.List(domain.TaskFilter{})
	if err != nil {
		t.Fatal(err)
	}
	if len(bobTasks) != 1 {
		t.Errorf("bob got %d tasks, want 1", len(bobTasks))
	}
}

// ---- List repo tests --------------------------------------------------------

func TestMySQLListRepo_CRUD(t *testing.T) {
	conn := openMySQLTestDB(t)
	userID := "user-list-crud"
	seedMySQLUser(t, conn, userID, "list-user")
	repo := NewListRepoMySQL(conn, userID)

	now := time.Now().UTC()
	l := &domain.TaskList{
		ID:        "list-m1",
		Name:      "MySQL List",
		Schema:    map[string]domain.FieldDefinition{},
		CreatedAt: now,
	}

	// Create
	if err := repo.Create(l); err != nil {
		t.Fatal(err)
	}

	// GetByID
	got, err := repo.GetByID("list-m1")
	if err != nil {
		t.Fatal(err)
	}
	if got.Name != "MySQL List" {
		t.Errorf("name = %q", got.Name)
	}
	if got.OwnerID != userID {
		t.Errorf("owner_id = %q, want %q", got.OwnerID, userID)
	}
	if got.Visibility != "private" {
		t.Errorf("visibility = %q, want private", got.Visibility)
	}

	// Update
	l.Name = "Renamed"
	if err := repo.Update(l); err != nil {
		t.Fatal(err)
	}
	got2, _ := repo.GetByID("list-m1")
	if got2.Name != "Renamed" {
		t.Errorf("name after update = %q", got2.Name)
	}

	// GetAll
	lists, err := repo.GetAll()
	if err != nil {
		t.Fatal(err)
	}
	if len(lists) != 1 {
		t.Errorf("got %d lists, want 1", len(lists))
	}

	// Delete
	if err := repo.Delete("list-m1"); err != nil {
		t.Fatal(err)
	}
	_, err = repo.GetByID("list-m1")
	if err == nil {
		t.Error("expected error after delete")
	}
}

func TestMySQLListRepo_SharedAccess(t *testing.T) {
	conn := openMySQLTestDB(t)

	seedMySQLUser(t, conn, "owner-id", "owner")
	seedMySQLUser(t, conn, "member-id", "member")
	seedMySQLUser(t, conn, "outsider-id", "outsider")

	ownerRepo := NewListRepoMySQL(conn, "owner-id")
	memberRepo := NewListRepoMySQL(conn, "member-id")
	outsiderRepo := NewListRepoMySQL(conn, "outsider-id")

	now := time.Now().UTC()

	// Owner creates a shared list.
	ownerRepo.Create(&domain.TaskList{
		ID: "shared-list", Name: "Team Tasks",
		Schema:     map[string]domain.FieldDefinition{},
		Visibility: "shared",
		CreatedAt:  now,
	})

	// Owner adds member.
	if err := ownerRepo.AddMember("shared-list", "member-id", "editor"); err != nil {
		t.Fatal(err)
	}

	// Member should see the list.
	memberLists, err := memberRepo.GetAll()
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, l := range memberLists {
		if l.ID == "shared-list" {
			found = true
		}
	}
	if !found {
		t.Error("member should see shared list")
	}

	// Outsider should NOT see it.
	outsiderLists, err := outsiderRepo.GetAll()
	if err != nil {
		t.Fatal(err)
	}
	for _, l := range outsiderLists {
		if l.ID == "shared-list" {
			t.Error("outsider should not see shared list")
		}
	}

	// Member can NOT delete (only owner can).
	if err := memberRepo.Delete("shared-list"); err == nil {
		t.Error("member should not be able to delete")
	}

	// Outsider can NOT update.
	l, _ := ownerRepo.GetByID("shared-list")
	l.Name = "Hacked"
	if err := outsiderRepo.Update(l); err == nil {
		t.Error("outsider should not be able to update")
	}

	// Verify member roles.
	members, err := ownerRepo.ListMembers("shared-list")
	if err != nil {
		t.Fatal(err)
	}
	if len(members) != 1 {
		t.Errorf("got %d members, want 1", len(members))
	}
	if members[0].Role != "editor" {
		t.Errorf("role = %q, want editor", members[0].Role)
	}

	// Remove member.
	if err := ownerRepo.RemoveMember("shared-list", "member-id"); err != nil {
		t.Fatal(err)
	}
	members2, _ := ownerRepo.ListMembers("shared-list")
	if len(members2) != 0 {
		t.Errorf("got %d members after remove, want 0", len(members2))
	}
}

func TestMySQLListRepo_NonOwnerCannotAddMember(t *testing.T) {
	conn := openMySQLTestDB(t)

	seedMySQLUser(t, conn, "own-id", "owner2")
	seedMySQLUser(t, conn, "nonown-id", "nonowner")

	ownerRepo := NewListRepoMySQL(conn, "own-id")
	nonOwnerRepo := NewListRepoMySQL(conn, "nonown-id")

	now := time.Now().UTC()
	ownerRepo.Create(&domain.TaskList{
		ID: "restricted-list", Name: "Restricted",
		Schema: map[string]domain.FieldDefinition{}, CreatedAt: now,
	})

	// Non-owner tries to add member — should fail.
	if err := nonOwnerRepo.AddMember("restricted-list", "nonown-id", "editor"); err == nil {
		t.Error("non-owner should not be able to add members")
	}
}

// ---- Update repo tests ------------------------------------------------------

func TestMySQLUpdateRepo_AddAndList(t *testing.T) {
	conn := openMySQLTestDB(t)
	userID := "user-upd"
	seedMySQLUser(t, conn, userID, "upd-user")
	listID := seedMySQLList(t, conn, userID)

	taskRepo := NewTaskRepoMySQL(conn, userID)
	updateRepo := NewUpdateRepoMySQL(conn)

	now := time.Now().UTC()
	taskRepo.Create(&domain.Task{
		ID: "task-upd", NumID: 100, ListID: listID, Title: "Has updates",
		Status: domain.StatusTodo, Priority: domain.PriorityMedium,
		DependsOn: []string{}, CreatedAt: now, UpdatedAt: now,
	})

	updateRepo.AddUpdate(&domain.Update{
		ID: "mu1", TaskID: "task-upd", Body: "First MySQL update", CreatedAt: now,
	})
	updateRepo.AddUpdate(&domain.Update{
		ID: "mu2", TaskID: "task-upd", Body: "Second", CreatedAt: now,
	})

	updates, err := updateRepo.ListByTaskID("task-upd")
	if err != nil {
		t.Fatal(err)
	}
	if len(updates) != 2 {
		t.Errorf("got %d updates, want 2", len(updates))
	}
}

// ---- Cascade delete ----------------------------------------------------------

func TestMySQLListRepo_DeleteCascade(t *testing.T) {
	conn := openMySQLTestDB(t)
	userID := "user-cascade"
	seedMySQLUser(t, conn, userID, "cascade-user")

	listRepo := NewListRepoMySQL(conn, userID)
	taskRepo := NewTaskRepoMySQL(conn, userID)

	now := time.Now().UTC()
	listRepo.Create(&domain.TaskList{
		ID: "del-list", Name: "Will delete",
		Schema: map[string]domain.FieldDefinition{}, CreatedAt: now,
	})

	taskRepo.Create(&domain.Task{
		ID: "cascade-task", NumID: 200, ListID: "del-list", Title: "Orphan",
		Status: domain.StatusTodo, Priority: domain.PriorityMedium,
		DependsOn: []string{}, CreatedAt: now, UpdatedAt: now,
	})

	if err := listRepo.Delete("del-list"); err != nil {
		t.Fatal(err)
	}

	_, err := taskRepo.GetByID("cascade-task")
	if err == nil {
		t.Error("expected task to be cascade-deleted with list")
	}
}
