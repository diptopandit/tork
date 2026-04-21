package tui

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/diptopandit/tork/internal/application"
	"github.com/diptopandit/tork/internal/domain"
	"github.com/diptopandit/tork/internal/infrastructure/bootstrap"
	"github.com/diptopandit/tork/internal/infrastructure/config"
	"github.com/diptopandit/tork/internal/interface/tui/styles"
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

// ConnectFunc creates services for a given remote (or "local").
// Provided by main() as a closure over the config.
type ConnectFunc func(remoteName, password string) (*bootstrap.Services, error)

// remoteConnectedMsg is sent after an async remote connection attempt.
type remoteConnectedMsg struct {
	svc        *bootstrap.Services
	remoteName string
	err        error
}

// promptPasswordMsg triggers the password overlay for a pending remote.
type promptPasswordMsg struct {
	remoteName string
	label      string
}

// ---- Model ------------------------------------------------------------------

type Model struct {
	taskSvc    *application.TaskService
	listSvc    *application.ListService
	cfg        *config.Config
	styles     styles.Styles
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

	// Theme picker state
	themePickerView   views.ThemePickerView
	originalThemeName string        // theme before opening picker (for Esc revert)
	originalStyles    styles.Styles // styles before opening picker

	// Sort state
	sortView views.SortView

	// Help overlay state
	helpViewport viewport.Model

	// Remote picker state
	remotePickerCursor  int
	remotePickerChoices []string // "local", remote names...
	remotePickerLabels  []string
	remoteAddMode       bool // true when adding a new remote
	remoteAddStep       int  // 0=name, 1=host, 2=port, 3=database, 4=username
	remoteAddInput      textinput.Model
	remoteAddName       string // collected name so far
	remoteAddHost       string
	remoteAddPort       string
	remoteAddDB         string

	// Password overlay state
	passwordInput     textinput.Model
	pendingRemoteName string // remote name awaiting password
	passwordLabel     string // display label for the remote

	// Connection management
	connectFunc ConnectFunc                // provided by main()
	currentDB   interface{ Close() error } // current DB handle for cleanup

	// Pane dimensions (snitch-style layout)
	leftWidth     int
	rightWidth    int
	contentHeight int
	inputBoxH     int // right pane: input box total height (border included)
	detailBoxH    int // right pane: detail box total height (border included)
	updatesBoxH   int // right pane: updates box total height (border included)

	tabOrder []string // configurable tab names (status strings + "all")
}

// NewModel assembles the root model.
func NewModel(
	taskSvc *application.TaskService,
	listSvc *application.ListService,
	cfg *config.Config,
	connectFunc ConnectFunc,
	currentDB interface{ Close() error },
) Model {
	km := NewKeyMap(cfg.Keybindings)
	h := help.New()
	s := styles.NewStyles(cfg.Theme)

	lsInput := textinput.New()
	lsInput.Placeholder = "List name"
	lsInput.CharLimit = 100

	// Remote add input.
	raInput := textinput.New()
	raInput.Placeholder = "remote name"
	raInput.CharLimit = 100

	// Password input for remote connections.
	pwInput := textinput.New()
	pwInput.Placeholder = "password"
	pwInput.EchoMode = textinput.EchoPassword
	pwInput.CharLimit = 256

	// Determine initial list from config
	activeListID := ""
	if cfg.LastList != "" {
		activeListID = cfg.LastList
	} else if cfg.DefaultList != "" {
		activeListID = cfg.DefaultList
	}

	// Resolve tab order from config
	var tabOrder []string
	for _, name := range cfg.Display.TabOrder {
		if ValidTab(cfg.Statuses, name) {
			tabOrder = append(tabOrder, name)
		}
	}
	if len(tabOrder) == 0 {
		// Default: all configured statuses + "all"
		for _, s := range cfg.Statuses {
			tabOrder = append(tabOrder, s.Name)
		}
		tabOrder = append(tabOrder, "all")
	}

	// Resolve default tab index
	defaultTabIdx := 0
	if cfg.Display.DefaultTab != "" {
		for i, name := range tabOrder {
			if name == cfg.Display.DefaultTab {
				defaultTabIdx = i
				break
			}
		}
	}

	return Model{
		taskSvc:         taskSvc,
		listSvc:         listSvc,
		cfg:             cfg,
		styles:          s,
		keymap:          km,
		help:            h,
		listView:        views.NewListView(cfg.Display, s, cfg.Priorities),
		detailView:      views.NewDetailView(cfg.Display, s, cfg.Priorities, cfg.Statuses),
		editView:        views.NewEditView(s, cfg.Statuses, cfg.Priorities),
		filterView:      views.NewFilterView(s, cfg.Statuses, cfg.Priorities),
		listSwitchInput: lsInput,
		remoteAddInput:  raInput,
		passwordInput:   pwInput,
		connectFunc:     connectFunc,
		currentDB:       currentDB,
		tabOrder:        tabOrder,
		state: AppState{
			ActivePane:   PaneList,
			ActiveTab:    defaultTabIdx,
			ActiveListID: activeListID,
		},
	}
}

