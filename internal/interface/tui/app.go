package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/diptopandit/tork/internal/application"
	"github.com/diptopandit/tork/internal/domain"
	"github.com/diptopandit/tork/internal/infrastructure/config"
	"github.com/diptopandit/tork/internal/interface/tui/views"
)

// ---- async messages ---------------------------------------------------------

type tasksLoadedMsg struct {
	tasks []domain.Task
	err   error
}

type listsLoadedMsg struct {
	lists []domain.TaskList
	err   error
}

type defaultListCreatedMsg struct {
	list *domain.TaskList
	err  error
}

type taskCreatedMsg struct {
	task *domain.Task
	err  error
}

type taskUpdatedMsg struct {
	task *domain.Task
	err  error
}

type taskDeletedMsg struct {
	id  string
	err error
}

type updateAddedMsg struct {
	update *domain.Update
	err    error
}

type updatesLoadedMsg struct {
	updates []domain.Update
	taskID  string
	err     error
}

type listCreatedMsg struct {
	list *domain.TaskList
	err  error
}

type listUpdatedMsg struct {
	list *domain.TaskList
	err  error
}

type listDeletedMsg struct {
	id  string
	err error
}

const minWidth = 60
const minHeight = 15

// ---- Model ------------------------------------------------------------------

type Model struct {
	taskSvc    *application.TaskService
	listSvc    *application.ListService
	cfg        *config.Config
	keymap     KeyMap
	help       help.Model
	state      AppState
	listView   views.ListView
	detailView views.DetailView
	editView   views.EditView
	filterView views.FilterView
	width      int
	height     int
	quitting   bool

	listSwitchCursor int // cursor for list switcher overlay
	listSwitchMode   int // 0=select, 1=create, 2=rename, 3=confirmDelete, 4=confirmDelete2
	listSwitchInput  textinput.Model

	// Pane dimensions (snitch-style layout)
	leftWidth     int
	rightWidth    int
	contentHeight int
	inputBoxH     int // right pane: input box total height
	updatesH      int // right pane: updates viewport inner height (computed)
}

// NewModel assembles the root model.
func NewModel(
	taskSvc *application.TaskService,
	listSvc *application.ListService,
	cfg *config.Config,
) Model {
	km := NewKeyMap(cfg.Keybindings)
	h := help.New()

	lsInput := textinput.New()
	lsInput.Placeholder = "List name"
	lsInput.CharLimit = 100

	// Determine initial list from config
	activeListID := ""
	if cfg.LastList != "" {
		activeListID = cfg.LastList
	} else if cfg.DefaultList != "" {
		activeListID = cfg.DefaultList
	}

	return Model{
		taskSvc:         taskSvc,
		listSvc:         listSvc,
		cfg:             cfg,
		keymap:          km,
		help:            h,
		listView:        views.NewListView(cfg.Display, cfg.Theme),
		detailView:      views.NewDetailView(cfg.Display, cfg.Theme),
		editView:        views.NewEditView(cfg.Theme),
		filterView:      views.NewFilterView(cfg.Theme),
		listSwitchInput: lsInput,
		state: AppState{
			ActivePane:   PaneList,
			ActiveTab:    TabAll,
			ActiveListID: activeListID,
		},
	}
}

// Init kicks off the initial data fetch.
func (m Model) Init() tea.Cmd {
	return tea.Batch(m.loadTasks(), m.loadLists())
}

