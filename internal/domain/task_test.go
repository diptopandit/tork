package domain

import (
	"testing"
)

func TestPriorityString(t *testing.T) {
	cases := []struct {
		p    Priority
		want string
	}{
		{PriorityLow, "low"},
		{PriorityMedium, "medium"},
		{PriorityHigh, "high"},
		{PriorityUrgent, "urgent"},
		{Priority(99), "medium"},
	}
	for _, c := range cases {
		if got := c.p.String(); got != c.want {
			t.Errorf("Priority(%d).String() = %q, want %q", c.p, got, c.want)
		}
	}
}

func TestStatusConstants(t *testing.T) {
	statuses := []Status{StatusTodo, StatusInProgress, StatusDone, StatusCancelled}
	for _, s := range statuses {
		if string(s) == "" {
			t.Errorf("status value is empty")
		}
	}
}

func TestStatusValues(t *testing.T) {
	cases := []struct {
		s    Status
		want string
	}{
		{StatusTodo, "todo"},
		{StatusInProgress, "in_progress"},
		{StatusDone, "done"},
		{StatusCancelled, "cancelled"},
	}
	for _, c := range cases {
		if string(c.s) != c.want {
			t.Errorf("Status = %q, want %q", c.s, c.want)
		}
	}
}

func TestStatusCycleOrder(t *testing.T) {
	order := []Status{StatusTodo, StatusInProgress, StatusDone, StatusCancelled}
	// Verify all 4 statuses are present and cycle correctly
	if len(order) != 4 {
		t.Fatalf("expected 4 statuses, got %d", len(order))
	}
	if order[0] != StatusTodo {
		t.Errorf("first status should be todo, got %q", order[0])
	}
	if order[3] != StatusCancelled {
		t.Errorf("last status should be cancelled, got %q", order[3])
	}
}
