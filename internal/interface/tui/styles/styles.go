package styles

import (
	"github.com/charmbracelet/lipgloss"

	"github.com/diptopandit/tork/internal/infrastructure/config"
)

// Styles holds pre-computed lipgloss styles for every TUI component.
type Styles struct {
	// Tab bar
	TabActive   lipgloss.Style
	TabInactive lipgloss.Style

	// Pane borders
	PaneBorderFocused   lipgloss.Style
	PaneBorderUnfocused lipgloss.Style

	// Separator
	Separator lipgloss.Style

	// Table (list view)
	TableHeaderActive     lipgloss.Style
	TableHeaderInactive   lipgloss.Style
	TableCellActive       lipgloss.Style
	TableCellInactive     lipgloss.Style
	TableSelectedActive   lipgloss.Style
	TableSelectedInactive lipgloss.Style

	// Detail view
	Title      lipgloss.Style
	FieldLabel lipgloss.Style
	FieldValue lipgloss.Style
	Timestamp  lipgloss.Style
	EmptyState lipgloss.Style

	// Form labels
	LabelFocused   lipgloss.Style
	LabelUnfocused lipgloss.Style

	// Overlay containers
	OverlayEdit   lipgloss.Style
	OverlayFilter lipgloss.Style
	OverlayHelp   lipgloss.Style
	OverlayList   lipgloss.Style

	// Help overlay parts
	HelpTitle   lipgloss.Style
	HelpSection lipgloss.Style
	HelpKey     lipgloss.Style
	HelpDesc    lipgloss.Style

	// List switcher
	ListCursor lipgloss.Style
	ListNormal lipgloss.Style

	// Footer
	FooterText lipgloss.Style

	// Misc
	HintText   lipgloss.Style
	AppTitle   lipgloss.Style
	ListLabel  lipgloss.Style
	DangerText lipgloss.Style

	// The theme reference for dynamic color lookups (status/priority colors).
	Theme config.ThemeConfig
}

// NewStyles constructs a Styles from the resolved theme configuration.
func NewStyles(theme config.ThemeConfig) Styles {
	border := theme.BorderStyle()

	return Styles{
		// Tab bar
		TabActive: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color(theme.TextBright)).
			Background(lipgloss.Color(theme.Primary)).
			Padding(0, 1),
		TabInactive: lipgloss.NewStyle().
			Foreground(lipgloss.Color(theme.Secondary)).
			Padding(0, 1),

		// Pane borders (width/height set dynamically at render time)
		PaneBorderFocused: lipgloss.NewStyle().
			Border(border).
			BorderForeground(lipgloss.Color(theme.Primary)),
		PaneBorderUnfocused: lipgloss.NewStyle().
			Border(border).
			BorderForeground(lipgloss.Color(theme.Inactive)),

		// Separator
		Separator: lipgloss.NewStyle().
			Foreground(lipgloss.Color(theme.Inactive)),

		// Table: active
		TableHeaderActive: lipgloss.NewStyle().
			Padding(0, 0).
			BorderStyle(lipgloss.NormalBorder()).
			BorderForeground(lipgloss.Color(theme.Primary)).
			BorderBottom(true).
			Bold(true).
			Foreground(lipgloss.Color(theme.TextBright)).
			MaxWidth(80),
		TableCellActive: lipgloss.NewStyle().Padding(0, 0),
		TableSelectedActive: lipgloss.NewStyle().
			Padding(0, 0).
			Foreground(lipgloss.Color(theme.TextBright)).
			Background(lipgloss.Color(theme.Primary)).
			Bold(true),

		// Table: inactive
		TableHeaderInactive: lipgloss.NewStyle().
			Padding(0, 0).
			BorderStyle(lipgloss.NormalBorder()).
			BorderForeground(lipgloss.Color(theme.Inactive)).
			BorderBottom(true).
			Bold(false).
			Foreground(lipgloss.Color(theme.TextMuted)).
			MaxWidth(80),
		TableCellInactive: lipgloss.NewStyle().Padding(0, 0),
		TableSelectedInactive: lipgloss.NewStyle().
			Padding(0, 0).
			Foreground(lipgloss.Color(theme.Text)).
			Background(lipgloss.Color(theme.Inactive)),

		// Detail view
		Title: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color(theme.Warning)),
		FieldLabel: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color(theme.Primary)),
		FieldValue: lipgloss.NewStyle().
			Foreground(lipgloss.Color(theme.Text)),
		Timestamp: lipgloss.NewStyle().
			Foreground(lipgloss.Color(theme.Accent)),
		EmptyState: lipgloss.NewStyle().
			Foreground(lipgloss.Color(theme.Secondary)),

		// Form labels
		LabelFocused: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color(theme.Primary)),
		LabelUnfocused: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color(theme.TextMuted)),

		// Overlay containers
		OverlayEdit: lipgloss.NewStyle().
			Border(border).
			BorderForeground(lipgloss.Color(theme.Warning)).
			Padding(1, 2),
		OverlayFilter: lipgloss.NewStyle().
			Border(border).
			BorderForeground(lipgloss.Color(theme.Secondary)).
			Padding(1, 2),
		OverlayHelp: lipgloss.NewStyle().
			Border(border).
			BorderForeground(lipgloss.Color(theme.Primary)).
			Padding(1, 2),
		OverlayList: lipgloss.NewStyle().
			Border(border).
			BorderForeground(lipgloss.Color(theme.Primary)).
			Padding(1, 2).
			Width(50),

		// Help overlay parts
		HelpTitle: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color(theme.Primary)),
		HelpSection: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color(theme.Warning)),
		HelpKey: lipgloss.NewStyle().
			Bold(true).
			Width(18).
			Foreground(lipgloss.Color(theme.Accent)),
		HelpDesc: lipgloss.NewStyle().
			Foreground(lipgloss.Color(theme.Text)),

		// List switcher
		ListCursor: lipgloss.NewStyle().
			Foreground(lipgloss.Color(theme.TextBright)).
			Background(lipgloss.Color(theme.Primary)).
			Bold(true).
			Padding(0, 1),
		ListNormal: lipgloss.NewStyle().
			Foreground(lipgloss.Color(theme.Text)).
			Padding(0, 1),

		// Footer
		FooterText: lipgloss.NewStyle().
			Foreground(lipgloss.Color(theme.Secondary)),

		// Misc
		HintText: lipgloss.NewStyle().
			Foreground(lipgloss.Color(theme.Secondary)),
		AppTitle: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color(theme.Primary)),
		ListLabel: lipgloss.NewStyle().
			Foreground(lipgloss.Color(theme.TextMuted)),
		DangerText: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color(theme.Danger)),

		Theme: theme,
	}
}