// Update is the root event handler.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.help.Width = msg.Width
		m.recalcLayout()
		cmd := m.syncDetailToSelection()
		return m, cmd

	case tasksLoadedMsg:
		if msg.err != nil {
			m.state.StatusMsg = "Error: " + msg.err.Error()
			return m, nil
		}
		m.state.Tasks = msg.tasks
		m.listView = m.listView.SetTasks(msg.tasks)
		cmd := m.syncDetailToSelection()
		return m, cmd

	case listsLoadedMsg:
		if msg.err != nil {
			m.state.StatusMsg = "Error: " + msg.err.Error()
			return m, nil
		}
		m.state.TaskLists = msg.lists
		if len(msg.lists) == 0 {
			return m, m.createDefaultList()
		}
		return m, nil

	case defaultListCreatedMsg:
		if msg.err != nil {
			m.state.StatusMsg = "Error: " + msg.err.Error()
			return m, nil
		}
		m.state.TaskLists = []domain.TaskList{*msg.list}
		return m, nil

	case taskCreatedMsg:
		if msg.err != nil {
			m.state.StatusMsg = "Error: " + msg.err.Error()
			return m, nil
		}
		m.state.ActiveOverlay = OverlayNone
		m.state.StatusMsg = "Task created"
		return m, m.loadTasks()

	case taskUpdatedMsg:
		if msg.err != nil {
			m.state.StatusMsg = "Error: " + msg.err.Error()
			return m, nil
		}
		m.state.ActiveOverlay = OverlayNone
		m.state.StatusMsg = "Task updated"
		return m, m.loadTasks()

	case taskDeletedMsg:
		if msg.err != nil {
			m.state.StatusMsg = "Error: " + msg.err.Error()
			return m, nil
		}
		m.state.StatusMsg = "Task deleted"
		return m, m.loadTasks()

	case updateAddedMsg:
		if msg.err != nil {
			m.state.StatusMsg = "Error: " + msg.err.Error()
			return m, nil
		}
		m.state.StatusMsg = "Update added"
		if m.state.SelectedTask != nil {
			return m, m.loadUpdates(m.state.SelectedTask.ID)
		}
		return m, nil

	case updatesLoadedMsg:
		if msg.err != nil {
			m.state.StatusMsg = "Error: " + msg.err.Error()
			return m, nil
		}
		if m.state.SelectedTask != nil && m.state.SelectedTask.ID == msg.taskID {
			m.state.SelectedTask.Updates = msg.updates
			m.detailView = m.detailView.SetTask(m.state.SelectedTask)
		}
		return m, nil

	case listCreatedMsg:
		if msg.err != nil {
			m.state.StatusMsg = "Error: " + msg.err.Error()
			return m, nil
		}
		m.state.StatusMsg = "List created: " + msg.list.Name
		m.state.ActiveListID = msg.list.ID
		m.state.ActiveOverlay = OverlayNone
		m.listSwitchMode = 0
		return m, tea.Batch(m.loadLists(), m.loadTasks())

	case listUpdatedMsg:
		if msg.err != nil {
			m.state.StatusMsg = "Error: " + msg.err.Error()
			return m, nil
		}
		m.state.StatusMsg = "List renamed: " + msg.list.Name
		m.state.ActiveOverlay = OverlayNone
		m.listSwitchMode = 0
		return m, m.loadLists()

	case listDeletedMsg:
		if msg.err != nil {
			m.state.StatusMsg = "Error: " + msg.err.Error()
			return m, nil
		}
		m.state.StatusMsg = "List deleted"
		m.state.ActiveOverlay = OverlayNone
		m.listSwitchMode = 0
		m.listSwitchCursor = 0
		// If the deleted list was active, switch to first available
		if m.state.ActiveListID == msg.id {
			m.state.ActiveListID = ""
		}
		return m, tea.Batch(m.loadLists(), m.loadTasks())

	case views.UpdateSubmittedMsg:
		return m, m.addUpdate(msg.TaskID, msg.Body)

	case tea.KeyMsg:
		if msg.String() == "ctrl+c" {
			m.quitting = true
			return m, tea.Quit
		}

		// Help overlay intercepts all keys
		if m.state.ActiveOverlay == OverlayHelp {
			if msg.Type == tea.KeyEsc || key.Matches(msg, m.keymap.Help) || msg.String() == "q" {
				m.state.ActiveOverlay = OverlayNone
			}
			return m, nil
		}

		// Edit overlay
		if m.state.ActiveOverlay == OverlayEdit {
			return m.handleEditKeys(msg)
		}

		// Filter overlay
		if m.state.ActiveOverlay == OverlayFilter {
			return m.handleFilterKeys(msg)
		}

		// List switcher overlay
		if m.state.ActiveOverlay == OverlayListSwitch {
			return m.handleListSwitchKeys(msg)
		}

		// Route to the active pane
		switch m.state.ActivePane {
		case PaneList:
			return m.handleListKeys(msg)
		case PaneDetail:
			return m.handleDetailKeys(msg)
		}
	}

	return m.forwardMsg(msg)
}

// View composes the full terminal layout (snitch-style split panes).
func (m Model) View() string {
	if m.quitting {
		return ""
	}

	if m.width < minWidth || m.height < minHeight {
		return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center,
			"Terminal too small.\nPlease resize to at least 80x20.")
	}

	// Header: app title + tab bar
	header := m.renderTabs()

	// Build panes
	leftPane := m.renderLeftPane()
	rightPane := m.renderRightPane()

	// Separator: vertical line
	sepLines := make([]string, m.contentHeight)
	for i := range sepLines {
		sepLines[i] = "\u2502"
	}
	sepCol := lipgloss.NewStyle().
		Foreground(lipgloss.Color(m.cfg.Theme.Inactive)).
		Render(strings.Join(sepLines, "\n"))

	body := lipgloss.JoinHorizontal(lipgloss.Top, leftPane, sepCol, rightPane)

	// Footer: status + help
	footer := m.renderFooter()

	full := header + "\n" + body + "\n" + footer

	// Overlay: edit modal
	if m.state.ActiveOverlay == OverlayEdit {
		editBox := m.renderEditOverlay()
		full = lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, editBox,
			lipgloss.WithWhitespaceChars(" "))
	}

	// Overlay: filter
	if m.state.ActiveOverlay == OverlayFilter {
		filterBox := m.renderFilterOverlay()
		full = lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, filterBox,
			lipgloss.WithWhitespaceChars(" "))
	}

	// Overlay: help
	if m.state.ActiveOverlay == OverlayHelp {
		helpBox := m.renderHelpOverlay()
		full = lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, helpBox,
			lipgloss.WithWhitespaceChars(" "))
	}

	// Overlay: list switcher
	if m.state.ActiveOverlay == OverlayListSwitch {
		lsBox := m.renderListSwitchOverlay()
		full = lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, lsBox,
			lipgloss.WithWhitespaceChars(" "))
	}

	return full
}

