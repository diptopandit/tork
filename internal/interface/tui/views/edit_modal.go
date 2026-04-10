package views

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/diptopandit/tork/internal/domain"
	"github.com/diptopandit/tork/internal/infrastructure/config"
	"github.com/diptopandit/tork/internal/interface/tui/styles"
)

// EditView is a modal form for creating or editing a task.
type EditView struct {
	titleInput textinput.Model
	descInput  textarea.Model
	priorityIn textinput.Model
	dueDateIn  textinput.Model
	tagsIn     textinput.Model
	focusIndex int          // 0=title,1=desc,2=priority,3=due,4=tags
	task       *domain.Task // nil when creating a new task
	savedTask  *domain.Task // non-nil when user pressed Enter/Ctrl+S
	styles     styles.Styles
	statuses   []config.StatusDef
	priorities []config.PriorityDef
	width      int
	height     int
}

const (
	efTitle    = 0
	efDesc     = 1
	efPriority = 2
	efDue      = 3
	efTags     = 4
	efCount    = 5
)

// NewEditView constructs an EditView with empty inputs.
func NewEditView(s styles.Styles, statuses []config.StatusDef, priorities []config.PriorityDef) EditView {
	ti := textinput.New()
	ti.Placeholder = "Task title"
	ti.Focus()
	ti.CharLimit = 200

	ta := textarea.New()
	ta.Placeholder = "Description (optional)"
	ta.CharLimit = 2000
	ta.SetHeight(5)

	// Build dynamic placeholder from configured priorities.
	var priParts []string
	for i, p := range priorities {
		priParts = append(priParts, fmt.Sprintf("%d=%s", i+1, p.Name))
	}
	defIdx := len(priorities)/2 + 1
	if defIdx < 1 {
		defIdx = 1
	}
	pi := textinput.New()
	pi.Placeholder = fmt.Sprintf("%s  (default: %d)", strings.Join(priParts, " "), defIdx)
	pi.CharLimit = 2

	di := textinput.New()
	di.Placeholder = "DD-MM-YYYY  (optional)"
	di.CharLimit = 10

	tgs := textinput.New()
	tgs.Placeholder = "comma-separated tags  (optional)"
	tgs.CharLimit = 200

	return EditView{
		titleInput: ti,
		descInput:  ta,
		priorityIn: pi,
		dueDateIn:  di,
		tagsIn:     tgs,
		styles:     s,
		statuses:   statuses,
		priorities: priorities,
	}
}

// SetTask populates the form from an existing task (edit mode) or clears it (create mode).
func (m EditView) SetTask(t *domain.Task) EditView {
	m.savedTask = nil
	m.task = t
	m.focusIndex = efTitle
	m.titleInput.SetValue("")
	m.descInput.SetValue("")
	m.priorityIn.SetValue("")
	m.dueDateIn.SetValue("")
	m.tagsIn.SetValue("")

	if t != nil {
		m.titleInput.SetValue(t.Title)
		m.descInput.SetValue(t.Description)
		// Show the 1-indexed position of the task's priority in the config list.
		priStr := "1"
		for i, p := range m.priorities {
			if p.Value == int(t.Priority) {
				priStr = strconv.Itoa(i + 1)
				break
			}
		}
		m.priorityIn.SetValue(priStr)
		if t.DueDate != nil {
			m.dueDateIn.SetValue(t.DueDate.Format("02-01-2006"))
		}
		if len(t.Tags) > 0 {
			m.tagsIn.SetValue(strings.Join(t.Tags, ", "))
		}
	}
	m.titleInput.Focus()
	return m
}

// SavedTask returns the task produced when the form was submitted, or nil.
func (m EditView) SavedTask() *domain.Task {
	return m.savedTask
}

// ClearSaved resets the saved-task pointer so the parent doesn't re-process it.
func (m EditView) ClearSaved() EditView {
	m.savedTask = nil
	return m
}

// SetSize resizes the form.
func (m EditView) SetSize(w, h int) EditView {
	m.width = w
	m.height = h
	m.descInput.SetWidth(w - 4)
	return m
}

// Update processes keyboard input.
func (m EditView) Update(msg tea.Msg) (EditView, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyTab:
			m = m.nextField()
		case tea.KeyShiftTab:
			m = m.prevField()
		case tea.KeyEnter:
			if m.focusIndex != efDesc { // Enter inside textarea = newline
				return m.submit()
			}
		case tea.KeyCtrlS:
			return m.submit()
		}
	}

	var cmd tea.Cmd
	switch m.focusIndex {
	case efTitle:
		m.titleInput, cmd = m.titleInput.Update(msg)
	case efDesc:
		m.descInput, cmd = m.descInput.Update(msg)
	case efPriority:
		m.priorityIn, cmd = m.priorityIn.Update(msg)
	case efDue:
		m.dueDateIn, cmd = m.dueDateIn.Update(msg)
	case efTags:
		m.tagsIn, cmd = m.tagsIn.Update(msg)
	}
	return m, cmd
}