// Init kicks off the initial data fetch. If a remote was configured as last
// used, it sends a message to trigger the password overlay.
func (m Model) Init() tea.Cmd {
	cmds := []tea.Cmd{m.loadTasks(), m.loadLists()}

	// If a remote is configured, schedule password prompt via message.
	if rc := m.cfg.ActiveRemote(m.cfg.LastRemote); rc != nil && m.connectFunc != nil {
		label := rc.Label()
		name := m.cfg.LastRemote
		cmds = append(cmds, func() tea.Msg {
			return promptPasswordMsg{remoteName: name, label: label}
		})
	}

	return tea.Batch(cmds...)
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

	case remoteConnectedMsg:
		if msg.err != nil {
			m.state.StatusMsg = "Connection failed: " + msg.err.Error() + " (using local)"
			m.pendingRemoteName = ""
			m.cfg.LastRemote = "local"
			_ = config.Save(m.cfg)
			return m, nil
		}
		// Close previous DB and swap services.
		if m.currentDB != nil {
			m.currentDB.Close()
		}
		m.taskSvc = msg.svc.TaskSvc
		m.listSvc = msg.svc.ListSvc
		m.currentDB = msg.svc.DB
		m.cfg.LastRemote = msg.remoteName
		_ = config.Save(m.cfg)
		m.state.StatusMsg = "Connected to " + msg.remoteName
		m.pendingRemoteName = ""
		return m, tea.Batch(m.loadTasks(), m.loadLists())

	case promptPasswordMsg:
		m.pendingRemoteName = msg.remoteName
		m.passwordLabel = msg.label
		m.passwordInput.Reset()
		m.passwordInput.Focus()
		m.state.ActiveOverlay = OverlayPasswordPrompt
		return m, textinput.Blink

	case tea.KeyMsg:
		if msg.String() == "ctrl+c" {
			m.quitting = true
			return m, tea.Quit
		}

		// Help overlay intercepts all keys
		if m.state.ActiveOverlay == OverlayHelp {
			if msg.Type == tea.KeyEsc || key.Matches(msg, m.keymap.Help) || msg.String() == "q" {
				m.state.ActiveOverlay = OverlayNone
				return m, nil
			}
			// Forward to the help viewport for scrolling.
			var cmd tea.Cmd
			m.helpViewport, cmd = m.helpViewport.Update(msg)
			return m, cmd
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

		// Theme picker overlay
		if m.state.ActiveOverlay == OverlayThemePicker {
			return m.handleThemePickerKeys(msg)
		}

		// Sort overlay
		if m.state.ActiveOverlay == OverlaySort {
			return m.handleSortKeys(msg)
		}

		// Remote picker overlay
		if m.state.ActiveOverlay == OverlayRemotePicker {
			return m.handleRemotePickerKeys(msg)
		}

		// Password overlay
		if m.state.ActiveOverlay == OverlayPasswordPrompt {
			return m.handlePasswordKeys(msg)
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
	sepCol := m.styles.Separator.
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

	// Overlay: theme picker
	if m.state.ActiveOverlay == OverlayThemePicker {
		tpBox := m.renderThemePickerOverlay()
		full = lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, tpBox,
			lipgloss.WithWhitespaceChars(" "))
	}

	// Overlay: sort
	if m.state.ActiveOverlay == OverlaySort {
		sortBox := m.renderSortOverlay()
		full = lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, sortBox,
			lipgloss.WithWhitespaceChars(" "))
	}

	// Overlay: remote picker
	if m.state.ActiveOverlay == OverlayRemotePicker {
		rpBox := m.renderRemotePickerOverlay()
		full = lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, rpBox,
			lipgloss.WithWhitespaceChars(" "))
	}

	// Overlay: password prompt
	if m.state.ActiveOverlay == OverlayPasswordPrompt {
		pwBox := m.renderPasswordOverlay()
		full = lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, pwBox,
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

	// Right pane: 3 stacked boxes sharing contentHeight, all fixed height.
	// Input box: fixed 3 lines total (border=2 + 1 content).
	// Detail box: fixed ~25% of remaining.
	// Updates box: whatever is left.
	rightInnerW := m.rightWidth - 2
	if rightInnerW < 10 {
		rightInnerW = 10
	}
	m.inputBoxH = 3

	m.detailView = m.detailView.SetWidth(rightInnerW)

	// Compute fixed heights for detail and updates boxes.
	remaining := m.contentHeight - m.inputBoxH // total for detail + updates
	detailTotal := remaining * 55 / 100        // ~55% for detail
	if detailTotal < 5 {
		detailTotal = 5
	}
	updatesTotal := remaining - detailTotal
	if updatesTotal < 4 {
		updatesTotal = 4
		detailTotal = remaining - updatesTotal
		if detailTotal < 3 {
			detailTotal = 3
		}
	}
	m.detailBoxH = detailTotal
	m.updatesBoxH = updatesTotal

	// Inner heights = total - 2 for border
	detailInnerH := detailTotal - 2
	if detailInnerH < 1 {
		detailInnerH = 1
	}
	updatesInnerH := updatesTotal - 2
	if updatesInnerH < 1 {
		updatesInnerH = 1
	}

	m.detailView = m.detailView.SetDetailHeight(detailInnerH)
	m.detailView = m.detailView.SetUpdatesHeight(updatesInnerH)

	m.editView = m.editView.SetSize(m.width*60/100, m.height*70/100)
	m.filterView = m.filterView.SetSize(m.width*50/100, m.height*50/100)
}