// ---- layout helpers ---------------------------------------------------------

func (m *Model) recalcLayout() {
	if m.width == 0 || m.height == 0 {
		return
	}
	// header(1) + "\n" + body(contentHeight) + "\n" + footer(1) = height
	m.contentHeight = m.height - 3
	if m.contentHeight < 8 {
		m.contentHeight = 8
	}

	// Horizontal: left(40%) + separator(1) + right(60%)
	m.leftWidth = m.width * 40 / 100
	m.rightWidth = m.width - m.leftWidth - 1

	// Left pane inner (border adds 2 each dimension)
	listInnerW := m.leftWidth - 2
	listInnerH := m.contentHeight - 2
	if listInnerW < 10 {
		listInnerW = 10
	}
	if listInnerH < 4 {
		listInnerH = 4
	}
	m.listView = m.listView.SetSize(listInnerW, listInnerH)

	// Right pane: 3 stacked boxes sharing contentHeight.
	// Input box: fixed 3 lines total (border=2 + 1 content).
	// Detail box & updates box: sized dynamically at render time.
	rightInnerW := m.rightWidth - 2
	if rightInnerW < 10 {
		rightInnerW = 10
	}
	m.inputBoxH = 3

	m.detailView = m.detailView.SetWidth(rightInnerW)

	// Compute updates viewport height from available space.
	// Detail box height is estimated (will be refined at render time).
	detailEstH := 8 // typical detail content lines + border
	updatesTotal := m.contentHeight - detailEstH - m.inputBoxH
	if updatesTotal < 4 {
		updatesTotal = 4
	}
	m.updatesH = updatesTotal - 3
	if m.updatesH < 1 {
		m.updatesH = 1
	}
	m.detailView = m.detailView.SetUpdatesHeight(m.updatesH)

	m.editView = m.editView.SetSize(m.width*60/100, m.height*70/100)
	m.filterView = m.filterView.SetSize(m.width*50/100, m.height*50/100)
}

func (m *Model) syncDetailToSelection() tea.Cmd {
	if t := m.listView.SelectedTask(); t != nil {
		m.state.SelectedTask = t
		m.detailView = m.detailView.SetTask(t)
		return m.loadUpdates(t.ID)
	}
	m.state.SelectedTask = nil
	m.detailView = m.detailView.SetTask(nil)
	return nil
}

// ---- render: header (tab bar) -----------------------------------------------

func (m Model) renderTabs() string {
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(m.cfg.Theme.Primary))
	title := titleStyle.Render("tork")

	tabs := []Tab{TabAll, TabTodo, TabInProgress, TabDone}
	var parts []string
	for _, t := range tabs {
		label := t.String()
		if t == m.state.ActiveTab {
			style := lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color(m.cfg.Theme.TextBright)).
				Background(lipgloss.Color(m.cfg.Theme.Primary)).
				Padding(0, 1)
			parts = append(parts, style.Render(label))
		} else {
			style := lipgloss.NewStyle().
				Foreground(lipgloss.Color(m.cfg.Theme.Secondary)).
				Padding(0, 1)
			parts = append(parts, style.Render(label))
		}
	}

	tabBar := lipgloss.JoinHorizontal(lipgloss.Top, parts...)
	leftPart := lipgloss.JoinHorizontal(lipgloss.Top, title, "  ", tabBar)

	// Show active list name right-aligned.
	listLabel := ""
	for _, l := range m.state.TaskLists {
		if l.ID == m.state.ActiveListID {
			listLabel = l.Name
			break
		}
	}
	if listLabel == "" && len(m.state.TaskLists) > 0 {
		listLabel = m.state.TaskLists[0].Name
	}
	rightPart := ""
	if listLabel != "" {
		rightPart = lipgloss.NewStyle().
			Foreground(lipgloss.Color(m.cfg.Theme.TextMuted)).
			Render("☰ " + listLabel)
	}

	gap := m.width - lipgloss.Width(leftPart) - lipgloss.Width(rightPart)
	if gap < 1 {
		gap = 1
	}
	header := leftPart + strings.Repeat(" ", gap) + rightPart
	// Truncate to terminal width so it doesn't wrap on narrow screens.
	return lipgloss.NewStyle().MaxWidth(m.width).Render(header)
}

