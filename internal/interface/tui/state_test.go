package tui

import (
	"testing"

	"github.com/diptopandit/tork/internal/domain"
	"github.com/diptopandit/tork/internal/infrastructure/config"
)

func TestTabLabel(t *testing.T) {
	defs := []config.StatusDef{
		{Name: "todo", Label: "Todo"},
		{Name: "in_progress", Label: "In Progress"},
	}

	if got := TabLabel(defs, "todo"); got != "Todo" {
		t.Errorf("TabLabel(todo) = %q", got)
	}
	if got := TabLabel(defs, "all"); got != "All" {
		t.Errorf("TabLabel(all) = %q", got)
	}
	if got := TabLabel(defs, "in_progress"); got != "In Progress" {
		t.Errorf("TabLabel(in_progress) = %q", got)
	}
	if got := TabLabel(defs, "unknown"); got != "unknown" {
		t.Errorf("TabLabel(unknown) = %q, want fallback", got)
	}
}

func TestTabStatus(t *testing.T) {
	if s := TabStatus("all"); s != nil {
		t.Errorf("TabStatus(all) = %v, want nil", s)
	}
	s := TabStatus("todo")
	if len(s) != 1 || s[0] != domain.StatusTodo {
		t.Errorf("TabStatus(todo) = %v", s)
	}
	s2 := TabStatus("in_progress")
	if len(s2) != 1 || s2[0] != domain.StatusInProgress {
		t.Errorf("TabStatus(in_progress) = %v", s2)
	}
}

func TestValidTab(t *testing.T) {
	defs := []config.StatusDef{
		{Name: "todo"},
		{Name: "done"},
	}

	if !ValidTab(defs, "all") {
		t.Error("all should always be valid")
	}
	if !ValidTab(defs, "todo") {
		t.Error("todo should be valid")
	}
	if ValidTab(defs, "nope") {
		t.Error("nope should not be valid")
	}
}
