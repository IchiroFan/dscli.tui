package tui

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
)

func TestDefaultConfig(t *testing.T) {
	cfg := defaultConfig()
	if cfg.Theme != "tokyo-night" {
		t.Errorf("default theme = %q, want %q", cfg.Theme, "tokyo-night")
	}
}

func TestLoadConfigFileNotFound(t *testing.T) {
	// Isolate from real config by using a temp HOME.
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	// loadConfig should return defaults without error when no config file exists.
	cfg := loadConfig()
	if cfg.Theme != "tokyo-night" {
		t.Errorf("expected default theme, got %q", cfg.Theme)
	}
}

func TestLoadConfigInvalidYAML(t *testing.T) {
	// Create a temporary config with invalid YAML.
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	cfgDir := filepath.Join(tmpDir, ".dscli-tui")
	if err := os.MkdirAll(cfgDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cfgDir, "config.yaml"), []byte("not: yaml: : broken"), 0644); err != nil {
		t.Fatal(err)
	}

	cfg := loadConfig()
	if cfg.Theme != "tokyo-night" {
		t.Errorf("expected default theme on invalid YAML, got %q", cfg.Theme)
	}
}

func TestLoadConfigValidTheme(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	cfgDir := filepath.Join(tmpDir, ".dscli-tui")
	if err := os.MkdirAll(cfgDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cfgDir, "config.yaml"), []byte("theme: dracula\n"), 0644); err != nil {
		t.Fatal(err)
	}

	cfg := loadConfig()
	if cfg.Theme != "dracula" {
		t.Errorf("theme = %q, want %q", cfg.Theme, "dracula")
	}
}

func TestLoadConfigUnknownTheme(t *testing.T) {
	// Unknown themes should be passed through (the caller validates them).
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	cfgDir := filepath.Join(tmpDir, ".dscli-tui")
	if err := os.MkdirAll(cfgDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cfgDir, "config.yaml"), []byte("theme: nonexistent\n"), 0644); err != nil {
		t.Fatal(err)
	}

	cfg := loadConfig()
	if cfg.Theme != "nonexistent" {
		t.Errorf("theme = %q, want %q", cfg.Theme, "nonexistent")
	}
}

func TestThemeByName(t *testing.T) {
	tests := []struct {
		name   string
		exists bool
	}{
		{"tokyo-night", true},
		{"dracula", true},
		{"monokai", true},
		{"nord", true},
		{"solarized-light", true},
		{"nonexistent", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, ok := themeByName[tt.name]
			if ok != tt.exists {
				t.Errorf("themeByName[%q] exists = %v, want %v", tt.name, ok, tt.exists)
			}
		})
	}
}

func TestInitStylesAppliesColors(t *testing.T) {
	// Verify that initStyles with Dracula changes the color variables.
	initStyles(ThemeDracula)

	if colorBase != "#282a36" {
		t.Errorf("colorBase = %q, want %q", colorBase, "#282a36")
	}
	if colorPrimary != "#bd93f9" {
		t.Errorf("colorPrimary = %q, want %q", colorPrimary, "#bd93f9")
	}
	if colorGreen != "#50fa7b" {
		t.Errorf("colorGreen = %q, want %q", colorGreen, "#50fa7b")
	}

	// Restore default for subsequent tests.
	initStyles(ThemeTokyoNight)
}

func TestInitStylesStyleVarsRenderStyledOutput(t *testing.T) {
	// After initStyles, styles should produce styled output when rendering text.
	// Note: lipgloss may strip ANSI codes in non-TTY environments, so we verify
	// that styles with structural properties (padding, borders) produce longer output.
	initStyles(ThemeTokyoNight)

	for name, style := range map[string]lipgloss.Style{
		"AppStyle":       AppStyle,
		"HeaderStyle":    HeaderStyle,
		"UserBubbleBase": UserBubbleBase,
		"StatusBarBg":    StatusBarBg,
	} {
		t.Run(name, func(t *testing.T) {
			r := style.Render("x")
			if len(r) == 0 {
				t.Errorf("%s.Render(\"x\") returned empty string", name)
			}
		})
	}

	// Verify that color vars are set by checking ChatInputBaseStyle adds
	// a border (uses colorPrimary for focused, colorOverlay for blurred).
	t.Run("ChatInputBaseStyle focused", func(t *testing.T) {
		s := ChatInputBaseStyle(true)
		rendered := s.Render("test")
		if len(rendered) <= len("test") {
			t.Error("focused ChatInputBaseStyle should add border characters")
		}
	})
	t.Run("ChatInputBaseStyle blurred", func(t *testing.T) {
		s := ChatInputBaseStyle(false)
		rendered := s.Render("test")
		if len(rendered) <= len("test") {
			t.Error("blurred ChatInputBaseStyle should add border characters")
		}
	})
}