// ---- render: left pane (task list) ------------------------------------------

func (m Model) renderLeftPane() string {
	borderColor := lipgloss.Color(m.cfg.Theme.Inactive)
	if m.state.ActivePane == PaneList {
		borderColor = lipgloss.Color(m.cfg.Theme.Primary)
	}

	style := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(borderColor).
		Width(m.leftWidth - 2).
		Height(m.contentHeight - 2)

	return style.Render(m.listView.View())
}

// ---- render: right pane (detail) --------------------------------------------

func (m Model) renderRightPane() string {
	borderColor := lipgloss.Color(m.cfg.Theme.Inactive)
	activeBorder := lipgloss.Color(m.cfg.Theme.Primary)
	if m.state.ActivePane == PaneDetail {
		borderColor = activeBorder
	}

	boxW := m.rightWidth - 2

	// Box 1: Task details — render content first to measure natural height
	detailContent := m.detailView.ViewDetails()
	detailInnerH := lipgloss.Height(detailContent)
	if detailInnerH < 1 {
		detailInnerH = 1
	}
	detailTotalH := detailInnerH + 2 // +2 for border

	// Box 3: Input — fixed height
	inputTotalH := m.inputBoxH // 3

	// Box 2: Updates — takes all remaining height
	updatesTotalH := m.contentHeight - detailTotalH - inputTotalH
	if updatesTotalH < 4 {
		// Shrink detail to make room rather than exceeding contentHeight.
		detailTotalH = m.contentHeight - 4 - inputTotalH
		if detailTotalH < 3 {
			detailTotalH = 3
		}
		detailInnerH = detailTotalH - 2
		if detailInnerH < 1 {
			detailInnerH = 1
		}
		updatesTotalH = m.contentHeight - detailTotalH - inputTotalH
	}
	updatesInnerH := updatesTotalH - 2
	if updatesInnerH < 1 {
		updatesInnerH = 1
	}

	detailStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(borderColor).
		Width(boxW).
		Height(detailInnerH)
	detailBox := detailStyle.Render(detailContent)

	updatesStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(borderColor).
		Width(boxW).
		Height(updatesInnerH)
	updatesBox := updatesStyle.Render(m.detailView.ViewUpdates())

	inputBorder := borderColor
	if m.detailView.InputFocused() {
		inputBorder = activeBorder
	}
	inputStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(inputBorder).
		Width(boxW).
		Height(m.inputBoxH - 2)
	inputBox := inputStyle.Render(m.detailView.ViewInput())

	// Wrap in fixed-height container so right pane always matches left pane.
	pane := lipgloss.JoinVertical(lipgloss.Left, detailBox, updatesBox, inputBox)
	return lipgloss.NewStyle().Height(m.contentHeight).Render(pane)
}

// ---- render: footer ---------------------------------------------------------

func (m Model) renderFooter() string {
	status := m.state.StatusMsg
	if status == "" {
		count := len(m.state.Tasks)
		status = fmt.Sprintf("%d task(s)", count)
	}
	left := lipgloss.NewStyle().Foreground(lipgloss.Color(m.cfg.Theme.Secondary)).
		Render(status)

	helpLine := m.help.ShortHelpView(m.keymap.ShortHelp())

	gap := m.width - lipgloss.Width(left) - lipgloss.Width(helpLine)
	if gap < 1 {
		gap = 1
	}
	return left + strings.Repeat(" ", gap) + helpLine
}

// ---- render: overlays -------------------------------------------------------

func (m Model) renderEditOverlay() string {
	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(m.cfg.Theme.Warning)).
		Padding(1, 2)
	return boxStyle.Render(m.editView.View())
}

func (m Model) renderFilterOverlay() string {
	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(m.cfg.Theme.Secondary)).
		Padding(1, 2)
	return boxStyle.Render(m.filterView.View())
}