func (m Model) nextTab() int {
	return (m.state.ActiveTab + 1) % len(m.tabOrder)
}

func (m Model) prevTab() int {
	return (m.state.ActiveTab - 1 + len(m.tabOrder)) % len(m.tabOrder)
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
	title := m.styles.AppTitle.Render("tork")

	tabs := m.tabOrder
	var parts []string
	for i, name := range tabs {
		label := TabLabel(m.cfg.Statuses, name)
		if i == m.state.ActiveTab {
			parts = append(parts, m.styles.TabActive.Render(label))
		} else {
			parts = append(parts, m.styles.TabInactive.Render(label))
		}
	}

	tabBar := lipgloss.JoinHorizontal(lipgloss.Top, parts...)
	leftPart := lipgloss.JoinHorizontal(lipgloss.Top, title, "  ", tabBar)

	// Show remote > list breadcrumb right-aligned.
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

	// Build breadcrumb: remote › list (or local › list).
	var breadcrumb string
	remote := m.cfg.LastRemote
	if remote != "" && remote != "local" {
		breadcrumb = remote
	} else {
		breadcrumb = "local"
	}
	if listLabel != "" {
		breadcrumb += " › " + listLabel
	}

	rightPart := ""
	if breadcrumb != "" {
		rightPart = m.styles.ListLabel.Render("☰ " + breadcrumb)
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
	style := m.styles.PaneBorderUnfocused
	if m.state.ActivePane == PaneList {
		style = m.styles.PaneBorderFocused
	}
	style = style.
		Width(m.leftWidth - 2).
		Height(m.contentHeight - 2)

	return style.Render(m.listView.View())
}

// ---- render: right pane (detail) --------------------------------------------

func (m Model) renderRightPane() string {
	unfocused := m.styles.PaneBorderUnfocused
	focused := m.styles.PaneBorderFocused
	isRight := m.state.ActivePane == PaneDetail

	boxW := m.rightWidth - 2

	// All three boxes use pre-computed fixed heights from recalcLayout.
	detailInnerH := m.detailBoxH - 2
	if detailInnerH < 1 {
		detailInnerH = 1
	}
	updatesInnerH := m.updatesBoxH - 2
	if updatesInnerH < 1 {
		updatesInnerH = 1
	}

	// Detail box: focused when right pane active and DetailFocus == FocusDetails
	detailBorder := unfocused
	if isRight && m.state.DetailFocus == FocusDetails {
		detailBorder = focused
	}
	detailStyle := detailBorder.
		Width(boxW).
		Height(detailInnerH)
	detailBox := detailStyle.Render(m.detailView.ViewDetails())

	// Updates box: focused when right pane active and DetailFocus == FocusUpdates
	updatesBorder := unfocused
	if isRight && m.state.DetailFocus == FocusUpdates {
		updatesBorder = focused
	}
	updatesStyle := updatesBorder.
		Width(boxW).
		Height(updatesInnerH)
	updatesBox := updatesStyle.Render(m.detailView.ViewUpdates())

	// Input box: focused when input has focus
	inputBorder := unfocused
	if m.detailView.InputFocused() {
		inputBorder = focused
	}
	inputStyle := inputBorder.
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
	left := m.styles.FooterText.Render(status)

	helpLine := m.help.ShortHelpView(m.keymap.ShortHelp())

	gap := m.width - lipgloss.Width(left) - lipgloss.Width(helpLine)
	if gap < 1 {
		gap = 1
	}
	return left + strings.Repeat(" ", gap) + helpLine
}

// ---- render: overlays -------------------------------------------------------

func (m Model) renderEditOverlay() string {
	return m.styles.OverlayEdit.Render(m.editView.View())
}

func (m Model) renderFilterOverlay() string {
	return m.styles.OverlayFilter.Render(m.filterView.View())
}

func (m Model) helpContent() string {
	var b strings.Builder

	b.WriteString(m.styles.HelpTitle.Render("Keybindings") + "\n\n")

	b.WriteString(m.styles.HelpSection.Render("Navigation") + "\n")
	for _, bind := range []struct{ k, desc string }{
		{"j/\u2193  k/\u2191", "Move down / up (scroll in detail pane)"},
		{"h/\u2190 l/\u2192", "Focus left / right pane"},
		{"Tab / Shift+Tab", "Next / previous status tab (left pane)"},
		{"Tab / Shift+Tab", "Cycle detail / updates focus (right pane)"},
	} {
		b.WriteString("  " + m.styles.HelpKey.Render(bind.k) + m.styles.HelpDesc.Render(bind.desc) + "\n")
	}

	b.WriteString("\n" + m.styles.HelpSection.Render("Actions") + "\n")
	for _, bind := range []struct{ k, desc string }{
		{"n", "New task"},
		{"e", "Edit task"},
		{"s", "Cycle status"},
		{"x", "Mark done"},
		{"d", "Delete task"},
		{"u", "Add update (enter to submit)"},
		{"/", "Search / filter"},
		{"Enter", "Select / view detail"},
	} {
		b.WriteString("  " + m.styles.HelpKey.Render(bind.k) + m.styles.HelpDesc.Render(bind.desc) + "\n")
	}

	b.WriteString("\n" + m.styles.HelpSection.Render("General") + "\n")
	for _, bind := range []struct{ k, desc string }{
		{"L", "Switch task list (n: new, r: rename, d: delete)"},
		{"R", "Switch remote database (n: add new)"},
		{"T", "Theme picker (live preview)"},
		{"S", "Sort tasks"},
		{"?", "Toggle this help"},
		{"q / Ctrl+C", "Quit"},
		{"Esc", "Close overlay / back"},
	} {
		b.WriteString("  " + m.styles.HelpKey.Render(bind.k) + m.styles.HelpDesc.Render(bind.desc) + "\n")
	}

	return b.String()
}

func (m Model) helpOverlayInnerSize() (int, int) {
	maxW := m.width * 70 / 100
	if maxW < 40 {
		maxW = 40
	}
	innerW := maxW - 6
	if innerW < 30 {
		innerW = 30
	}
	maxH := m.height * 70 / 100
	if maxH < 10 {
		maxH = 10
	}
	innerH := maxH - 6
	return innerW, innerH
}

func (m *Model) initHelpViewport() {
	content := m.helpContent()
	innerW, innerH := m.helpOverlayInnerSize()

	m.helpViewport = viewport.New(innerW, innerH)
	m.helpViewport.SetContent(content)
}

func (m Model) renderHelpOverlay() string {
	innerW, innerH := m.helpOverlayInnerSize()
	content := m.helpViewport.View()

	contentH := lipgloss.Height(m.helpContent())
	if contentH > innerH {
		scrollPct := int(m.helpViewport.ScrollPercent() * 100)
		content += "\n" + m.styles.HintText.Render(fmt.Sprintf("↑/↓ scroll • %d%%", scrollPct))
	}

	return m.styles.OverlayHelp.
		Width(innerW).
		Render(content)
}

// ---- key routing ------------------------------------------------------------

func (m Model) handleListKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	prevSelected := m.listView.Cursor()

	switch {
	case isKey(msg, m.keymap.Quit):
		m.quitting = true
		return m, tea.Quit

	case isKey(msg, m.keymap.Help):
		m.initHelpViewport()
		m.state.ActiveOverlay = OverlayHelp
		return m, nil

	case isKey(msg, m.keymap.Right):
		m.state.ActivePane = PaneDetail
		m.detailView = m.detailView.SetDetailFocus(m.state.DetailFocus == FocusDetails)
		return m, nil

	case isKey(msg, m.keymap.Tab):
		m.state.ActiveTab = m.nextTab()
		m.state.StatusMsg = ""
		return m, m.loadTasks()

	case isKey(msg, m.keymap.ShiftTab):
		m.state.ActiveTab = m.prevTab()
		m.state.StatusMsg = ""
		return m, m.loadTasks()

	case isKey(msg, m.keymap.ListSwitch):
		m.state.ActiveOverlay = OverlayListSwitch
		return m, nil

	case isKey(msg, m.keymap.ThemePicker):
		return m.openThemePicker(), nil

	case isKey(msg, m.keymap.Sort):
		return m.openSortPicker(), nil

	case isKey(msg, m.keymap.RemoteSwitch):
		return m.openRemotePicker(), nil

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
		m.initHelpViewport()
		m.state.ActiveOverlay = OverlayHelp
		return m, nil

	case isKey(msg, m.keymap.ThemePicker):
		return m.openThemePicker(), nil

	case isKey(msg, m.keymap.Sort):
		return m.openSortPicker(), nil

	case isKey(msg, m.keymap.RemoteSwitch):
		return m.openRemotePicker(), nil

	case isKey(msg, m.keymap.Tab):
		// Cycle detail sub-focus: details → updates → details
		if m.state.DetailFocus == FocusDetails {
			m.state.DetailFocus = FocusUpdates
		} else {
			m.state.DetailFocus = FocusDetails
		}
		m.detailView = m.detailView.SetDetailFocus(m.state.DetailFocus == FocusDetails)
		return m, nil

	case isKey(msg, m.keymap.ShiftTab):
		// Reverse cycle
		if m.state.DetailFocus == FocusUpdates {
			m.state.DetailFocus = FocusDetails
		} else {
			m.state.DetailFocus = FocusUpdates
		}
		m.detailView = m.detailView.SetDetailFocus(m.state.DetailFocus == FocusDetails)
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
	if m.state.ActiveTab >= 0 && m.state.ActiveTab < len(m.tabOrder) {
		f.Statuses = TabStatus(m.tabOrder[m.state.ActiveTab])
	} else {
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
	// Use the third status in config as "done", or fall back to domain const.
	var doneStatus domain.Status
	if len(m.cfg.Statuses) >= 3 {
		doneStatus = domain.Status(m.cfg.Statuses[2].Name)
	} else {
		doneStatus = domain.StatusDone
	}
	return func() tea.Msg {
		updated, err := svc.UpdateTask(application.UpdateTaskInput{
			ID:     t.ID,
			Status: &doneStatus,
		})
		return taskUpdatedMsg{task: updated, err: err}
	}
}

func (m Model) cycleStatus(t *domain.Task) tea.Cmd {
	svc := m.taskSvc
	// Build status order from config.
	var order []domain.Status
	for _, s := range m.cfg.Statuses {
		order = append(order, domain.Status(s.Name))
	}
	if len(order) == 0 {
		order = []domain.Status{domain.StatusTodo, domain.StatusInProgress, domain.StatusDone, domain.StatusCancelled}
	}
	next := order[0]
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
			Status:      t.Status,
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
	titleStyle := m.styles.HelpTitle
	cursorStyle := m.styles.ListCursor
	normalStyle := m.styles.ListNormal
	hintStyle := m.styles.HintText
	dangerStyle := m.styles.DangerText

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

	return m.styles.OverlayList.Render(b.String())
}

// ---- theme picker -----------------------------------------------------------

func (m Model) openThemePicker() Model {
	configDir, _ := m.cfg.ResolveDataDir()
	names := config.ListThemes(configDir)
	m.originalThemeName = m.cfg.ThemeName
	m.originalStyles = m.styles
	m.themePickerView = views.NewThemePickerView(names, configDir, m.styles)
	m.state.ActiveOverlay = OverlayThemePicker
	return m
}

func (m Model) handleThemePickerKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case msg.Type == tea.KeyEsc:
		// Revert to original theme.
		m.styles = m.originalStyles
		m.cfg.ThemeName = m.originalThemeName
		m.rebuildAllViewStyles()
		m.state.ActiveOverlay = OverlayNone
		return m, nil

	case msg.String() == "j" || msg.Type == tea.KeyDown:
		m.themePickerView = m.themePickerView.CursorDown()
		return m.applyThemePreview()

	case msg.String() == "k" || msg.Type == tea.KeyUp:
		m.themePickerView = m.themePickerView.CursorUp()
		return m.applyThemePreview()

	case msg.Type == tea.KeyEnter:
		m.themePickerView = m.themePickerView.Select()
		name := m.themePickerView.SelectedName()
		if name != "" {
			m.cfg.ThemeName = name
			_ = config.Save(m.cfg)
			m.state.StatusMsg = "Theme set to " + name
		}
		m.state.ActiveOverlay = OverlayNone
		return m, nil
	}

	return m, nil
}

