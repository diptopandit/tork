package views

import (
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/diptopandit/tork/internal/domain"
	"github.com/diptopandit/tork/internal/infrastructure/config"
	"github.com/diptopandit/tork/internal/interface/tui/styles"
)

// FilterView renders a dynamic filter builder panel.
type FilterView struct {
	searchInput textinput.Model
	statuses    map[domain.Status]bool
	priorities  map[domain.Priority]bool
	tagsInput   textinput.Model
	dueBefore   textinput.Model
	focusField  int // 0=search,1=due,2=tags
	applied     bool
	filter      domain.TaskFilter
	styles      styles.Styles
	width       int
	height      int
}

const (
	ffSearch = 0
	ffDue    = 1
	ffTags   = 2
	ffCount  = 3
)

// NewFilterView constructs a FilterView.
func NewFilterView(s styles.Styles, statuses []config.StatusDef, priorities []config.PriorityDef) FilterView {
	si := textinput.New()
	si.Placeholder = "Search..."
	si.Focus()
	si.CharLimit = 200

	di := textinput.New()
	di.Placeholder = "Due before YYYY-MM-DD"
	di.CharLimit = 10

	ti := textinput.New()
	ti.Placeholder = "Tags (comma-separated)"
	ti.CharLimit = 200

	smap := make(map[domain.Status]bool, len(statuses))
	for _, s := range statuses {
		smap[domain.Status(s.Name)] = false
	}

	pmap := make(map[domain.Priority]bool, len(priorities))
	for _, p := range priorities {
		pmap[domain.Priority(p.Value)] = false
	}

	return FilterView{
		searchInput: si,
		dueBefore:   di,
		tagsInput:   ti,
		styles:      s,
		statuses:    smap,
		priorities:  pmap,
	}
}

// FocusSearch moves keyboard focus to the search input.
func (m FilterView) FocusSearch() FilterView {
	m.focusField = ffSearch
	m.searchInput.Focus()
	m.dueBefore.Blur()
	m.tagsInput.Blur()
	return m
}

// Applied returns true if the user submitted the filter.
func (m FilterView) Applied() bool { return m.applied }

// Filter returns the most recently applied TaskFilter.
func (m FilterView) Filter() domain.TaskFilter { return m.filter }

// ClearApplied resets the applied flag.
func (m FilterView) ClearApplied() FilterView {
	m.applied = false
	return m
}

// SetSize resizes the panel.
func (m FilterView) SetSize(w, h int) FilterView {
	m.width = w
	m.height = h
	return m
}

// Update handles keyboard navigation.
func (m FilterView) Update(msg tea.Msg) (FilterView, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyTab, tea.KeyDown:
			m = m.nextField()
		case tea.KeyShiftTab, tea.KeyUp:
			m = m.prevField()
		case tea.KeyEnter:
			m.filter = m.buildFilter()
			m.applied = true
			return m, nil
		}
	}

	var cmd tea.Cmd
	switch m.focusField {
	case ffSearch:
		m.searchInput, cmd = m.searchInput.Update(msg)
	case ffDue:
		m.dueBefore, cmd = m.dueBefore.Update(msg)
	case ffTags:
		m.tagsInput, cmd = m.tagsInput.Update(msg)
	}
	return m, cmd
}

// View renders the filter panel.
func (m FilterView) View() string {
	heading := m.styles.LabelFocused.Render("Filters")
	hint := m.styles.HintText.
		Render("Tab/↑↓ to navigate  •  Enter to apply  •  Esc to cancel")

	rows := []string{
		heading,
		"",
		m.fieldRow("Search", m.searchInput.View(), m.focusField == ffSearch),
		"",
		m.fieldRow("Due Before", m.dueBefore.View(), m.focusField == ffDue),
		"",
		m.fieldRow("Tags", m.tagsInput.View(), m.focusField == ffTags),
		"",
		hint,
	}
	return strings.Join(rows, "\n")
}

// ---- helpers ----------------------------------------------------------------

func (m FilterView) fieldRow(label, input string, focused bool) string {
	lStyle := m.styles.LabelUnfocused
	if focused {
		lStyle = m.styles.LabelFocused
	}
	return lStyle.Render(label+":") + "\n  " + input
}

func (m FilterView) nextField() FilterView {
	m = m.blurFields()
	m.focusField = (m.focusField + 1) % ffCount
	return m.focusCurrentField()
}

func (m FilterView) prevField() FilterView {
	m = m.blurFields()
	m.focusField = (m.focusField - 1 + ffCount) % ffCount
	return m.focusCurrentField()
}

func (m FilterView) blurFields() FilterView {
	m.searchInput.Blur()
	m.dueBefore.Blur()
	m.tagsInput.Blur()
	return m
}

func (m FilterView) focusCurrentField() FilterView {
	switch m.focusField {
	case ffSearch:
		m.searchInput.Focus()
	case ffDue:
		m.dueBefore.Focus()
	case ffTags:
		m.tagsInput.Focus()
	}
	return m
}

func (m FilterView) buildFilter() domain.TaskFilter {
	f := domain.TaskFilter{
		Search: strings.TrimSpace(m.searchInput.Value()),
	}

	if ds := strings.TrimSpace(m.dueBefore.Value()); ds != "" {
		if t, err := time.Parse("2006-01-02", ds); err == nil {
			f.DueBefore = &t
		}
	}

	if raw := strings.TrimSpace(m.tagsInput.Value()); raw != "" {
		for _, t := range strings.Split(raw, ",") {
			if s := strings.TrimSpace(t); s != "" {
				f.Tags = append(f.Tags, s)
			}
		}
	}

	return f
}