func (m Model) renderHelpOverlay() string {
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(m.cfg.Theme.Primary))
	sectionStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(m.cfg.Theme.Warning))
	keyStyle := lipgloss.NewStyle().Bold(true).Width(18).Foreground(lipgloss.Color(m.cfg.Theme.Accent))
	descStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(m.cfg.Theme.Text))

	var b strings.Builder

	b.WriteString(titleStyle.Render("Keybindings") + "\n\n")

	b.WriteString(sectionStyle.Render("Navigation") + "\n")
	for _, bind := range []struct{ k, desc string }{
		{"j/\u2193  k/\u2191", "Move down / up"},
		{"h/\u2190 l/\u2192", "Focus left / right pane"},
		{"Tab / Shift+Tab", "Next / previous status tab"},
	} {
		b.WriteString("  " + keyStyle.Render(bind.k) + descStyle.Render(bind.desc) + "\n")
	}

	b.WriteString("\n" + sectionStyle.Render("Actions") + "\n")
	for _, bind := range []struct{ k, desc string }{
		{"n", "New task"},
		{"e", "Edit task"},
		{"s", "Cycle status (todo→in_progress→done→cancelled)"},
		{"x", "Mark done"},
		{"d", "Delete task"},
		{"u", "Add update (enter to submit)"},
		{"/", "Search / filter"},
		{"Enter", "Select / view detail"},
	} {
		b.WriteString("  " + keyStyle.Render(bind.k) + descStyle.Render(bind.desc) + "\n")
	}

	b.WriteString("\n" + sectionStyle.Render("General") + "\n")
	for _, bind := range []struct{ k, desc string }{
		{"L", "Switch task list (n: new, r: rename, d: delete)"},
		{"?", "Toggle this help"},
		{"q / Ctrl+C", "Quit"},
		{"Esc", "Close overlay / back"},
	} {
		b.WriteString("  " + keyStyle.Render(bind.k) + descStyle.Render(bind.desc) + "\n")
	}

	b.WriteString("\n" + lipgloss.NewStyle().Foreground(lipgloss.Color(m.cfg.Theme.Secondary)).
		Render("Press Esc or ? to close"))

	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(m.cfg.Theme.Primary)).
		Padding(1, 2)

	return boxStyle.Render(b.String())
}

// ---- key routing ------------------------------------------------------------

func (m Model) handleListKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	prevSelected := m.listView.Cursor()

	switch {
	case isKey(msg, m.keymap.Quit):
		m.quitting = true
		return m, tea.Quit

	case isKey(msg, m.keymap.Help):
		m.state.ActiveOverlay = OverlayHelp
		return m, nil

	case isKey(msg, m.keymap.Right):
		m.state.ActivePane = PaneDetail
		return m, nil

	case isKey(msg, m.keymap.Tab):
		m.state.ActiveTab = Tab((int(m.state.ActiveTab) + 1) % TabCount)
		m.state.StatusMsg = ""
		return m, m.loadTasks()

	case isKey(msg, m.keymap.ShiftTab):
		m.state.ActiveTab = Tab((int(m.state.ActiveTab) - 1 + TabCount) % TabCount)
		m.state.StatusMsg = ""
		return m, m.loadTasks()

	case isKey(msg, m.keymap.ListSwitch):
		m.state.ActiveOverlay = OverlayListSwitch
		return m, nil

	case isKey(msg, m.keymap.Comment):
		if t := m.listView.SelectedTask(); t != nil {
			m.state.SelectedTask = t
			m.detailView = m.detailView.FocusInput()
			m.state.ActivePane = PaneDetail
		}
		return m, nil

	case isKey(msg, m.keymap.New):
		m.editView = m.editView.SetTask(nil)
		m.state.ActiveOverlay = OverlayEdit
		return m, nil

	case isKey(msg, m.keymap.Edit):
		if t := m.listView.SelectedTask(); t != nil {
			m.editView = m.editView.SetTask(t)
			m.state.ActiveOverlay = OverlayEdit
		}
		return m, nil

	case isKey(msg, m.keymap.Delete):
		if t := m.listView.SelectedTask(); t != nil {
			return m, m.deleteTask(t.ID)
		}
		return m, nil

	case isKey(msg, m.keymap.Done):
		if t := m.listView.SelectedTask(); t != nil {
			return m, m.markDone(t)
		}
		return m, nil

	case isKey(msg, m.keymap.Status):
		if t := m.listView.SelectedTask(); t != nil {
			return m, m.cycleStatus(t)
		}
		return m, nil

	case isKey(msg, m.keymap.Search):
		m.filterView = m.filterView.FocusSearch()
		m.state.ActiveOverlay = OverlayFilter
		return m, nil

	case isKey(msg, m.keymap.Select):
		if t := m.listView.SelectedTask(); t != nil {
			m.state.SelectedTask = t
			m.detailView = m.detailView.SetTask(t)
			m.state.ActivePane = PaneDetail
		}
		return m, nil
	}

	// Delegate navigation to list view
	var cmd tea.Cmd
	m.listView, cmd = m.listView.Update(msg)

	// If selection changed, sync detail panel
	if m.listView.Cursor() != prevSelected {
		syncCmd := m.syncDetailToSelection()
		return m, tea.Batch(cmd, syncCmd)
	}

	return m, cmd
}