// applyThemePreview lazily resolves the theme at the cursor and live-applies it.
func (m Model) applyThemePreview() (tea.Model, tea.Cmd) {
	picker, tf, err := m.themePickerView.ResolveAtCursor()
	m.themePickerView = picker
	if err != nil {
		m.state.StatusMsg = "Theme error: " + err.Error()
		return m, nil
	}

	// Apply the preview theme.
	config.ApplyTheme(m.cfg, tf)
	m.styles = styles.NewStyles(m.cfg.Theme)
	m.themePickerView = m.themePickerView.SetStyles(m.styles)
	m.rebuildAllViewStyles()
	return m, nil
}

// rebuildAllViewStyles propagates the current styles to all sub-views.
func (m *Model) rebuildAllViewStyles() {
	m.listView = m.listView.SetStyles(m.styles)
	m.detailView = m.detailView.SetStyles(m.styles)
	m.editView = m.editView.SetStyles(m.styles)
	m.filterView = m.filterView.SetStyles(m.styles)
}

func (m Model) renderThemePickerOverlay() string {
	return m.styles.OverlayList.Render(m.themePickerView.View())
}

// ---- sort picker ------------------------------------------------------------

func (m Model) openSortPicker() Model {
	m.sortView = views.NewSortView(m.styles).
		SetCursorFromSort(m.state.Filter.SortField, m.state.Filter.SortDir)
	m.state.ActiveOverlay = OverlaySort
	return m
}

