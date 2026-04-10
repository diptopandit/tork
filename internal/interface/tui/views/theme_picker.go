package views

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/diptopandit/tork/internal/infrastructure/config"
	"github.com/diptopandit/tork/internal/interface/tui/styles"
)

// ThemePickerView presents a list of themes with lazy loading and live preview.
type ThemePickerView struct {
	names     []string                    // all known theme names (built-in + custom)
	descs     map[string]string           // name → description (populated lazily)
	cache     map[string]config.ThemeFile // name → fully resolved theme (populated lazily)
	cursor    int
	configDir string
	styles    styles.Styles
	selected  bool   // true after Enter
	selName   string // name chosen on Enter
}

// NewThemePickerView builds the picker. themeNames should come from config.ListThemes.
func NewThemePickerView(themeNames []string, configDir string, s styles.Styles) ThemePickerView {
	descs := make(map[string]string, len(themeNames))
	cache := make(map[string]config.ThemeFile, len(themeNames))

	// Pre-populate descriptions for built-in themes (cheap, in-memory).
	for _, n := range themeNames {
		if t, ok := config.BuiltinThemes[n]; ok {
			descs[n] = t.Description
			cache[n] = t
		}
	}

	return ThemePickerView{
		names:     themeNames,
		descs:     descs,
		cache:     cache,
		configDir: configDir,
		styles:    s,
	}
}

// SetStyles replaces the styles used for rendering.
func (v ThemePickerView) SetStyles(s styles.Styles) ThemePickerView {
	v.styles = s
	return v
}

// CursorUp moves the cursor up.
func (v ThemePickerView) CursorUp() ThemePickerView {
	if v.cursor > 0 {
		v.cursor--
	}
	return v
}

// CursorDown moves the cursor down.
func (v ThemePickerView) CursorDown() ThemePickerView {
	if v.cursor < len(v.names)-1 {
		v.cursor++
	}
	return v
}

// Select marks the current cursor item as selected.
func (v ThemePickerView) Select() ThemePickerView {
	if v.cursor >= 0 && v.cursor < len(v.names) {
		v.selected = true
		v.selName = v.names[v.cursor]
	}
	return v
}

// Selected reports whether the user pressed Enter.
func (v ThemePickerView) Selected() bool { return v.selected }

// SelectedName returns the name chosen by Enter.
func (v ThemePickerView) SelectedName() string { return v.selName }

// CursorName returns the theme name at the current cursor position.
func (v ThemePickerView) CursorName() string {
	if v.cursor >= 0 && v.cursor < len(v.names) {
		return v.names[v.cursor]
	}
	return ""
}

// ResolveAtCursor lazily resolves the theme under the cursor. Built-in themes
// are returned from the cache; custom themes are read from disk once and cached.
func (v ThemePickerView) ResolveAtCursor() (ThemePickerView, config.ThemeFile, error) {
	name := v.CursorName()
	if name == "" {
		return v, config.ThemeFile{}, fmt.Errorf("no theme selected")
	}

	if tf, ok := v.cache[name]; ok {
		return v, tf, nil
	}

	// Lazy load custom theme from disk.
	tf, err := config.ResolveTheme(name, v.configDir)
	if err != nil {
		return v, config.ThemeFile{}, err
	}
	v.cache[name] = tf
	v.descs[name] = tf.Description
	return v, tf, nil
}

// View renders the picker list.
func (v ThemePickerView) View() string {
	var b strings.Builder

	b.WriteString(v.styles.HelpTitle.Render("Theme Picker") + "\n")
	b.WriteString(v.styles.HintText.Render("j/k navigate • Enter select • Esc cancel") + "\n\n")

	for i, name := range v.names {
		desc := v.descs[name]
		if desc == "" {
			desc = "(custom theme)"
		}

		label := fmt.Sprintf("%-20s %s", name, v.styles.HintText.Render(desc))

		if i == v.cursor {
			b.WriteString(v.styles.ListCursor.Render("▸ "+label) + "\n")
		} else {
			b.WriteString(v.styles.ListNormal.Render("  "+label) + "\n")
		}
	}

	// Show current selection info with a color swatch preview.
	if tf, ok := v.cache[v.CursorName()]; ok {
		b.WriteString("\n")
		swatch := lipgloss.NewStyle().Background(lipgloss.Color(tf.Colors.Primary)).Render("  ") +
			lipgloss.NewStyle().Background(lipgloss.Color(tf.Colors.Success)).Render("  ") +
			lipgloss.NewStyle().Background(lipgloss.Color(tf.Colors.Warning)).Render("  ") +
			lipgloss.NewStyle().Background(lipgloss.Color(tf.Colors.Danger)).Render("  ") +
			lipgloss.NewStyle().Background(lipgloss.Color(tf.Colors.Accent)).Render("  ")
		b.WriteString("  " + swatch)
		if tf.Author != "" {
			b.WriteString("  " + v.styles.HintText.Render("by "+tf.Author))
		}
	}

	return b.String()
}
