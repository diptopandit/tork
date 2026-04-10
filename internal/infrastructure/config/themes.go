package config

import "fmt"

// BuiltinThemes maps theme names to their complete definitions.
var BuiltinThemes = map[string]ThemeFile{
	"default":        themeDefault,
	"light":          themeLight,
	"dracula":        themeDracula,
	"solarized-dark": themeSolarizedDark,
	"nord":           themeNord,
}

var themeDefault = ThemeFile{
	Name:        "default",
	Description: "Default dark theme with purple accents",
	Author:      "tork",
	Colors: ThemeConfig{
		Primary:    "#7C3AED",
		Secondary:  "#6B7280",
		Active:     "#7C3AED",
		Inactive:   "#374151",
		Success:    "#10B981",
		Warning:    "#F59E0B",
		Danger:     "#EF4444",
		Text:       "#E5E7EB",
		TextMuted:  "#9CA3AF",
		TextBright: "#FFFFFF",
		Accent:     "#60A5FA",
	},
	Border:         "rounded",
	StatusColors:   []string{"#F59E0B", "#60A5FA", "#10B981", "#EF4444"},
	PriorityColors: []string{"#6B7280", "#F59E0B", "#FB923C", "#EF4444"},
}

var themeLight = ThemeFile{
	Name:        "light",
	Description: "Light theme for bright terminal backgrounds",
	Author:      "tork",
	Colors: ThemeConfig{
		Primary:    "#6D28D9",
		Secondary:  "#9CA3AF",
		Active:     "#6D28D9",
		Inactive:   "#D1D5DB",
		Success:    "#059669",
		Warning:    "#D97706",
		Danger:     "#DC2626",
		Text:       "#1F2937",
		TextMuted:  "#6B7280",
		TextBright: "#111827",
		Accent:     "#2563EB",
	},
	Border:         "rounded",
	StatusColors:   []string{"#D97706", "#2563EB", "#059669", "#DC2626"},
	PriorityColors: []string{"#9CA3AF", "#D97706", "#EA580C", "#DC2626"},
}

var themeDracula = ThemeFile{
	Name:        "dracula",
	Description: "Dracula color scheme",
	Author:      "tork",
	Colors: ThemeConfig{
		Primary:    "#BD93F9",
		Secondary:  "#6272A4",
		Active:     "#BD93F9",
		Inactive:   "#44475A",
		Success:    "#50FA7B",
		Warning:    "#F1FA8C",
		Danger:     "#FF5555",
		Text:       "#F8F8F2",
		TextMuted:  "#6272A4",
		TextBright: "#FFFFFF",
		Accent:     "#8BE9FD",
	},
	Border:         "rounded",
	StatusColors:   []string{"#F1FA8C", "#8BE9FD", "#50FA7B", "#FF5555"},
	PriorityColors: []string{"#6272A4", "#F1FA8C", "#FFB86C", "#FF5555"},
}

var themeSolarizedDark = ThemeFile{
	Name:        "solarized-dark",
	Description: "Solarized Dark color scheme",
	Author:      "tork",
	Colors: ThemeConfig{
		Primary:    "#268BD2",
		Secondary:  "#586E75",
		Active:     "#268BD2",
		Inactive:   "#073642",
		Success:    "#859900",
		Warning:    "#B58900",
		Danger:     "#DC322F",
		Text:       "#839496",
		TextMuted:  "#586E75",
		TextBright: "#FDF6E3",
		Accent:     "#2AA198",
	},
	Border:         "rounded",
	StatusColors:   []string{"#B58900", "#268BD2", "#859900", "#DC322F"},
	PriorityColors: []string{"#586E75", "#B58900", "#CB4B16", "#DC322F"},
}

var themeNord = ThemeFile{
	Name:        "nord",
	Description: "Nord color scheme — arctic blue tones",
	Author:      "tork",
	Colors: ThemeConfig{
		Primary:    "#88C0D0",
		Secondary:  "#4C566A",
		Active:     "#88C0D0",
		Inactive:   "#3B4252",
		Success:    "#A3BE8C",
		Warning:    "#EBCB8B",
		Danger:     "#BF616A",
		Text:       "#D8DEE9",
		TextMuted:  "#4C566A",
		TextBright: "#ECEFF4",
		Accent:     "#81A1C1",
	},
	Border:         "rounded",
	StatusColors:   []string{"#EBCB8B", "#81A1C1", "#A3BE8C", "#BF616A"},
	PriorityColors: []string{"#4C566A", "#EBCB8B", "#D08770", "#BF616A"},
}

// ValidateTheme checks that a ThemeFile has all required fields populated.
func ValidateTheme(t ThemeFile) error {
	if t.Name == "" {
		return fmt.Errorf("theme: missing required field \"name\"")
	}
	c := t.Colors
	for _, pair := range []struct {
		field, value string
	}{
		{"primary", c.Primary},
		{"secondary", c.Secondary},
		{"active", c.Active},
		{"inactive", c.Inactive},
		{"success", c.Success},
		{"warning", c.Warning},
		{"danger", c.Danger},
		{"text", c.Text},
		{"text_muted", c.TextMuted},
		{"text_bright", c.TextBright},
		{"accent", c.Accent},
	} {
		if pair.value == "" {
			return fmt.Errorf("theme %q: missing required color %q", t.Name, pair.field)
		}
	}
	switch t.Border {
	case "rounded", "normal", "double", "hidden", "":
		// valid
	default:
		return fmt.Errorf("theme %q: invalid border style %q (must be rounded, normal, double, or hidden)", t.Name, t.Border)
	}
	if len(t.StatusColors) == 0 {
		return fmt.Errorf("theme %q: status_colors must not be empty", t.Name)
	}
	if len(t.PriorityColors) == 0 {
		return fmt.Errorf("theme %q: priority_colors must not be empty", t.Name)
	}
	return nil
}

// BuiltinThemeNames returns the sorted list of built-in theme names.
func BuiltinThemeNames() []string {
	return []string{"default", "dracula", "light", "nord", "solarized-dark"}
}