func (m Model) handleSortKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case msg.Type == tea.KeyEsc:
		m.state.ActiveOverlay = OverlayNone
		return m, nil

	case msg.String() == "j" || msg.Type == tea.KeyDown:
		m.sortView = m.sortView.CursorDown()
		return m, nil

	case msg.String() == "k" || msg.Type == tea.KeyUp:
		m.sortView = m.sortView.CursorUp()
		return m, nil

	case msg.Type == tea.KeyEnter:
		m.sortView = m.sortView.Select()
		opt := m.sortView.SelectedOption()
		m.state.Filter.SortField = opt.Field
		m.state.Filter.SortDir = opt.Dir
		m.state.ActiveOverlay = OverlayNone
		m.state.StatusMsg = "Sorted by " + opt.Label
		return m, m.loadTasks()
	}

	return m, nil
}

func (m Model) renderSortOverlay() string {
	return m.styles.OverlayList.Render(m.sortView.View())
}

// ---- remote picker ----------------------------------------------------------

func (m Model) openRemotePicker() Model {
	choices := []string{"local"}
	labels := []string{"Local (SQLite)"}
	for _, name := range m.cfg.RemoteNames() {
		rc := m.cfg.Remotes[name]
		labels = append(labels, fmt.Sprintf("%s — %s", name, rc.Label()))
		choices = append(choices, name)
	}
	// Set cursor to the currently active remote.
	cursor := 0
	for i, c := range choices {
		if c == m.cfg.LastRemote {
			cursor = i
			break
		}
	}
	m.remotePickerChoices = choices
	m.remotePickerLabels = labels
	m.remotePickerCursor = cursor
	m.state.ActiveOverlay = OverlayRemotePicker
	return m
}

