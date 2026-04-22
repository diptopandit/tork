package application

import (
	"strings"
	"testing"
	"time"

	"github.com/diptopandit/tork/internal/domain"
	"github.com/diptopandit/tork/internal/infrastructure/config"
)

// ---- FilterBuilder tests ----------------------------------------------------

func TestFilterBuilder_Empty(t *testing.T) {
	f := NewFilterBuilder().Build()
	if len(f.ListIDs) != 0 || len(f.Statuses) != 0 || len(f.Tags) != 0 || f.Search != "" || f.DueBefore != nil {
		t.Error("empty builder should produce empty filter")
	}
}

func TestFilterBuilder_WithLists(t *testing.T) {
	f := NewFilterBuilder().WithLists("a", "b").Build()
	if len(f.ListIDs) != 2 || f.ListIDs[0] != "a" || f.ListIDs[1] != "b" {
		t.Errorf("ListIDs = %v", f.ListIDs)
	}
}

func TestFilterBuilder_WithStatuses(t *testing.T) {
	f := NewFilterBuilder().WithStatuses(domain.StatusTodo, domain.StatusDone).Build()
	if len(f.Statuses) != 2 {
		t.Errorf("Statuses = %v", f.Statuses)
	}
}

func TestFilterBuilder_WithPriorities(t *testing.T) {
	f := NewFilterBuilder().WithPriorities(domain.PriorityHigh).Build()
	if len(f.Priorities) != 1 || f.Priorities[0] != domain.PriorityHigh {
		t.Errorf("Priorities = %v", f.Priorities)
	}
}

func TestFilterBuilder_WithTags(t *testing.T) {
	f := NewFilterBuilder().WithTags("go", "test").Build()
	if len(f.Tags) != 2 {
		t.Errorf("Tags = %v", f.Tags)
	}
}

func TestFilterBuilder_WithSearch(t *testing.T) {
	f := NewFilterBuilder().WithSearch("deploy").Build()
	if f.Search != "deploy" {
		t.Errorf("Search = %q", f.Search)
	}
}

func TestFilterBuilder_WithDueBefore(t *testing.T) {
	due := time.Date(2026, 12, 31, 0, 0, 0, 0, time.UTC)
	f := NewFilterBuilder().WithDueBefore(due).Build()
	if f.DueBefore == nil || !f.DueBefore.Equal(due) {
		t.Errorf("DueBefore = %v", f.DueBefore)
	}
}

func TestFilterBuilder_Chaining(t *testing.T) {
	due := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	f := NewFilterBuilder().
		WithLists("list-1").
		WithStatuses(domain.StatusTodo).
		WithPriorities(domain.PriorityHigh).
		WithTags("urgent").
		WithSearch("deploy").
		WithDueBefore(due).
		Build()

	if len(f.ListIDs) != 1 {
		t.Error("ListIDs")
	}
	if len(f.Statuses) != 1 {
		t.Error("Statuses")
	}
	if len(f.Priorities) != 1 {
		t.Error("Priorities")
	}
	if len(f.Tags) != 1 {
		t.Error("Tags")
	}
	if f.Search != "deploy" {
		t.Error("Search")
	}
	if f.DueBefore == nil {
		t.Error("DueBefore")
	}
}

// ---- ExportService tests ----------------------------------------------------

var testPriorities = []config.PriorityDef{
	{Name: "low", Value: 1, Label: "Low"},
	{Name: "medium", Value: 2, Label: "Medium"},
	{Name: "high", Value: 3, Label: "High"},
	{Name: "urgent", Value: 4, Label: "Urgent"},
}

func sampleTasks() []domain.Task {
	due := time.Date(2026, 4, 30, 0, 0, 0, 0, time.UTC)
	return []domain.Task{
		{ID: "t1", Title: "Ship README", Status: domain.StatusDone, Priority: 3, Tags: []string{"docs"}, DueDate: &due, ListID: "l1"},
		{ID: "t2", Title: "Fix bug", Status: domain.StatusTodo, Priority: 2, Tags: []string{"bug", "api"}, ListID: "l1"},
		{ID: "t3", Title: "Add tests", Status: domain.StatusTodo, Priority: 1, ListID: "l1"},
	}
}

func TestMarkdownExporter_Export(t *testing.T) {
	e := MarkdownExporter{Priorities: testPriorities}
	data, err := e.Export(sampleTasks())
	if err != nil {
		t.Fatal(err)
	}
	out := string(data)

	// Done tasks should have [x], others [ ]
	if !strings.Contains(out, "[x] **Ship README**") {
		t.Error("expected [x] for done task")
	}
	if !strings.Contains(out, "[ ] **Fix bug**") {
		t.Error("expected [ ] for todo task")
	}
	// Priority label
	if !strings.Contains(out, "(High)") {
		t.Error("expected (High) priority label")
	}
	// Due date
	if !strings.Contains(out, "due:2026-04-30") {
		t.Error("expected due date")
	}
	// No due date for task without one
	lines := strings.Split(out, "\n")
	for _, l := range lines {
		if strings.Contains(l, "Add tests") && strings.Contains(l, "due:") {
			t.Error("task without due date should not have due:")
		}
	}
}

func TestMarkdownExporter_EmptyTasks(t *testing.T) {
	e := MarkdownExporter{Priorities: testPriorities}
	data, err := e.Export(nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(data) != 0 {
		t.Errorf("expected empty output for nil tasks, got %q", data)
	}
}

func TestCSVExporter_Export(t *testing.T) {
	e := CSVExporter{Priorities: testPriorities}
	data, err := e.Export(sampleTasks())
	if err != nil {
		t.Fatal(err)
	}
	out := string(data)
	lines := strings.Split(strings.TrimSpace(out), "\n")

	// Header + 3 data rows
	if len(lines) != 4 {
		t.Fatalf("expected 4 lines, got %d", len(lines))
	}
	// Header
	if !strings.Contains(lines[0], "id,title,status,priority,due_date,tags,list_id") {
		t.Errorf("unexpected header: %s", lines[0])
	}
	// First data row — done task with due date
	if !strings.Contains(lines[1], "Ship README") || !strings.Contains(lines[1], "done") {
		t.Errorf("unexpected row 1: %s", lines[1])
	}
	if !strings.Contains(lines[1], "2026-04-30") {
		t.Error("expected due date in CSV")
	}
	// Tags use pipe separator
	if !strings.Contains(lines[2], "bug|api") {
		t.Errorf("expected pipe-separated tags in row 2: %s", lines[2])
	}
}

func TestCSVExporter_EmptyTasks(t *testing.T) {
	e := CSVExporter{Priorities: testPriorities}
	data, err := e.Export(nil)
	if err != nil {
		t.Fatal(err)
	}
	// Should still have header row.
	out := string(data)
	if !strings.Contains(out, "id,title") {
		t.Error("expected header even with no tasks")
	}
}
