package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestValidateTheme_Complete(t *testing.T) {
	for name, tf := range BuiltinThemes {
		if err := ValidateTheme(tf); err != nil {
			t.Errorf("built-in theme %q should be valid: %v", name, err)
		}
	}
}

func TestValidateTheme_MissingName(t *testing.T) {
	tf := BuiltinThemes["default"]
	tf.Name = ""
	if err := ValidateTheme(tf); err == nil {
		t.Error("expected error for missing name")
	}
}

func TestValidateTheme_MissingColor(t *testing.T) {
	tf := BuiltinThemes["default"]
	tf.Colors.Primary = ""
	if err := ValidateTheme(tf); err == nil {
		t.Error("expected error for missing primary color")
	}
}

func TestValidateTheme_InvalidBorder(t *testing.T) {
	tf := BuiltinThemes["default"]
	tf.Border = "zigzag"
	if err := ValidateTheme(tf); err == nil {
		t.Error("expected error for invalid border style")
	}
}

func TestValidateTheme_EmptyStatusColors(t *testing.T) {
	tf := BuiltinThemes["default"]
	tf.StatusColors = nil
	if err := ValidateTheme(tf); err == nil {
		t.Error("expected error for empty status_colors")
	}
}

func TestValidateTheme_EmptyPriorityColors(t *testing.T) {
	tf := BuiltinThemes["default"]
	tf.PriorityColors = nil
	if err := ValidateTheme(tf); err == nil {
		t.Error("expected error for empty priority_colors")
	}
}

func TestResolveTheme_Builtin(t *testing.T) {
	for _, name := range BuiltinThemeNames() {
		tf, err := ResolveTheme(name, t.TempDir())
		if err != nil {
			t.Errorf("ResolveTheme(%q): %v", name, err)
		}
		if tf.Name != name {
			t.Errorf("ResolveTheme(%q).Name = %q", name, tf.Name)
		}
	}
}

func TestResolveTheme_DefaultOnEmpty(t *testing.T) {
	tf, err := ResolveTheme("", t.TempDir())
	if err != nil {
		t.Fatalf("ResolveTheme empty: %v", err)
	}
	if tf.Name != "default" {
		t.Errorf("expected default theme, got %q", tf.Name)
	}
}

func TestResolveTheme_NotFound(t *testing.T) {
	_, err := ResolveTheme("nonexistent", t.TempDir())
	if err == nil {
		t.Error("expected error for nonexistent theme")
	}
}

func TestResolveTheme_UserFile(t *testing.T) {
	dir := t.TempDir()
	themesDir := filepath.Join(dir, "themes")
	_ = os.MkdirAll(themesDir, 0o755)

	tf := BuiltinThemes["dracula"]
	tf.Name = "my-custom"
	data, _ := json.Marshal(tf)
	_ = os.WriteFile(filepath.Join(themesDir, "my-custom.json"), data, 0o644)

	resolved, err := ResolveTheme("my-custom", dir)
	if err != nil {
		t.Fatalf("ResolveTheme user file: %v", err)
	}
	if resolved.Name != "my-custom" {
		t.Errorf("expected my-custom, got %q", resolved.Name)
	}
}

func TestResolveTheme_UserFileInvalid(t *testing.T) {
	dir := t.TempDir()
	themesDir := filepath.Join(dir, "themes")
	_ = os.MkdirAll(themesDir, 0o755)

	// Write an incomplete theme
	data := []byte(`{"name":"bad","colors":{"primary":"#FFF"},"status_colors":["#FFF"],"priority_colors":["#FFF"]}`)
	_ = os.WriteFile(filepath.Join(themesDir, "bad.json"), data, 0o644)

	_, err := ResolveTheme("bad", dir)
	if err == nil {
		t.Error("expected validation error for incomplete theme file")
	}
}

func TestStatusColor_InBounds(t *testing.T) {
	tc := ThemeConfig{
		Primary:      "#AAA",
		StatusColors: []string{"#111", "#222", "#333"},
	}
	if got := tc.StatusColor(0); got != "#111" {
		t.Errorf("StatusColor(0) = %q, want #111", got)
	}
	if got := tc.StatusColor(2); got != "#333" {
		t.Errorf("StatusColor(2) = %q, want #333", got)
	}
}

func TestStatusColor_OutOfBounds(t *testing.T) {
	tc := ThemeConfig{
		Primary:      "#AAA",
		StatusColors: []string{"#111"},
	}
	if got := tc.StatusColor(5); got != "#AAA" {
		t.Errorf("StatusColor(5) = %q, want fallback #AAA", got)
	}
}

func TestPriorityColor_InBounds(t *testing.T) {
	tc := ThemeConfig{
		Primary:        "#AAA",
		PriorityColors: []string{"#111", "#222"},
	}
	if got := tc.PriorityColor(1); got != "#222" {
		t.Errorf("PriorityColor(1) = %q, want #222", got)
	}
}

func TestPriorityColor_OutOfBounds(t *testing.T) {
	tc := ThemeConfig{
		Primary:        "#AAA",
		PriorityColors: []string{"#111"},
	}
	if got := tc.PriorityColor(-1); got != "#AAA" {
		t.Errorf("PriorityColor(-1) = %q, want fallback #AAA", got)
	}
}

func TestBorderStyle_AllTypes(t *testing.T) {
	for _, border := range []string{"rounded", "normal", "double", "hidden", ""} {
		tc := ThemeConfig{Border: border}
		// Just verifying no panic; lipgloss.Border is not directly comparable.
		_ = tc.BorderStyle()
	}
}

func TestListThemes(t *testing.T) {
	dir := t.TempDir()
	themesDir := filepath.Join(dir, "themes")
	_ = os.MkdirAll(themesDir, 0o755)
	_ = os.WriteFile(filepath.Join(themesDir, "ocean.json"), []byte("{}"), 0o644)
	_ = os.WriteFile(filepath.Join(themesDir, "forest.json"), []byte("{}"), 0o644)

	names := ListThemes(dir)
	// Should include all 5 builtins + ocean + forest
	found := make(map[string]bool)
	for _, n := range names {
		found[n] = true
	}
	for _, bn := range BuiltinThemeNames() {
		if !found[bn] {
			t.Errorf("missing built-in theme %q in ListThemes", bn)
		}
	}
	if !found["ocean"] {
		t.Error("missing user theme 'ocean' in ListThemes")
	}
	if !found["forest"] {
		t.Error("missing user theme 'forest' in ListThemes")
	}
}