func (m Model) handleRemotePickerKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Add-remote mode: multi-step text input
	if m.remoteAddMode {
		switch msg.Type {
		case tea.KeyEsc:
			m.remoteAddMode = false
			m.remoteAddStep = 0
			m.remoteAddInput.Blur()
			return m, nil
		case tea.KeyEnter:
			val := strings.TrimSpace(m.remoteAddInput.Value())
			switch m.remoteAddStep {
			case 0: // name
				if val == "" || val == "local" {
					return m, nil
				}
				m.remoteAddName = val
				m.remoteAddStep = 1
				m.remoteAddInput.SetValue("")
				m.remoteAddInput.Placeholder = "host (e.g. db.example.com)"
				return m, nil
			case 1: // host
				if val == "" {
					return m, nil
				}
				m.remoteAddHost = val
				m.remoteAddStep = 2
				m.remoteAddInput.SetValue("3306")
				m.remoteAddInput.Placeholder = "port"
				return m, nil
			case 2: // port
				if val == "" {
					val = "3306"
				}
				m.remoteAddPort = val
				m.remoteAddStep = 3
				m.remoteAddInput.SetValue("tork")
				m.remoteAddInput.Placeholder = "database"
				return m, nil
			case 3: // database
				if val == "" {
					val = "tork"
				}
				m.remoteAddDB = val
				m.remoteAddStep = 4
				m.remoteAddInput.SetValue("")
				m.remoteAddInput.Placeholder = "username"
				return m, nil
			case 4: // username
				if val == "" {
					return m, nil
				}
				// Build and save the new remote config.
				port := 3306
				if p, err := strconv.Atoi(m.remoteAddPort); err == nil && p > 0 {
					port = p
				}
				rc := &config.RemoteConfig{
					Driver:   "mysql",
					Host:     m.remoteAddHost,
					Port:     port,
					Database: m.remoteAddDB,
					Username: val,
				}
				if m.cfg.Remotes == nil {
					m.cfg.Remotes = make(map[string]*config.RemoteConfig)
				}
				m.cfg.Remotes[m.remoteAddName] = rc
				_ = config.Save(m.cfg)
				m.state.StatusMsg = "Remote \"" + m.remoteAddName + "\" added"
				// Reset add mode and refresh picker.
				m.remoteAddMode = false
				m.remoteAddStep = 0
				m.remoteAddInput.Blur()
				m = m.openRemotePicker()
				return m, nil
			}
		}
		var cmd tea.Cmd
		m.remoteAddInput, cmd = m.remoteAddInput.Update(msg)
		return m, cmd
	}

	// Selection mode
	switch {
	case msg.Type == tea.KeyEsc:
		m.state.ActiveOverlay = OverlayNone
		return m, nil

	case msg.String() == "j" || msg.Type == tea.KeyDown:
		if m.remotePickerCursor < len(m.remotePickerChoices)-1 {
			m.remotePickerCursor++
		}
		return m, nil

	case msg.String() == "k" || msg.Type == tea.KeyUp:
		if m.remotePickerCursor > 0 {
			m.remotePickerCursor--
		}
		return m, nil

	case msg.String() == "n" || msg.String() == "N":
		m.remoteAddMode = true
		m.remoteAddStep = 0
		m.remoteAddInput.SetValue("")
		m.remoteAddInput.Placeholder = "remote name"
		m.remoteAddInput.Focus()
		return m, textinput.Blink

	case msg.String() == "d" || msg.String() == "D":
		// Delete the selected remote (not "local").
		if m.remotePickerCursor > 0 && m.remotePickerCursor < len(m.remotePickerChoices) {
			name := m.remotePickerChoices[m.remotePickerCursor]
			delete(m.cfg.Remotes, name)
			if m.cfg.LastRemote == name {
				m.cfg.LastRemote = "local"
			}
			_ = config.Save(m.cfg)
			m.state.StatusMsg = "Remote \"" + name + "\" removed"
			m = m.openRemotePicker()
		}
		return m, nil

	case msg.Type == tea.KeyEnter:
		picked := m.remotePickerChoices[m.remotePickerCursor]
		m.state.ActiveOverlay = OverlayNone
		if picked == m.cfg.LastRemote {
			m.state.StatusMsg = "Already connected to " + picked
			return m, nil
		}
		if picked == "local" {
			// Switch to local: re-connect with local SQLite.
			m.state.StatusMsg = "Switching to local..."
			return m, m.connectRemote("local", "")
		}
		// Remote selected — show password overlay.
		rc := m.cfg.ActiveRemote(picked)
		if rc == nil {
			m.state.StatusMsg = "Remote not found: " + picked
			return m, nil
		}
		m.pendingRemoteName = picked
		m.passwordLabel = rc.Label()
		m.passwordInput.Reset()
		m.passwordInput.Focus()
		m.state.ActiveOverlay = OverlayPasswordPrompt
		return m, textinput.Blink
	}
	return m, nil
}

