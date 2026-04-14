package search

import (
	"database/sql"
	"fmt"

	"github.com/diptopandit/tork/internal/domain"
)

// MySQLSearch implements domain.SearchService using MySQL FULLTEXT indexes.
type MySQLSearch struct {
	db *sql.DB
}

// NewMySQLSearch creates a MySQLSearch backed by the given DB connection.
func NewMySQLSearch(db *sql.DB) *MySQLSearch {
	return &MySQLSearch{db: db}
}

// Index is a no-op for MySQL — the FULLTEXT index is maintained automatically.
func (s *MySQLSearch) Index(_ *domain.Task) error {
	return nil
}

// Delete is a no-op for MySQL — rows are removed via DELETE cascade.
func (s *MySQLSearch) Delete(_ string) error {
	return nil
}

// Search performs a FULLTEXT query and returns matching task IDs.
func (s *MySQLSearch) Search(query string) ([]string, error) {
	rows, err := s.db.Query(
		`SELECT id FROM tasks WHERE MATCH(title, description) AGAINST(? IN BOOLEAN MODE)`,
		query,
	)
	if err != nil {
		return nil, fmt.Errorf("mysql search: %w", err)
	}
	defer rows.Close()

	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("mysql search scan: %w", err)
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}
