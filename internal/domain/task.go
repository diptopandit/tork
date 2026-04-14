package domain

import "time"

// Status represents the lifecycle state of a task.
type Status string

const (
	StatusTodo       Status = "todo"
	StatusInProgress Status = "in_progress"
	StatusDone       Status = "done"
	StatusCancelled  Status = "cancelled"
)

// Priority represents the urgency level of a task.
type Priority int

const (
	PriorityLow    Priority = 1
	PriorityMedium Priority = 2
	PriorityHigh   Priority = 3
	PriorityUrgent Priority = 4
)

// String returns a human-readable name for the priority level.
func (p Priority) String() string {
	switch p {
	case PriorityLow:
		return "low"
	case PriorityMedium:
		return "medium"
	case PriorityHigh:
		return "high"
	case PriorityUrgent:
		return "urgent"
	default:
		return "medium"
	}
}

// FieldDefinition describes a custom field in a TaskList schema.
type FieldDefinition struct {
	Name     string `json:"name"`
	Type     string `json:"type"`
	Required bool   `json:"required"`
}

// Task is the core entity of the application.
type Task struct {
	ID          string
	NumID       int // human-friendly auto-increment ID
	ListID      string
	Title       string
	Description string
	Status      Status
	Priority    Priority
	DueDate     *time.Time
	Tags        []string

	CustomFields map[string]interface{}

	ParentID  *string
	DependsOn []string

	Updates []Update

	CreatedAt time.Time
	UpdatedAt time.Time
}

// Update is an immutable, timestamped comment/note on a task.
type Update struct {
	ID        string
	TaskID    string
	Body      string
	CreatedAt time.Time
}

// TaskList is a named collection of tasks that may carry a custom field schema.
type TaskList struct {
	ID         string
	Name       string
	Schema     map[string]FieldDefinition
	OwnerID    string // empty for SQLite (single-user); set for remote DB
	Visibility string // "private" or "shared"; empty defaults to "private"
	CreatedAt  time.Time
}

// User represents a tork user for multi-user remote DB mode.
type User struct {
	ID        string
	Username  string
	CreatedAt time.Time
}

// ListMember represents a user's membership in a shared list.
type ListMember struct {
	ListID string
	UserID string
	Role   string // "viewer", "editor", "admin"
}