func (m Model) renderRemotePickerOverlay() string {
	var b strings.Builder

	if m.remoteAddMode {
		b.WriteString("  Add Remote\n\n")
		fields := []struct{ label, value string }{
			{"Name", m.remoteAddName},
			{"Host", m.remoteAddHost},
			{"Port", m.remoteAddPort},
			{"Database", m.remoteAddDB},
			{"Username", ""},
		}
		for i, f := range fields {
			if i < m.remoteAddStep {
				b.WriteString(fmt.Sprintf("  %s: %s\n", f.label, m.styles.HintText.Render(f.value)))
			} else if i == m.remoteAddStep {
				b.WriteString(fmt.Sprintf("  %s: %s\n", f.label, m.remoteAddInput.View()))
			}
		}
		b.WriteString("\n  " + m.styles.HintText.Render("Enter to continue · Esc to cancel"))
		return m.styles.OverlayList.Render(b.String())
	}

	b.WriteString("  Switch Database\n\n")
	for i, label := range m.remotePickerLabels {
		cursor := "  "
		if i == m.remotePickerCursor {
			cursor = "> "
		}
		suffix := ""
		if m.remotePickerChoices[i] == m.cfg.LastRemote {
			suffix = " (current)"
		}
		b.WriteString(fmt.Sprintf("  %s%s%s\n", cursor, label, suffix))
	}
	b.WriteString("\n  ↑/↓ navigate · Enter select · n add · d delete · Esc cancel")
	return m.styles.OverlayList.Render(b.String())
}