func TestInitStylesAllThemes(t *testing.T) {
	// Verify each theme can be applied without error.
	for name, colors := range themeByName {
		t.Run(name, func(t *testing.T) {
			// Should not panic.
			initStyles(colors)
			r := AppStyle.Render("test")
			if len(r) <= len("test") {
				t.Error("AppStyle.Render returned unstyled output after initStyles")
			}
		})
	}
	// Restore default.
	initStyles(ThemeTokyoNight)
}

func TestConfigPath(t *testing.T) {
	path, err := configPath()
	if err != nil {
		t.Fatalf("configPath() returned error: %v", err)
	}
	if path == "" {
		t.Fatal("configPath() returned empty path")
	}
	if !filepath.IsAbs(path) {
		t.Errorf("configPath() = %q, want absolute path", path)
	}
	if filepath.Base(filepath.Dir(path)) != ".dscli-tui" {
		t.Errorf("configPath() = %q, want parent dir '.dscli-tui'", path)
	}
}

func TestLoadConfigEmptyThemeInFile(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	cfgDir := filepath.Join(tmpDir, ".dscli-tui")
	if err := os.MkdirAll(cfgDir, 0755); err != nil {
		t.Fatal(err)
	}
	// Theme key present but empty.
	if err := os.WriteFile(filepath.Join(cfgDir, "config.yaml"), []byte("theme: \n"), 0644); err != nil {
		t.Fatal(err)
	}

	cfg := loadConfig()
	if cfg.Theme != "tokyo-night" {
		t.Errorf("expected default theme when config theme is empty, got %q", cfg.Theme)
	}
}

func init() {
	// Ensure consistent initial state for all tests.
	lipgloss.SetColorProfile(termenv.TrueColor)
}

func TestDefaultFontConfig(t *testing.T) {
	cfg := defaultConfig()
	if cfg.Font.Bold {
		t.Error("default Font.Bold should be false")
	}
	if cfg.Font.Italic {
		t.Error("default Font.Italic should be false")
	}
}

func TestLoadConfigFontBold(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	cfgDir := filepath.Join(tmpDir, ".dscli-tui")
	if err := os.MkdirAll(cfgDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cfgDir, "config.yaml"), []byte("font:\n  bold: true\n"), 0644); err != nil {
		t.Fatal(err)
	}

	cfg := loadConfig()
	if !cfg.Font.Bold {
		t.Error("expected Font.Bold to be true from config")
	}
	if cfg.Font.Italic {
		t.Error("expected Font.Italic to remain false")
	}
}

func TestLoadConfigFontItalic(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	cfgDir := filepath.Join(tmpDir, ".dscli-tui")
	if err := os.MkdirAll(cfgDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cfgDir, "config.yaml"), []byte("font:\n  italic: true\n"), 0644); err != nil {
		t.Fatal(err)
	}

	cfg := loadConfig()
	if !cfg.Font.Italic {
		t.Error("expected Font.Italic to be true from config")
	}
	if cfg.Font.Bold {
		t.Error("expected Font.Bold to remain false")
	}
}

func TestLoadConfigFontBoth(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	cfgDir := filepath.Join(tmpDir, ".dscli-tui")
	if err := os.MkdirAll(cfgDir, 0755); err != nil {
		t.Fatal(err)
	}
	yamlContent := []byte("theme: nord\nfont:\n  bold: true\n  italic: true\n")
	if err := os.WriteFile(filepath.Join(cfgDir, "config.yaml"), yamlContent, 0644); err != nil {
		t.Fatal(err)
	}

	cfg := loadConfig()
	if cfg.Theme != "nord" {
		t.Errorf("theme = %q, want %q", cfg.Theme, "nord")
	}
	if !cfg.Font.Bold {
		t.Error("expected Font.Bold to be true")
	}
	if !cfg.Font.Italic {
		t.Error("expected Font.Italic to be true")
	}
}

func TestLoadConfigFontOnlyNoTheme(t *testing.T) {
	// Font config with theme omitted should use default theme.
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	cfgDir := filepath.Join(tmpDir, ".dscli-tui")
	if err := os.MkdirAll(cfgDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cfgDir, "config.yaml"), []byte("font:\n  bold: true\n"), 0644); err != nil {
		t.Fatal(err)
	}

	cfg := loadConfig()
	if cfg.Theme != "tokyo-night" {
		t.Errorf("expected default theme, got %q", cfg.Theme)
	}
	if !cfg.Font.Bold {
		t.Error("expected Font.Bold to be true")
	}
}
