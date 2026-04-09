package search

import (
	"database/sql"
	"fmt"

	"github.com/diptopandit/tork/internal/domain"
)

// FTSSearch implements domain.SearchService using SQLite FTS5.
type FTSSearch struct {
	db *sql.DB
}

// NewFTSSearch creates a FTSSearch backed by the given DB connection.
func NewFTSSearch(db *sql.DB) *FTSSearch {
	return &FTSSearch{db: db}
}

// Index inserts or refreshes the FTS index entry for a task.
// The triggers on the tasks table keep the FTS table in sync automatically,
// so this method is provided for cases where an explicit re-index is needed.
func (s *FTSSearch) Index(task *domain.Task) error {
	// Delete the old entry first (idempotent).
	if err := s.Delete(task.ID); err != nil {
		return err
	}
	_, err := s.db.Exec(
		`INSERT INTO tasks_fts(id, title, description) VALUES (?,?,?)`,
		task.ID, task.Title, task.Description,
	)
	if err != nil {
		return fmt.Errorf("fts index: %w", err)
	}
	return nil
}

// Delete removes a task from the FTS index.
func (s *FTSSearch) Delete(taskID string) error {
	_, err := s.db.Exec(
		`DELETE FROM tasks_fts WHERE id=?`, taskID,
	)
	if err != nil {
		return fmt.Errorf("fts delete: %w", err)
	}
	return nil
}

// Search performs a full-text query and returns matching task IDs.
func (s *FTSSearch) Search(query string) ([]string, error) {
	rows, err := s.db.Query(
		`SELECT id FROM tasks_fts WHERE tasks_fts MATCH ? ORDER BY rank`,
		query,
	)
	if err != nil {
		return nil, fmt.Errorf("fts search: %w", err)
	}
	defer rows.Close()

	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("fts search scan: %w", err)
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}
