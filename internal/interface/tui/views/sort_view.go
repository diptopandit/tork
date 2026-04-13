package views

import (
	"fmt"
	"strings"

	"github.com/diptopandit/tork/internal/domain"
	"github.com/diptopandit/tork/internal/interface/tui/styles"
)

// SortOption represents a single sort choice.
type SortOption struct {
	Field domain.SortField
	Dir   domain.SortDir
	Label string
}

var sortOptions = []SortOption{
	{Field: domain.SortByPriority, Dir: domain.SortDesc, Label: "Priority (high → low)"},
	{Field: domain.SortByPriority, Dir: domain.SortAsc, Label: "Priority (low → high)"},
	{Field: domain.SortByDueDate, Dir: domain.SortAsc, Label: "Due date (earliest first)"},
	{Field: domain.SortByDueDate, Dir: domain.SortDesc, Label: "Due date (latest first)"},
	{Field: domain.SortByID, Dir: domain.SortAsc, Label: "ID (oldest first)"},
	{Field: domain.SortByID, Dir: domain.SortDesc, Label: "ID (newest first)"},
	{Field: "", Dir: "", Label: "Default (priority, due date, created)"},
}

// SortView presents a list of sort options.
type SortView struct {
	cursor   int
	styles   styles.Styles
	selected bool
}

// NewSortView constructs a SortView.
func NewSortView(s styles.Styles) SortView {
	return SortView{styles: s}
}

// SetStyles replaces the styles used for rendering.
func (v SortView) SetStyles(s styles.Styles) SortView {
	v.styles = s
	return v
}

// CursorUp moves the cursor up.
func (v SortView) CursorUp() SortView {
	if v.cursor > 0 {
		v.cursor--
	}
	return v
}

// CursorDown moves the cursor down.
func (v SortView) CursorDown() SortView {
	if v.cursor < len(sortOptions)-1 {
		v.cursor++
	}
	return v
}

// Select marks the current option as selected.
func (v SortView) Select() SortView {
	v.selected = true
	return v
}

// Selected reports whether the user pressed Enter.
func (v SortView) Selected() bool { return v.selected }

// SelectedOption returns the chosen sort option.
func (v SortView) SelectedOption() SortOption {
	if v.cursor >= 0 && v.cursor < len(sortOptions) {
		return sortOptions[v.cursor]
	}
	return SortOption{}
}

// ResetSelected clears the selected flag.
func (v SortView) ResetSelected() SortView {
	v.selected = false
	return v
}

// SetCursorFromSort positions the cursor to match the current sort.
func (v SortView) SetCursorFromSort(field domain.SortField, dir domain.SortDir) SortView {
	for i, o := range sortOptions {
		if o.Field == field && o.Dir == dir {
			v.cursor = i
			return v
		}
	}
	// Default option is last
	v.cursor = len(sortOptions) - 1
	return v
}

// View renders the sort picker.
func (v SortView) View() string {
	var b strings.Builder

	b.WriteString(v.styles.HelpTitle.Render("Sort Tasks") + "\n")
	b.WriteString(v.styles.HintText.Render("j/k navigate • Enter select • Esc cancel") + "\n\n")

	for i, opt := range sortOptions {
		label := fmt.Sprintf("  %s", opt.Label)
		if i == v.cursor {
			b.WriteString(v.styles.ListCursor.Render("▸ "+opt.Label) + "\n")
		} else {
			b.WriteString(v.styles.ListNormal.Render(label) + "\n")
		}
	}

	return b.String()
}
