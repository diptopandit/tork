package repository

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/diptopandit/tork/internal/domain"
)

// TaskRepoMySQL implements domain.TaskRepository backed by MySQL.
type TaskRepoMySQL struct {
	db     *sql.DB
	userID string // scoping: only operate on tasks in lists the user can access
}

// NewTaskRepoMySQL creates a new TaskRepoMySQL.
func NewTaskRepoMySQL(db *sql.DB, userID string) *TaskRepoMySQL {
	return &TaskRepoMySQL{db: db, userID: userID}
}

// Create inserts a new task row.
func (r *TaskRepoMySQL) Create(task *domain.Task) error {
	tags, err := json.Marshal(task.Tags)
	if err != nil {
		return fmt.Errorf("task create: marshal tags: %w", err)
	}
	cf, err := json.Marshal(task.CustomFields)
	if err != nil {
		return fmt.Errorf("task create: marshal custom_fields: %w", err)
	}
	dep, err := json.Marshal(task.DependsOn)
	if err != nil {
		return fmt.Errorf("task create: marshal depends_on: %w", err)
	}

	_, err = r.db.Exec(`
		INSERT INTO tasks
			(id, num_id, list_id, title, description, status, priority, due_date,
			 tags, custom_fields, parent_id, depends_on, created_at, updated_at)
		VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		task.ID, task.NumID, task.ListID, task.Title, task.Description,
		string(task.Status), int(task.Priority),
		nullTime(task.DueDate),
		string(tags), string(cf),
		nullString(task.ParentID),
		string(dep),
		task.CreatedAt.UTC(), task.UpdatedAt.UTC(),
	)
	if err != nil {
		return fmt.Errorf("task create: %w", err)
	}
	return nil
}

// Update replaces an existing task row.
func (r *TaskRepoMySQL) Update(task *domain.Task) error {
	tags, err := json.Marshal(task.Tags)
	if err != nil {
		return fmt.Errorf("task update: marshal tags: %w", err)
	}
	cf, err := json.Marshal(task.CustomFields)
	if err != nil {
		return fmt.Errorf("task update: marshal custom_fields: %w", err)
	}
	dep, err := json.Marshal(task.DependsOn)
	if err != nil {
		return fmt.Errorf("task update: marshal depends_on: %w", err)
	}

	res, err := r.db.Exec(`
		UPDATE tasks SET
			list_id=?, title=?, description=?, status=?, priority=?,
			due_date=?, tags=?, custom_fields=?, parent_id=?,
			depends_on=?, updated_at=?
		WHERE id=?`,
		task.ListID, task.Title, task.Description,
		string(task.Status), int(task.Priority),
		nullTime(task.DueDate),
		string(tags), string(cf),
		nullString(task.ParentID),
		string(dep),
		task.UpdatedAt.UTC(),
		task.ID,
	)
	if err != nil {
		return fmt.Errorf("task update: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("task update: not found: %s", task.ID)
	}
	return nil
}

// Delete removes a task by ID.
func (r *TaskRepoMySQL) Delete(taskID string) error {
	_, err := r.db.Exec("DELETE FROM tasks WHERE id=?", taskID)
	if err != nil {
		return fmt.Errorf("task delete: %w", err)
	}
	return nil
}

// GetByID fetches a single task.
func (r *TaskRepoMySQL) GetByID(id string) (*domain.Task, error) {
	row := r.db.QueryRow(`
		SELECT id, num_id, list_id, title, description, status, priority, due_date,
		       tags, custom_fields, parent_id, depends_on, created_at, updated_at
		FROM tasks WHERE id=?`, id)
	t, err := scanTask(row)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("task not found: %s", id)
	}
	if err != nil {
		return nil, fmt.Errorf("task get: %w", err)
	}
	return t, nil
}

// GetByNumID fetches a single task by its numeric short ID.
func (r *TaskRepoMySQL) GetByNumID(numID int) (*domain.Task, error) {
	row := r.db.QueryRow(`
		SELECT id, num_id, list_id, title, description, status, priority, due_date,
		       tags, custom_fields, parent_id, depends_on, created_at, updated_at
		FROM tasks WHERE num_id=?`, numID)
	t, err := scanTask(row)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("task not found: #%d", numID)
	}
	if err != nil {
		return nil, fmt.Errorf("task get by num_id: %w", err)
	}
	return t, nil
}

// NextNumID returns the next available numeric ID.
func (r *TaskRepoMySQL) NextNumID() (int, error) {
	var maxID sql.NullInt64
	err := r.db.QueryRow("SELECT MAX(num_id) FROM tasks").Scan(&maxID)
	if err != nil {
		return 1, nil
	}
	if !maxID.Valid {
		return 1, nil
	}
	return int(maxID.Int64) + 1, nil
}

// List returns tasks matching the given filter, scoped to accessible lists.
func (r *TaskRepoMySQL) List(filter domain.TaskFilter) ([]domain.Task, error) {
	conds := []string{"1=1"}
	args := []interface{}{}

	// Scope to lists the user can access (owner or member).
	if r.userID != "" && len(filter.ListIDs) == 0 {
		conds = append(conds, `list_id IN (
			SELECT id FROM task_lists WHERE owner_id=?
			UNION
			SELECT list_id FROM list_members WHERE user_id=?
		)`)
		args = append(args, r.userID, r.userID)
	}

	if len(filter.ListIDs) > 0 {
		ph := placeholders(len(filter.ListIDs))
		conds = append(conds, "list_id IN ("+ph+")")
		for _, id := range filter.ListIDs {
			args = append(args, id)
		}
	}
	if len(filter.Statuses) > 0 {
		ph := placeholders(len(filter.Statuses))
		conds = append(conds, "status IN ("+ph+")")
		for _, s := range filter.Statuses {
			args = append(args, string(s))
		}
	}
	if len(filter.Priorities) > 0 {
		ph := placeholders(len(filter.Priorities))
		conds = append(conds, "priority IN ("+ph+")")
		for _, p := range filter.Priorities {
			args = append(args, int(p))
		}
	}
	if filter.DueBefore != nil {
		conds = append(conds, "due_date < ?")
		args = append(args, filter.DueBefore.UTC())
	}

	q := "SELECT id, num_id, list_id, title, description, status, priority, due_date, " +
		"tags, custom_fields, parent_id, depends_on, created_at, updated_at " +
		"FROM tasks WHERE " + strings.Join(conds, " AND ")

	orderClause := buildOrderClause(filter.SortField, filter.SortDir)
	q += " ORDER BY " + orderClause

	rows, err := r.db.Query(q, args...)
	if err != nil {
		return nil, fmt.Errorf("task list: %w", err)
	}
	defer rows.Close()

	var tasks []domain.Task
	for rows.Next() {
		t, err := scanTask(rows)
		if err != nil {
			return nil, fmt.Errorf("task list scan: %w", err)
		}
		tasks = append(tasks, *t)
	}
	return tasks, rows.Err()
}

// ---- MySQL update repo ------------------------------------------------------

// UpdateRepoMySQL implements domain.UpdateRepository backed by MySQL.
type UpdateRepoMySQL struct {
	db *sql.DB
}

// NewUpdateRepoMySQL creates a new UpdateRepoMySQL.
func NewUpdateRepoMySQL(db *sql.DB) *UpdateRepoMySQL {
	return &UpdateRepoMySQL{db: db}
}

// AddUpdate inserts an immutable update/comment for a task.
func (r *UpdateRepoMySQL) AddUpdate(u *domain.Update) error {
	_, err := r.db.Exec(`
		INSERT INTO task_updates (id, task_id, body, created_at)
		VALUES (?,?,?,?)`,
		u.ID, u.TaskID, u.Body, u.CreatedAt.UTC(),
	)
	if err != nil {
		return fmt.Errorf("add update: %w", err)
	}
	return nil
}

// ListByTaskID returns all updates for a task, newest first.
func (r *UpdateRepoMySQL) ListByTaskID(taskID string) ([]domain.Update, error) {
	rows, err := r.db.Query(`
		SELECT id, task_id, body, created_at
		FROM task_updates
		WHERE task_id=?
		ORDER BY created_at DESC`, taskID)
	if err != nil {
		return nil, fmt.Errorf("list updates: %w", err)
	}
	defer rows.Close()

	var updates []domain.Update
	for rows.Next() {
		var u domain.Update
		if err := rows.Scan(&u.ID, &u.TaskID, &u.Body, &u.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan update: %w", err)
		}
		updates = append(updates, u)
	}
	return updates, rows.Err()
}

// ---- MySQL user repo --------------------------------------------------------

// UserRepoMySQL implements domain.UserRepository backed by MySQL.
type UserRepoMySQL struct {
	db *sql.DB
}

// NewUserRepoMySQL creates a new UserRepoMySQL.
func NewUserRepoMySQL(db *sql.DB) *UserRepoMySQL {
	return &UserRepoMySQL{db: db}
}

// EnsureUser creates the user if it doesn't exist, or updates the username.
func (r *UserRepoMySQL) EnsureUser(user *domain.User) error {
	_, err := r.db.Exec(`
		INSERT INTO users (id, username, created_at)
		VALUES (?, ?, ?)
		ON DUPLICATE KEY UPDATE username=VALUES(username)`,
		user.ID, user.Username, user.CreatedAt.UTC(),
	)
	if err != nil {
		return fmt.Errorf("ensure user: %w", err)
	}
	return nil
}

// GetByID fetches a user by ID.
func (r *UserRepoMySQL) GetByID(id string) (*domain.User, error) {
	row := r.db.QueryRow(`SELECT id, username, created_at FROM users WHERE id=?`, id)
	var u domain.User
	if err := row.Scan(&u.ID, &u.Username, &u.CreatedAt); err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("user not found: %s", id)
		}
		return nil, fmt.Errorf("user get: %w", err)
	}
	return &u, nil
}

// GetByUsername fetches a user by username.
func (r *UserRepoMySQL) GetByUsername(name string) (*domain.User, error) {
	row := r.db.QueryRow(`SELECT id, username, created_at FROM users WHERE username=?`, name)
	var u domain.User
	if err := row.Scan(&u.ID, &u.Username, &u.CreatedAt); err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("user not found: %s", name)
		}
		return nil, fmt.Errorf("user get by username: %w", err)
	}
	return &u, nil
}
