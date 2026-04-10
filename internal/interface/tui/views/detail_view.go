package views

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/diptopandit/tork/internal/domain"
	"github.com/diptopandit/tork/internal/infrastructure/config"
	"github.com/diptopandit/tork/internal/interface/tui/styles"
)

// UpdateSubmittedMsg is sent when the user submits a new update from the inline input.
type UpdateSubmittedMsg struct {
	TaskID string
	Body   string
}

// DetailView manages task detail, updates viewport, and update input.
// Each section is rendered independently so the caller can compose them
// into separate bordered boxes.
type DetailView struct {
	cfg        config.DisplayConfig
	styles     styles.Styles
	priorities []config.PriorityDef
	statuses   []config.StatusDef
	task       *domain.Task
	updates    viewport.Model
	input      textarea.Model
	inputFocus bool
	width      int
}

// NewDetailView constructs a DetailView.
func NewDetailView(cfg config.DisplayConfig, s styles.Styles, priorities []config.PriorityDef, statuses []config.StatusDef) DetailView {
	vp := viewport.New(80, 10)

	ta := textarea.New()
	ta.Placeholder = "Type update… (enter submit, esc cancel)"
	ta.CharLimit = 500
	ta.ShowLineNumbers = false
	ta.Blur()

	return DetailView{
		cfg:        cfg,
		styles:     s,
		priorities: priorities,
		statuses:   statuses,
		updates:    vp,
		input:      ta,
	}
}

// SetTask updates the displayed task and refreshes updates content.
func (m DetailView) SetTask(t *domain.Task) DetailView {
	m.task = t
	m.updates.SetContent(m.renderUpdates(t))
	m.updates.GotoBottom()
	return m
}

// SetWidth sets the available inner width for all sections.
func (m DetailView) SetWidth(w int) DetailView {
	m.width = w
	m.input.SetWidth(w - 4)
	m.input.SetHeight(1)
	if m.task != nil {
		m.updates.SetContent(m.renderUpdates(m.task))
	}
	return m
}

// SetUpdatesHeight sets the viewport height for the updates section.
func (m DetailView) SetUpdatesHeight(h int) DetailView {
	m.updates.Width = m.width
	m.updates.Height = h
	return m
}

// FocusInput activates the textarea for typing.
func (m DetailView) FocusInput() DetailView {
	m.inputFocus = true
	m.input.Focus()
	return m
}

// BlurInput deactivates the textarea.
func (m DetailView) BlurInput() DetailView {
	m.inputFocus = false
	m.input.Blur()
	m.input.Reset()
	return m
}

// InputFocused reports whether the textarea currently has focus.
func (m DetailView) InputFocused() bool {
	return m.inputFocus
}

// Update handles messages for the updates viewport and input textarea.
func (m DetailView) Update(msg tea.Msg) (DetailView, tea.Cmd) {
	if m.inputFocus {
		if kmsg, ok := msg.(tea.KeyMsg); ok {
			switch {
			case kmsg.Type == tea.KeyEsc:
				m = m.BlurInput()
				return m, nil
			case kmsg.Type == tea.KeyEnter, kmsg.Type == tea.KeyCtrlS:
				body := strings.TrimSpace(m.input.Value())
				if body != "" && m.task != nil {
					taskID := m.task.ID
					m = m.BlurInput()
					return m, func() tea.Msg {
						return UpdateSubmittedMsg{TaskID: taskID, Body: body}
					}
				}
				return m, nil
			}
		}
		var cmd tea.Cmd
		m.input, cmd = m.input.Update(msg)
		return m, cmd
	}

	// When not input-focused, scroll the updates viewport
	var cmd tea.Cmd
	m.updates, cmd = m.updates.Update(msg)
	return m, cmd
}

// ViewDetails renders the fixed task-detail section.
func (m DetailView) ViewDetails() string {
	if m.task == nil {
		return m.styles.EmptyState.Render("No task selected.")
	}
	return m.renderDetails(m.task)
}

// ViewUpdates renders the scrollable updates viewport.
func (m DetailView) ViewUpdates() string {
	return m.updates.View()
}

// ViewInput renders the update input textarea.
func (m DetailView) ViewInput() string {
	return m.input.View()
}

// renderDetails builds the fixed top section content.
func (m DetailView) renderDetails(t *domain.Task) string {
	if t == nil {
		return ""
	}
	labelStyle := m.styles.FieldLabel
	valueStyle := m.styles.FieldValue

	var sb strings.Builder

	// Line 1: Title with bright contrasting color
	sb.WriteString(m.styles.Title.Render(fmt.Sprintf("#%d %s", t.NumID, t.Title)) + "\n")

	// Two-column layout helper
	colW := m.width / 2
	if colW < 15 {
		colW = 15
	}
	leftCol := lipgloss.NewStyle().Width(colW)

	// Status color from positional theme colors
	statusColor := m.styles.Theme.Primary
	for i, s := range m.statuses {
		if s.Name == string(t.Status) {
			statusColor = m.styles.Theme.StatusColor(i)
			break
		}
	}
	statusValueStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(statusColor))

	// Line 2: Status | Created
	statusCell := labelStyle.Render("Status: ") + statusValueStyle.Render(string(t.Status))
	createdCell := ""
	if m.cfg.ShowTimestamps {
		createdCell = labelStyle.Render("Created: ") + valueStyle.Render(t.CreatedAt.Format(m.cfg.DateFormat+" 15:04"))
	}
	sb.WriteString(leftCol.Render(statusCell) + createdCell + "\n")

	// Priority color from positional theme colors
	priColor := m.styles.Theme.Primary
	for i, p := range m.priorities {
		if p.Value == int(t.Priority) {
			priColor = m.styles.Theme.PriorityColor(i)
			break
		}
	}
	priValueStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(priColor))

	// Line 3: Priority | Modified
	priCell := labelStyle.Render("Priority: ") + priValueStyle.Render(config.PriorityLabel(m.priorities, int(t.Priority)))
	modifiedCell := ""
	if m.cfg.ShowTimestamps {
		modifiedCell = labelStyle.Render("Modified: ") + valueStyle.Render(t.UpdatedAt.Format(m.cfg.DateFormat+" 15:04"))
	}
	sb.WriteString(leftCol.Render(priCell) + modifiedCell + "\n")

	// Line 4: Due date | (reserved)
	due := "—"
	if t.DueDate != nil {
		due = t.DueDate.Format(m.cfg.DateFormat)
	}
	dueCell := labelStyle.Render("Due: ") + valueStyle.Render(due)
	sb.WriteString(leftCol.Render(dueCell) + "\n")

	// Line 5+: Description
	if t.Description != "" {
		sb.WriteString(labelStyle.Render("Desc: ") + valueStyle.Render(t.Description) + "\n")
	}

	return sb.String()
}

// renderUpdates builds the scrollable updates content (newest first).
func (m DetailView) renderUpdates(t *domain.Task) string {
	if t == nil || len(t.Updates) == 0 {
		return m.styles.EmptyState.Render("  No updates yet. Press u to add one.")
	}
	valueStyle := m.styles.FieldValue
	tsStyle := m.styles.Timestamp
	var sb strings.Builder
	for _, u := range t.Updates {
		ts := u.CreatedAt.Format(m.cfg.DateFormat + " 15:04")
		sb.WriteString(tsStyle.Render("["+ts+"]") + "  " + valueStyle.Render(u.Body) + "\n")
	}
	return sb.String()
}
