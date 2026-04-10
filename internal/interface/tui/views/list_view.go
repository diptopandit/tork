package views

import (
	"fmt"

	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/diptopandit/tork/internal/domain"
	"github.com/diptopandit/tork/internal/infrastructure/config"
)

// ListView wraps bubbles/table and manages the task list pane.
type ListView struct {
	table       table.Model
	cfg         config.DisplayConfig
	theme       config.ThemeConfig
	priorities  []config.PriorityDef
	tasks       []domain.Task
	width       int
	height      int
	activeStyle table.Styles
	inactStyle  table.Styles
}

// NewListView constructs a ListView.
func NewListView(cfg config.DisplayConfig, theme config.ThemeConfig, priorities []config.PriorityDef) ListView {
	activeStyle := table.DefaultStyles()
	activeStyle.Cell = activeStyle.Cell.Padding(0, 0)
	activeStyle.Header = lipgloss.NewStyle().
		Padding(0, 0).
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color(theme.Primary)).
		BorderBottom(true).
		Bold(true).
		Foreground(lipgloss.Color(theme.TextBright)).
		MaxWidth(80)
	activeStyle.Selected = lipgloss.NewStyle().
		Padding(0, 0).
		Foreground(lipgloss.Color(theme.TextBright)).
		Background(lipgloss.Color(theme.Primary)).
		Bold(true)

	inactStyle := table.DefaultStyles()
	inactStyle.Cell = inactStyle.Cell.Padding(0, 0)
	inactStyle.Header = lipgloss.NewStyle().
		Padding(0, 0).
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color(theme.Inactive)).
		BorderBottom(true).
		Bold(false).
		Foreground(lipgloss.Color(theme.TextMuted)).
		MaxWidth(80)
	inactStyle.Selected = lipgloss.NewStyle().
		Padding(0, 0).
		Foreground(lipgloss.Color(theme.Text)).
		Background(lipgloss.Color(theme.Inactive))

	t := table.New(
		table.WithColumns(buildColumns(cfg.Columns, 80)),
		table.WithFocused(true),
		table.WithStyles(activeStyle),
	)

	return ListView{
		table:       t,
		cfg:         cfg,
		theme:       theme,
		priorities:  priorities,
		activeStyle: activeStyle,
		inactStyle:  inactStyle,
	}
}

// SetActive updates the table focus style.
func (m ListView) SetActive(active bool) ListView {
	if active {
		m.table.SetStyles(m.activeStyle)
		m.table.Focus()
	} else {
		m.table.SetStyles(m.inactStyle)
		m.table.Blur()
	}
	return m
}

// SetTasks replaces the displayed tasks.
func (m ListView) SetTasks(tasks []domain.Task) ListView {
	m.tasks = tasks
	m.table.SetRows(tasksToRows(tasks, m.cfg, m.priorities))
	return m
}

// SelectedTask returns the task currently highlighted in the table, if any.
func (m ListView) SelectedTask() *domain.Task {
	row := m.table.Cursor()
	if row < 0 || row >= len(m.tasks) {
		return nil
	}
	t := m.tasks[row]
	return &t
}

// Cursor returns the current table cursor position.
func (m ListView) Cursor() int {
	return m.table.Cursor()
}

// SetSize resizes the list view.
func (m ListView) SetSize(w, h int) ListView {
	m.width = w
	m.height = h
	m.table.SetWidth(w)
	m.table.SetHeight(h)
	cols := buildColumns(m.cfg.Columns, w)
	m.table.SetColumns(cols)

	// Constrain header border to table width so it doesn't overshoot.
	active := m.activeStyle
	active.Header = active.Header.MaxWidth(w)
	m.activeStyle = active

	inact := m.inactStyle
	inact.Header = inact.Header.MaxWidth(w)
	m.inactStyle = inact

	// Re-apply whichever style is currently active.
	if m.table.Focused() {
		m.table.SetStyles(m.activeStyle)
	} else {
		m.table.SetStyles(m.inactStyle)
	}
	return m
}

// Update processes bubbletea messages.
func (m ListView) Update(msg tea.Msg) (ListView, tea.Cmd) {
	var cmd tea.Cmd
	m.table, cmd = m.table.Update(msg)
	return m, cmd
}

// View renders the list pane.
func (m ListView) View() string {
	// Clip to the configured width so the header border line never overshoots
	// and wraps, which would push the tab bar off-screen on small terminals.
	out := m.table.View()
	if m.width > 0 {
		out = lipgloss.NewStyle().MaxWidth(m.width).Render(out)
	}
	return out
}

// ---- helpers ----------------------------------------------------------------

func buildColumns(cols []string, totalWidth int) []table.Column {
	// Always include a narrow # column for numeric IDs
	idWidth := 5
	fixedTotal := idWidth
	fixedWidths := map[string]int{
		"status":   12,
		"priority": 8,
		"due_date": 12,
	}
	for _, c := range cols {
		if w, ok := fixedWidths[c]; ok {
			fixedTotal += w
		}
	}

	titleWidth := totalWidth - fixedTotal
	if titleWidth < 10 {
		titleWidth = 10
	}

	colDefs := []table.Column{{Title: "#", Width: idWidth}}
	for _, c := range cols {
		switch c {
		case "title":
			colDefs = append(colDefs, table.Column{Title: "Title", Width: titleWidth})
		case "status":
			colDefs = append(colDefs, table.Column{Title: "Status", Width: 12})
		case "priority":
			colDefs = append(colDefs, table.Column{Title: "Pri", Width: 8})
		case "due_date":
			colDefs = append(colDefs, table.Column{Title: "Due", Width: 12})
		}
	}
	return colDefs
}

func tasksToRows(tasks []domain.Task, cfg config.DisplayConfig, priorities []config.PriorityDef) []table.Row {
	rows := make([]table.Row, len(tasks))
	for i, t := range tasks {
		row := table.Row{fmt.Sprintf("%d", t.NumID)}
		for _, c := range cfg.Columns {
			switch c {
			case "title":
				row = append(row, t.Title)
			case "status":
				row = append(row, string(t.Status))
			case "priority":
				row = append(row, config.PriorityLabel(priorities, int(t.Priority)))
			case "due_date":
				if t.DueDate != nil {
					row = append(row, t.DueDate.Format("2006-01-02"))
				} else {
					row = append(row, "")
				}
			}
		}
		rows[i] = row
	}
	return rows
}