// ---- password overlay -------------------------------------------------------

func (m Model) handlePasswordKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEsc:
		m.state.ActiveOverlay = OverlayNone
		m.pendingRemoteName = ""
		m.passwordInput.Reset()
		return m, nil

	case tea.KeyEnter:
		password := strings.TrimSpace(m.passwordInput.Value())
		m.state.ActiveOverlay = OverlayNone
		m.state.StatusMsg = "Connecting to " + m.pendingRemoteName + "..."
		return m, m.connectRemote(m.pendingRemoteName, password)
	}

	var cmd tea.Cmd
	m.passwordInput, cmd = m.passwordInput.Update(msg)
	return m, cmd
}

func (m Model) renderPasswordOverlay() string {
	var b strings.Builder
	b.WriteString("  Connect to " + m.pendingRemoteName + "\n")
	b.WriteString("  " + m.styles.HintText.Render(m.passwordLabel) + "\n\n")
	b.WriteString("  Password: " + m.passwordInput.View() + "\n\n")
	b.WriteString("  " + m.styles.HintText.Render("Enter to connect · Esc to cancel"))
	return m.styles.OverlayList.Render(b.String())
}

// connectRemote fires an async command that calls ConnectFunc and returns
// the result as a remoteConnectedMsg.
func (m Model) connectRemote(remoteName, password string) tea.Cmd {
	fn := m.connectFunc
	return func() tea.Msg {
		svc, err := fn(remoteName, password)
		return remoteConnectedMsg{svc: svc, remoteName: remoteName, err: err}
	}
}
