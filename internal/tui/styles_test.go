package tui

import (
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
)

func init() {
	lipgloss.SetColorProfile(termenv.TrueColor)
}

func TestApplyFontNoOp(t *testing.T) {
	// Reset to a clean state.
	initStyles(ThemeTokyoNight)

	// Capture pre-apply state for a non-bold, non-italic style.
	preBold := MenuItemStyle.GetBold()
	preItalic := MenuItemStyle.GetItalic()

	// Apply empty font config — should be a no-op.
	applyFont(FontConfig{})

	if MenuItemStyle.GetBold() != preBold {
		t.Error("applyFont with empty config should not change Bold")
	}
	if MenuItemStyle.GetItalic() != preItalic {
		t.Error("applyFont with empty config should not change Italic")
	}
}

func TestApplyFontBold(t *testing.T) {
	initStyles(ThemeTokyoNight)

	// MenuItemStyle is not bold by default.
	before := MenuItemStyle.GetBold()
	if before {
		t.Skip("MenuItemStyle is unexpectedly bold in this version of lipgloss")
	}

	applyFont(FontConfig{Bold: true})

	if !MenuItemStyle.GetBold() {
		t.Error("MenuItemStyle should be bold after applyFont(Bold: true)")
	}

	// HeaderStyle was already bold — should still be bold.
	if !HeaderStyle.GetBold() {
		t.Error("HeaderStyle should remain bold after applyFont(Bold: true)")
	}
}

func TestApplyFontItalic(t *testing.T) {
	initStyles(ThemeTokyoNight)

	// MenuItemStyle is not italic by default.
	before := MenuItemStyle.GetItalic()
	if before {
		t.Skip("MenuItemStyle is unexpectedly italic in this version of lipgloss")
	}

	applyFont(FontConfig{Italic: true})

	if !MenuItemStyle.GetItalic() {
		t.Error("MenuItemStyle should be italic after applyFont(Italic: true)")
	}

	// TimestampStyle was already italic — should still be italic.
	if !TimestampStyle.GetItalic() {
		t.Error("TimestampStyle should remain italic after applyFont(Italic: true)")
	}
}

func TestApplyFontBoth(t *testing.T) {
	initStyles(ThemeTokyoNight)

	applyFont(FontConfig{Bold: true, Italic: true})

	if !MenuItemStyle.GetBold() {
		t.Error("MenuItemStyle should be bold after applyFont(Bold: true)")
	}
	if !MenuItemStyle.GetItalic() {
		t.Error("MenuItemStyle should be italic after applyFont(Italic: true)")
	}
}

func TestApplyFontAllStyles(t *testing.T) {
	// Verify that ALL styles in allStyles are affected by applyFont.
	initStyles(ThemeTokyoNight)

	// Check a representative sample of styles before applying.
	initialBold := make(map[string]bool)
	initialItalic := make(map[string]bool)
	for _, s := range allStyles {
		name := "style" // we don't have names, just store by index
		initialBold[name+string(rune(len(initialBold)))] = s.GetBold()
		initialItalic[name+string(rune(len(initialItalic)))] = s.GetItalic()
	}

	applyFont(FontConfig{Bold: true, Italic: true})

	for _, s := range allStyles {
		if !s.GetBold() {
			t.Error("all styles should be bold after applyFont(Bold: true)")
		}
		if !s.GetItalic() {
			t.Error("all styles should be italic after applyFont(Italic: true)")
		}
	}
}

func TestApplyFontPreservesTheme(t *testing.T) {
	// Verify that applyFont does not alter color variables.
	initStyles(ThemeTokyoNight)

	expectedBase := ThemeTokyoNight.Base
	expectedPrimary := ThemeTokyoNight.Primary

	applyFont(FontConfig{Bold: true, Italic: true})

	if colorBase != expectedBase {
		t.Errorf("colorBase changed from %q to %q after applyFont", expectedBase, colorBase)
	}
	if colorPrimary != expectedPrimary {
		t.Errorf("colorPrimary changed from %q to %q after applyFont", expectedPrimary, colorPrimary)
	}
}

func TestApplyFontWithDracula(t *testing.T) {
	// applyFont should work correctly with non-default themes.
	initStyles(ThemeDracula)
	applyFont(FontConfig{Bold: true})

	if !AppStyle.GetBold() {
		t.Error("AppStyle should be bold after applyFont with Dracula theme")
	}
	if colorBase != ThemeDracula.Base {
		t.Errorf("colorBase changed to %q after applyFont", colorBase)
	}

	// Restore default.
	initStyles(ThemeTokyoNight)
}
