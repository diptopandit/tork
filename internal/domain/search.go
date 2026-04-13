package domain

import "time"

// SortField identifies which task field to sort by.
type SortField string

const (
	SortByPriority SortField = "priority"
	SortByDueDate  SortField = "due_date"
	SortByID       SortField = "id"
)

// SortDir specifies ascending or descending sort direction.
type SortDir string

const (
	SortAsc  SortDir = "asc"
	SortDesc SortDir = "desc"
)

// TaskFilter holds all the criteria for filtering a list of tasks.
type TaskFilter struct {
	ListIDs    []string
	Statuses   []Status
	Priorities []Priority
	Tags       []string
	Search     string
	DueBefore  *time.Time
	SortField  SortField // empty means default
	SortDir    SortDir   // empty means default
}

// SearchService is the port for full-text search indexing and querying.
type SearchService interface {
	Index(task *Task) error
	Delete(taskID string) error
	Search(query string) ([]string, error) // returns task IDs
}