func (m Model) handleDetailKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// When the update input is focused, forward everything to detail view
	if m.detailView.InputFocused() {
		var cmd tea.Cmd
		m.detailView, cmd = m.detailView.Update(msg)
		return m, cmd
	}

	switch {
	case isKey(msg, m.keymap.Left), msg.Type == tea.KeyEsc:
		m.state.ActivePane = PaneList
		return m, nil

	case isKey(msg, m.keymap.Quit):
		m.quitting = true
		return m, tea.Quit

	case isKey(msg, m.keymap.Help):
		m.state.ActiveOverlay = OverlayHelp
		return m, nil

	case isKey(msg, m.keymap.Comment):
		if m.state.SelectedTask != nil {
			m.detailView = m.detailView.FocusInput()
		}
		return m, nil

	case isKey(msg, m.keymap.Edit):
		if t := m.state.SelectedTask; t != nil {
			m.editView = m.editView.SetTask(t)
			m.state.ActiveOverlay = OverlayEdit
		}
		return m, nil

	case isKey(msg, m.keymap.Done):
		if t := m.state.SelectedTask; t != nil {
			return m, m.markDone(t)
		}
		return m, nil

	case isKey(msg, m.keymap.Status):
		if t := m.state.SelectedTask; t != nil {
			return m, m.cycleStatus(t)
		}
		return m, nil
	}

	var cmd tea.Cmd
	m.detailView, cmd = m.detailView.Update(msg)
	return m, cmd
}

func (m Model) handleEditKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if msg.Type == tea.KeyEsc {
		m.state.ActiveOverlay = OverlayNone
		return m, nil
	}

	var cmd tea.Cmd
	m.editView, cmd = m.editView.Update(msg)

	if saved := m.editView.SavedTask(); saved != nil {
		m.editView = m.editView.ClearSaved()
		if saved.ID == "" {
			listID := m.state.ActiveListID
			if listID == "" && len(m.state.TaskLists) > 0 {
				listID = m.state.TaskLists[0].ID
			}
			if listID != "" {
				return m, m.createTask(saved, listID)
			}
			m.state.StatusMsg = "No task list available"
			m.state.ActiveOverlay = OverlayNone
		} else {
			return m, m.updateTask(saved)
		}
	}
	return m, cmd
}

func (m Model) handleFilterKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if msg.Type == tea.KeyEsc {
		m.state.ActiveOverlay = OverlayNone
		return m, nil
	}

	var cmd tea.Cmd
	m.filterView, cmd = m.filterView.Update(msg)

	if m.filterView.Applied() {
		f := m.filterView.Filter()
		m.filterView = m.filterView.ClearApplied()
		m.state.Filter = f
		m.state.ActiveOverlay = OverlayNone
		return m, m.loadTasksWithFilter(f)
	}
	return m, cmd
}

func (m Model) forwardMsg(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	switch m.state.ActivePane {
	case PaneDetail:
		m.detailView, cmd = m.detailView.Update(msg)
	default:
		m.listView, cmd = m.listView.Update(msg)
	}
	return m, cmd
}

// ---- async commands ---------------------------------------------------------

func (m Model) loadTasks() tea.Cmd {
	f := m.state.Filter
	if m.state.ActiveListID != "" {
		f.ListIDs = []string{m.state.ActiveListID}
	}
	switch m.state.ActiveTab {
	case TabTodo:
		f.Statuses = []domain.Status{domain.StatusTodo}
	case TabInProgress:
		f.Statuses = []domain.Status{domain.StatusInProgress}
	case TabDone:
		f.Statuses = []domain.Status{domain.StatusDone}
	default:
		f.Statuses = nil
	}
	return m.loadTasksWithFilter(f)
}

func (m Model) loadTasksWithFilter(f domain.TaskFilter) tea.Cmd {
	svc := m.taskSvc
	return func() tea.Msg {
		tasks, err := svc.ListTasks(f)
		return tasksLoadedMsg{tasks: tasks, err: err}
	}
}

func (m Model) loadLists() tea.Cmd {
	svc := m.listSvc
	return func() tea.Msg {
		lists, err := svc.GetAllLists()
		return listsLoadedMsg{lists: lists, err: err}
	}
}

func (m Model) createDefaultList() tea.Cmd {
	svc := m.listSvc
	return func() tea.Msg {
		list, err := svc.CreateList("Default", nil)
		return defaultListCreatedMsg{list: list, err: err}
	}
}

func (m Model) deleteTask(id string) tea.Cmd {
	svc := m.taskSvc
	return func() tea.Msg {
		err := svc.DeleteTask(id)
		return taskDeletedMsg{id: id, err: err}
	}
}

func (m Model) markDone(t *domain.Task) tea.Cmd {
	svc := m.taskSvc
	status := domain.StatusDone
	return func() tea.Msg {
		updated, err := svc.UpdateTask(application.UpdateTaskInput{
			ID:     t.ID,
			Status: &status,
		})
		return taskUpdatedMsg{task: updated, err: err}
	}
}

func (m Model) cycleStatus(t *domain.Task) tea.Cmd {
	svc := m.taskSvc
	order := []domain.Status{
		domain.StatusTodo,
		domain.StatusInProgress,
		domain.StatusDone,
		domain.StatusCancelled,
	}
	next := domain.StatusTodo
	for i, s := range order {
		if s == t.Status {
			next = order[(i+1)%len(order)]
			break
		}
	}
	return func() tea.Msg {
		updated, err := svc.UpdateTask(application.UpdateTaskInput{
			ID:     t.ID,
			Status: &next,
		})
		return taskUpdatedMsg{task: updated, err: err}
	}
}

