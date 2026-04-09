package repository

import (
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/diptopandit/tork/internal/domain"
)

// ListRepoSQLite implements domain.TaskListRepository backed by SQLite.
type ListRepoSQLite struct {
	db *sql.DB
}

// NewListRepo creates a new ListRepoSQLite.
func NewListRepo(db *sql.DB) *ListRepoSQLite {
	return &ListRepoSQLite{db: db}
}

// Create inserts a TaskList.
func (r *ListRepoSQLite) Create(list *domain.TaskList) error {
	schemaJSON, err := json.Marshal(list.Schema)
	if err != nil {
		return fmt.Errorf("list create: marshal schema: %w", err)
	}
	_, err = r.db.Exec(
		`INSERT INTO task_lists (id, name, schema_json, created_at) VALUES (?,?,?,?)`,
		list.ID, list.Name, string(schemaJSON), list.CreatedAt.UTC(),
	)
	if err != nil {
		return fmt.Errorf("list create: %w", err)
	}
	return nil
}

// Update updates a TaskList's name and schema.
func (r *ListRepoSQLite) Update(list *domain.TaskList) error {
	schemaJSON, err := json.Marshal(list.Schema)
	if err != nil {
		return fmt.Errorf("list update: marshal schema: %w", err)
	}
	res, err := r.db.Exec(
		`UPDATE task_lists SET name=?, schema_json=? WHERE id=?`,
		list.Name, string(schemaJSON), list.ID,
	)
	if err != nil {
		return fmt.Errorf("list update: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("list update: not found: %s", list.ID)
	}
	return nil
}

// GetByID fetches a TaskList by ID.
func (r *ListRepoSQLite) GetByID(id string) (*domain.TaskList, error) {
	row := r.db.QueryRow(
		`SELECT id, name, schema_json, created_at FROM task_lists WHERE id=?`, id)
	l, err := scanList(row)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("list not found: %s", id)
	}
	if err != nil {
		return nil, fmt.Errorf("list get: %w", err)
	}
	return l, nil
}

// GetAll returns every TaskList.
func (r *ListRepoSQLite) GetAll() ([]domain.TaskList, error) {
	rows, err := r.db.Query(
		`SELECT id, name, schema_json, created_at FROM task_lists ORDER BY created_at ASC`)
	if err != nil {
		return nil, fmt.Errorf("list get-all: %w", err)
	}
	defer rows.Close()

	var lists []domain.TaskList
	for rows.Next() {
		l, err := scanList(rows)
		if err != nil {
			return nil, fmt.Errorf("list get-all scan: %w", err)
		}
		lists = append(lists, *l)
	}
	return lists, rows.Err()
}

// Delete removes a TaskList and cascades to its tasks.
func (r *ListRepoSQLite) Delete(listID string) error {
	_, err := r.db.Exec("DELETE FROM task_lists WHERE id=?", listID)
	if err != nil {
		return fmt.Errorf("list delete: %w", err)
	}
	return nil
}

// ---- helpers ----------------------------------------------------------------

func scanList(s scanner) (*domain.TaskList, error) {
	var l domain.TaskList
	var schemaJSON string
	if err := s.Scan(&l.ID, &l.Name, &schemaJSON, &l.CreatedAt); err != nil {
		return nil, err
	}
	if err := json.Unmarshal([]byte(schemaJSON), &l.Schema); err != nil {
		l.Schema = nil
	}
	return &l, nil
}
