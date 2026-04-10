package cli

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/diptopandit/tork/internal/application"
	"github.com/diptopandit/tork/internal/domain"
	"github.com/diptopandit/tork/internal/infrastructure/config"
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

// ParsePriority converts a string (name or 1-indexed position) to a domain.Priority
// using the configured priority definitions.
func ParsePriority(defs []config.PriorityDef, s string) (domain.Priority, error) {
	if s == "" {
		return domain.Priority(config.DefaultPriority(defs)), nil
	}
	// Try name match first.
	if v, ok := config.PriorityByName(defs, s); ok {
		return domain.Priority(v), nil
	}
	// Try as a 1-indexed position.
	if idx, err := strconv.Atoi(s); err == nil && idx >= 1 && idx <= len(defs) {
		return domain.Priority(defs[idx-1].Value), nil
	}
	names := make([]string, len(defs))
	for i, d := range defs {
		names[i] = d.Name
	}
	return 0, fmt.Errorf("unknown priority %q (use %s)", s, strings.Join(names, "/"))
}

// ParseStatus converts a string to a domain.Status using the configured status definitions.
func ParseStatus(defs []config.StatusDef, s string) (domain.Status, error) {
	if config.ValidStatus(defs, s) {
		return domain.Status(s), nil
	}
	// Also accept dash-separated form (in-progress → in_progress).
	alt := strings.ReplaceAll(s, "-", "_")
	if config.ValidStatus(defs, alt) {
		return domain.Status(alt), nil
	}
	names := config.StatusNames(defs)
	return "", fmt.Errorf("unknown status %q (use %s)", s, strings.Join(names, "/"))
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