func (m Model) createTask(t *domain.Task, listID string) tea.Cmd {
	svc := m.taskSvc
	return func() tea.Msg {
		created, err := svc.CreateTask(application.CreateTaskInput{
			ListID:      listID,
			Title:       t.Title,
			Description: t.Description,
			Priority:    t.Priority,
			DueDate:     t.DueDate,
			Tags:        t.Tags,
		})
		return taskCreatedMsg{task: created, err: err}
	}
}

func (m Model) updateTask(t *domain.Task) tea.Cmd {
	svc := m.taskSvc
	id := t.ID
	title := t.Title
	desc := t.Description
	status := t.Status
	pri := t.Priority
	due := t.DueDate
	tags := t.Tags
	return func() tea.Msg {
		updated, err := svc.UpdateTask(application.UpdateTaskInput{
			ID:          id,
			Title:       &title,
			Description: &desc,
			Status:      &status,
			Priority:    &pri,
			DueDate:     due,
			Tags:        tags,
		})
		return taskUpdatedMsg{task: updated, err: err}
	}
}

func (m Model) addUpdate(taskID, body string) tea.Cmd {
	svc := m.taskSvc
	return func() tea.Msg {
		u, err := svc.AddUpdate(taskID, body)
		return updateAddedMsg{update: u, err: err}
	}
}

func (m Model) loadUpdates(taskID string) tea.Cmd {
	svc := m.taskSvc
	return func() tea.Msg {
		updates, err := svc.GetUpdates(taskID)
		return updatesLoadedMsg{updates: updates, taskID: taskID, err: err}
	}
}

func (m Model) createList(name string) tea.Cmd {
	svc := m.listSvc
	return func() tea.Msg {
		list, err := svc.CreateList(name, nil)
		return listCreatedMsg{list: list, err: err}
	}
}

func (m Model) renameList(id, name string) tea.Cmd {
	svc := m.listSvc
	return func() tea.Msg {
		list, err := svc.UpdateList(id, name)
		return listUpdatedMsg{list: list, err: err}
	}
}

func (m Model) deleteList(id string) tea.Cmd {
	svc := m.listSvc
	return func() tea.Msg {
		err := svc.DeleteList(id)
		return listDeletedMsg{id: id, err: err}
	}
}

// ---- overlay key handlers ---------------------------------------------------

func (m Model) handleListSwitchKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Mode 1: creating a new list — text input active
	if m.listSwitchMode == 1 {
		switch msg.Type {
		case tea.KeyEsc:
			m.listSwitchMode = 0
			m.listSwitchInput.Blur()
			return m, nil
		case tea.KeyEnter:
			name := strings.TrimSpace(m.listSwitchInput.Value())
			if name != "" {
				m.listSwitchInput.Blur()
				return m, m.createList(name)
			}
			return m, nil
		}
		var cmd tea.Cmd
		m.listSwitchInput, cmd = m.listSwitchInput.Update(msg)
		return m, cmd
	}

	// Mode 2: renaming a list — text input active
	if m.listSwitchMode == 2 {
		switch msg.Type {
		case tea.KeyEsc:
			m.listSwitchMode = 0
			m.listSwitchInput.Blur()
			return m, nil
		case tea.KeyEnter:
			name := strings.TrimSpace(m.listSwitchInput.Value())
			if name != "" && m.listSwitchCursor < len(m.state.TaskLists) {
				listID := m.state.TaskLists[m.listSwitchCursor].ID
				m.listSwitchInput.Blur()
				return m, m.renameList(listID, name)
			}
			return m, nil
		}
		var cmd tea.Cmd
		m.listSwitchInput, cmd = m.listSwitchInput.Update(msg)
		return m, cmd
	}

	// Mode 3: first delete confirmation
	if m.listSwitchMode == 3 {
		switch {
		case msg.String() == "y" || msg.String() == "Y":
			m.listSwitchMode = 4 // second confirmation
			return m, nil
		default:
			m.listSwitchMode = 0
			return m, nil
		}
	}

	// Mode 4: second delete confirmation
	if m.listSwitchMode == 4 {
		switch {
		case msg.String() == "y" || msg.String() == "Y":
			if m.listSwitchCursor < len(m.state.TaskLists) {
				listID := m.state.TaskLists[m.listSwitchCursor].ID
				return m, m.deleteList(listID)
			}
			m.listSwitchMode = 0
			return m, nil
		default:
			m.listSwitchMode = 0
			return m, nil
		}
	}

	// Mode 0: selection mode
	switch {
	case msg.Type == tea.KeyEsc:
		m.state.ActiveOverlay = OverlayNone
		m.listSwitchMode = 0
		return m, nil

	case msg.String() == "j" || msg.Type == tea.KeyDown:
		if m.listSwitchCursor < len(m.state.TaskLists)-1 {
			m.listSwitchCursor++
		}
		return m, nil

	case msg.String() == "k" || msg.Type == tea.KeyUp:
		if m.listSwitchCursor > 0 {
			m.listSwitchCursor--
		}
		return m, nil

	case msg.Type == tea.KeyEnter:
		if m.listSwitchCursor >= 0 && m.listSwitchCursor < len(m.state.TaskLists) {
			selected := m.state.TaskLists[m.listSwitchCursor]
			m.state.ActiveListID = selected.ID
			m.state.StatusMsg = ""
			m.cfg.LastList = selected.ID
			_ = config.Save(m.cfg)
		}
		m.state.ActiveOverlay = OverlayNone
		m.listSwitchMode = 0
		return m, m.loadTasks()

	case msg.String() == "n" || msg.String() == "N":
		m.listSwitchMode = 1
		m.listSwitchInput.SetValue("")
		m.listSwitchInput.Focus()
		return m, nil

	case msg.String() == "r" || msg.String() == "R":
		if m.listSwitchCursor < len(m.state.TaskLists) {
			m.listSwitchMode = 2
			m.listSwitchInput.SetValue(m.state.TaskLists[m.listSwitchCursor].Name)
			m.listSwitchInput.Focus()
		}
		return m, nil

	case msg.String() == "d" || msg.String() == "D":
		if m.listSwitchCursor < len(m.state.TaskLists) {
			m.listSwitchMode = 3
		}
		return m, nil
	}
	return m, nil
}

