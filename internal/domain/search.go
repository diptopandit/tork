package domain

import "time"

// TaskFilter holds all the criteria for filtering a list of tasks.
type TaskFilter struct {
	ListIDs    []string
	Statuses   []Status
	Priorities []Priority
	Tags       []string
	Search     string
	DueBefore  *time.Time
}

// SearchService is the port for full-text search indexing and querying.
type SearchService interface {
	Index(task *Task) error
	Delete(taskID string) error
	Search(query string) ([]string, error) // returns task IDs
}