// View renders the edit form.
func (m EditView) View() string {
	heading := "New Task"
	if m.task != nil {
		heading = "Edit Task"
	}
	title := m.styles.LabelFocused.Render(heading)

	hint := m.styles.HintText.
		Render("Tab to navigate  •  Enter/Ctrl+S to save  •  Esc to cancel")

	fields := []string{
		m.fieldRow("Title", m.titleInput.View(), m.focusIndex == efTitle),
		m.fieldRow("Description", m.descInput.View(), m.focusIndex == efDesc),
		m.fieldRow("Priority", m.priorityIn.View(), m.focusIndex == efPriority),
		m.fieldRow("Due Date", m.dueDateIn.View(), m.focusIndex == efDue),
		m.fieldRow("Tags", m.tagsIn.View(), m.focusIndex == efTags),
	}

	return title + "\n\n" + strings.Join(fields, "\n\n") + "\n\n" + hint
}

// ---- helpers ----------------------------------------------------------------

func (m EditView) fieldRow(label, input string, focused bool) string {
	lStyle := m.styles.LabelUnfocused
	if focused {
		lStyle = m.styles.LabelFocused
	}
	// Indent all lines consistently (fixes multiline textarea alignment).
	indented := "  " + strings.ReplaceAll(input, "\n", "\n  ")
	return lStyle.Render(label+":") + "\n" + indented
}

func (m EditView) nextField() EditView {
	m = m.blurAll()
	m.focusIndex = (m.focusIndex + 1) % efCount
	return m.focusCurrent()
}

func (m EditView) prevField() EditView {
	m = m.blurAll()
	m.focusIndex = (m.focusIndex - 1 + efCount) % efCount
	return m.focusCurrent()
}

func (m EditView) blurAll() EditView {
	m.titleInput.Blur()
	m.descInput.Blur()
	m.priorityIn.Blur()
	m.dueDateIn.Blur()
	m.tagsIn.Blur()
	return m
}

func (m EditView) focusCurrent() EditView {
	switch m.focusIndex {
	case efTitle:
		m.titleInput.Focus()
	case efDesc:
		m.descInput.Focus()
	case efPriority:
		m.priorityIn.Focus()
	case efDue:
		m.dueDateIn.Focus()
	case efTags:
		m.tagsIn.Focus()
	}
	return m
}

func (m EditView) submit() (EditView, tea.Cmd) {
	title := strings.TrimSpace(m.titleInput.Value())
	if title == "" {
		return m, nil // silently ignore empty title
	}

	// Resolve priority from config: accept 1-indexed number or name.
	priority := domain.Priority(config.DefaultPriority(m.priorities))
	if raw := strings.TrimSpace(m.priorityIn.Value()); raw != "" {
		if v, ok := config.PriorityByName(m.priorities, raw); ok {
			priority = domain.Priority(v)
		} else if idx, err := strconv.Atoi(raw); err == nil && idx >= 1 && idx <= len(m.priorities) {
			priority = domain.Priority(m.priorities[idx-1].Value)
		}
	}

	var dueDate *time.Time
	if ds := strings.TrimSpace(m.dueDateIn.Value()); ds != "" {
		for _, layout := range []string{"02-01-2006", "2006-01-02"} {
			if t, err := time.Parse(layout, ds); err == nil {
				dueDate = &t
				break
			}
		}
	}

	var tags []string
	if raw := strings.TrimSpace(m.tagsIn.Value()); raw != "" {
		for _, t := range strings.Split(raw, ",") {
			if s := strings.TrimSpace(t); s != "" {
				tags = append(tags, s)
			}
		}
	}

	saved := &domain.Task{
		Title:       title,
		Description: strings.TrimSpace(m.descInput.Value()),
		Priority:    priority,
		DueDate:     dueDate,
		Tags:        tags,
		Status:      domain.Status(config.DefaultStatus(m.statuses)),
	}
	if m.task != nil {
		saved.ID = m.task.ID
		saved.ListID = m.task.ListID
		saved.Status = m.task.Status
		saved.CustomFields = m.task.CustomFields
		saved.DependsOn = m.task.DependsOn
		saved.CreatedAt = m.task.CreatedAt
	}

	m.savedTask = saved
	return m, nil
}

// SetStyles replaces the styles used for rendering.
func (m EditView) SetStyles(s styles.Styles) EditView {
	m.styles = s
	return m
}