// ---- overlay renderers ------------------------------------------------------

func (m Model) renderListSwitchOverlay() string {
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(m.cfg.Theme.Primary))
	cursorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(m.cfg.Theme.TextBright)).Background(lipgloss.Color(m.cfg.Theme.Primary)).Bold(true).Padding(0, 1)
	normalStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(m.cfg.Theme.Text)).Padding(0, 1)
	hintStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(m.cfg.Theme.Secondary))
	dangerStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(m.cfg.Theme.Danger))

	var b strings.Builder

	switch m.listSwitchMode {
	case 1: // create
		b.WriteString(titleStyle.Render("Create New List") + "\n\n")
		b.WriteString(m.listSwitchInput.View() + "\n\n")
		b.WriteString(hintStyle.Render("Enter: create  Esc: cancel"))

	case 2: // rename
		b.WriteString(titleStyle.Render("Rename List") + "\n\n")
		b.WriteString(m.listSwitchInput.View() + "\n\n")
		b.WriteString(hintStyle.Render("Enter: save  Esc: cancel"))

	case 3: // first delete confirmation
		listName := ""
		if m.listSwitchCursor < len(m.state.TaskLists) {
			listName = m.state.TaskLists[m.listSwitchCursor].Name
		}
		b.WriteString(dangerStyle.Render("Delete List") + "\n\n")
		b.WriteString(normalStyle.Render("Delete \""+listName+"\"?") + "\n")
		b.WriteString(normalStyle.Render("All tasks in this list will be deleted.") + "\n\n")
		b.WriteString(dangerStyle.Render("Press y to confirm, any other key to cancel"))

	case 4: // second delete confirmation
		listName := ""
		if m.listSwitchCursor < len(m.state.TaskLists) {
			listName = m.state.TaskLists[m.listSwitchCursor].Name
		}
		b.WriteString(dangerStyle.Render("⚠ FINAL CONFIRMATION") + "\n\n")
		b.WriteString(normalStyle.Render("This will permanently delete \""+listName+"\"") + "\n")
		b.WriteString(normalStyle.Render("and ALL its tasks. This cannot be undone.") + "\n\n")
		b.WriteString(dangerStyle.Render("Press y to permanently delete, any other key to cancel"))

	default: // 0: select mode
		b.WriteString(titleStyle.Render("Task Lists") + "\n\n")

		for i, l := range m.state.TaskLists {
			label := l.Name
			if l.ID == m.state.ActiveListID {
				label += " *"
			}
			if i == m.listSwitchCursor {
				b.WriteString(cursorStyle.Render("> "+label) + "\n")
			} else {
				b.WriteString(normalStyle.Render("  "+label) + "\n")
			}
		}

		if len(m.state.TaskLists) == 0 {
			b.WriteString(normalStyle.Render("  No lists available") + "\n")
		}

		b.WriteString("\n" + hintStyle.Render("j/k: navigate  Enter: select  n: new  r: rename  d: delete  Esc: close"))
	}

	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(m.cfg.Theme.Primary)).
		Padding(1, 2).
		Width(50)
	return boxStyle.Render(b.String())
}
