package cli

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/diptopandit/tork/internal/application"
	"github.com/diptopandit/tork/internal/domain"
)

// resolveTaskID accepts a numeric ID (optionally prefixed with #) or a UUID
// prefix and returns the full UUID. Numeric IDs are preferred for CLI usage.
func resolveTaskID(taskSvc *application.TaskService, raw string) (string, error) {
	s := strings.TrimPrefix(raw, "#")
	if n, err := strconv.Atoi(s); err == nil {
		t, err := taskSvc.GetTaskByNum(n)
		if err != nil {
			return "", fmt.Errorf("task #%d not found", n)
		}
		return t.ID, nil
	}
	return raw, nil
}

// ParsePriority converts a string ("low", "medium", "high", "urgent", or "1"-"4")
// to a domain.Priority.
func ParsePriority(s string) (domain.Priority, error) {
	switch s {
	case "1", "low":
		return domain.PriorityLow, nil
	case "2", "medium", "":
		return domain.PriorityMedium, nil
	case "3", "high":
		return domain.PriorityHigh, nil
	case "4", "urgent":
		return domain.PriorityUrgent, nil
	default:
		return 0, fmt.Errorf("unknown priority %q (use low/medium/high/urgent)", s)
	}
}

// ParseStatus converts a string to a domain.Status.
func ParseStatus(s string) (domain.Status, error) {
	switch s {
	case "todo":
		return domain.StatusTodo, nil
	case "in_progress", "in-progress":
		return domain.StatusInProgress, nil
	case "done":
		return domain.StatusDone, nil
	case "cancelled":
		return domain.StatusCancelled, nil
	default:
		return "", fmt.Errorf("unknown status %q (use todo/in_progress/done/cancelled)", s)
	}
}

// ParseDate parses a date string in DD-MM-YYYY or YYYY-MM-DD format into a *time.Time.
// Returns nil, nil when s is empty.
func ParseDate(s string) (*time.Time, error) {
	if s == "" {
		return nil, nil
	}
	for _, layout := range []string{"02-01-2006", "2006-01-02"} {
		if t, err := time.Parse(layout, s); err == nil {
			return &t, nil
		}
	}
	return nil, fmt.Errorf("invalid date %q (expected DD-MM-YYYY or YYYY-MM-DD)", s)
}
