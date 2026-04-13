package repository

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/diptopandit/tork/internal/domain"
)

// TaskRepoSQLite implements domain.TaskRepository backed by SQLite.
type TaskRepoSQLite struct {
	db *sql.DB
}

// NewTaskRepo creates a new TaskRepoSQLite.
func NewTaskRepo(db *sql.DB) *TaskRepoSQLite {
	return &TaskRepoSQLite{db: db}
}

// Create inserts a new task row.
func (r *TaskRepoSQLite) Create(task *domain.Task) error {
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
func (r *TaskRepoSQLite) Update(task *domain.Task) error {
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
func (r *TaskRepoSQLite) Delete(taskID string) error {
	_, err := r.db.Exec("DELETE FROM tasks WHERE id=?", taskID)
	if err != nil {
		return fmt.Errorf("task delete: %w", err)
	}
	return nil
}

// GetByID fetches a single task.
func (r *TaskRepoSQLite) GetByID(id string) (*domain.Task, error) {
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
func (r *TaskRepoSQLite) GetByNumID(numID int) (*domain.Task, error) {
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
func (r *TaskRepoSQLite) NextNumID() (int, error) {
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

// List returns tasks matching the given filter.
func (r *TaskRepoSQLite) List(filter domain.TaskFilter) ([]domain.Task, error) {
	conds := []string{"1=1"}
	args := []interface{}{}

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

	// Apply sort order.
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

// ---- helpers ----------------------------------------------------------------

type scanner interface {
	Scan(dest ...interface{}) error
}

func scanTask(s scanner) (*domain.Task, error) {
	var t domain.Task
	var numID sql.NullInt64
	var status string
	var priority int
	var dueDate sql.NullTime
	var tagsJSON, cfJSON, depJSON string
	var parentID sql.NullString

	err := s.Scan(
		&t.ID, &numID, &t.ListID, &t.Title, &t.Description,
		&status, &priority,
		&dueDate,
		&tagsJSON, &cfJSON,
		&parentID,
		&depJSON,
		&t.CreatedAt, &t.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	if numID.Valid {
		t.NumID = int(numID.Int64)
	}
	t.Status = domain.Status(status)
	t.Priority = domain.Priority(priority)
	if dueDate.Valid {
		tt := dueDate.Time
		t.DueDate = &tt
	}
	if parentID.Valid {
		t.ParentID = &parentID.String
	}
	if err := json.Unmarshal([]byte(tagsJSON), &t.Tags); err != nil {
		t.Tags = nil
	}
	if err := json.Unmarshal([]byte(cfJSON), &t.CustomFields); err != nil {
		t.CustomFields = nil
	}
	if err := json.Unmarshal([]byte(depJSON), &t.DependsOn); err != nil {
		t.DependsOn = nil
	}
	return &t, nil
}

func placeholders(n int) string {
	p := make([]string, n)
	for i := range p {
		p[i] = "?"
	}
	return strings.Join(p, ",")
}

func buildOrderClause(field domain.SortField, dir domain.SortDir) string {
	col := ""
	switch field {
	case domain.SortByPriority:
		col = "priority"
	case domain.SortByDueDate:
		col = "due_date"
	case domain.SortByID:
		col = "num_id"
	}

	d := "ASC"
	if dir == domain.SortDesc {
		d = "DESC"
	} else if dir == domain.SortAsc {
		d = "ASC"
	}

	if col == "" {
		// Default sort: priority DESC, due_date ASC, created_at ASC
		return "priority DESC, due_date ASC, created_at ASC"
	}
	return col + " " + d
}

func nullTime(t *time.Time) interface{} {
	if t == nil {
		return nil
	}
	return t.UTC()
}

func nullString(s *string) interface{} {
	if s == nil {
		return nil
	}
	return *s
}

// ---- UpdateRepoSQLite -------------------------------------------------------

// UpdateRepoSQLite implements domain.UpdateRepository backed by SQLite.
type UpdateRepoSQLite struct {
	db *sql.DB
}

// NewUpdateRepo creates a new UpdateRepoSQLite.
func NewUpdateRepo(db *sql.DB) *UpdateRepoSQLite {
	return &UpdateRepoSQLite{db: db}
}

// AddUpdate inserts an immutable update/comment for a task.
func (r *UpdateRepoSQLite) AddUpdate(u *domain.Update) error {
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
func (r *UpdateRepoSQLite) ListByTaskID(taskID string) ([]domain.Update, error) {
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
