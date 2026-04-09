package application

import (
	"time"

	"github.com/diptopandit/tork/internal/domain"
)

// CreateTaskInput carries the data needed to create a new task.
type CreateTaskInput struct {
	ListID       string
	Title        string
	Description  string
	Priority     domain.Priority
	DueDate      *time.Time
	Tags         []string
	CustomFields map[string]interface{}
	ParentID     *string
	DependsOn    []string
}

// UpdateTaskInput carries the fields that may be changed on an existing task.
// Only non-nil pointer fields are applied.
type UpdateTaskInput struct {
	ID           string
	Title        *string
	Description  *string
	Status       *domain.Status
	Priority     *domain.Priority
	DueDate      *time.Time
	Tags         []string
	CustomFields map[string]interface{}
	DependsOn    []string
}

// TaskFilter mirrors domain.TaskFilter but lives in the application layer for
// use by service methods that accept user-facing inputs.
type TaskFilter = domain.TaskFilter
