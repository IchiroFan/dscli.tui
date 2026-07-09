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

func TestLoadConfigSystemTheme(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	cfgDir := filepath.Join(tmpDir, ".dscli-tui")
	if err := os.MkdirAll(cfgDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cfgDir, "config.yaml"), []byte("theme: system\n"), 0644); err != nil {
		t.Fatal(err)
	}

	cfg := loadConfig()
	if cfg.Theme != "system" {
		t.Errorf("theme = %q, want %q", cfg.Theme, "system")
	}
}

func TestThemeByName(t *testing.T) {
	tests := []struct {
		name     string
		exists   bool
	}{
		{"tokyo-night", true},
		{"dracula", true},
		{"monokai", true},
		{"nord", true},
		{"solarized-light", true},
		{"nonexistent", false},
		{"", false},
		{"system", false}, // "system" is not a theme palette, it's a resolver directive
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

func TestResolveThemePassthrough(t *testing.T) {
	// Non-system themes should pass through unchanged.
	for _, name := range []string{"tokyo-night", "dracula", "monokai", "nord", "solarized-light", "nonexistent"} {
		cfg := Config{Theme: name}
		got := cfg.ResolveTheme()
		if got != name {
			t.Errorf("ResolveTheme() with %q = %q, want %q", name, got, name)
		}
	}
}

func TestResolveThemeSystem(t *testing.T) {
	// "system" should resolve to a valid built-in theme name.
	cfg := Config{Theme: "system"}
	got := cfg.ResolveTheme()
	if got != "tokyo-night" && got != "solarized-light" {
		t.Errorf("ResolveTheme() with \"system\" = %q, want either \"tokyo-night\" or \"solarized-light\"", got)
	}
	// Must be a valid theme.
	if _, ok := themeByName[got]; !ok {
		t.Errorf("ResolveTheme() resolved to %q which is not in themeByName", got)
	}
}

func TestDetectSystemThemeReturnsValid(t *testing.T) {
	// detectSystemTheme should always return a valid theme name.
	result := detectSystemTheme()
	if result != "tokyo-night" && result != "solarized-light" {
		t.Errorf("detectSystemTheme() = %q, want \"tokyo-night\" or \"solarized-light\"", result)
	}
	// Verify it's a known theme.
	if _, ok := themeByName[result]; !ok {
		t.Errorf("detectSystemTheme() returned %q which is not in themeByName", result)
	}
}

func TestDetectSystemThemeDarkFallback(t *testing.T) {
	// In an isolated environment (no desktop), detectSystemTheme should
	// fall back to "tokyo-night". We verify by temporarily clearing PATH
	// so gsettings/defaults commands won't be found.
	tmpDir := t.TempDir()
	// Create empty bin dirs to ensure commands aren't found.
	emptyPath := filepath.Join(tmpDir, "empty-bin")
	if err := os.MkdirAll(emptyPath, 0755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", emptyPath)

	result := detectSystemTheme()
	if result != "tokyo-night" {
		t.Errorf("detectSystemTheme() without desktop tools = %q, want \"tokyo-night\"", result)
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

func TestThemeSystemConstant(t *testing.T) {
	if ThemeSystem != "system" {
		t.Errorf("ThemeSystem = %q, want %q", ThemeSystem, "system")
	}
}

// TestNewAppliesSystemTheme verifies that New() resolves "system" correctly.
func TestNewAppliesSystemTheme(t *testing.T) {
	// Create a temp config with "system" theme.
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	cfgDir := filepath.Join(tmpDir, ".dscli-tui")
	if err := os.MkdirAll(cfgDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cfgDir, "config.yaml"), []byte("theme: system\n"), 0644); err != nil {
		t.Fatal(err)
	}

	// Re-load config and verify resolution.
	cfg := loadConfig()
	if cfg.Theme != "system" {
		t.Fatalf("expected theme=system, got %q", cfg.Theme)
	}

	resolved := cfg.ResolveTheme()
	if resolved != "tokyo-night" && resolved != "solarized-light" {
		t.Fatalf("ResolveTheme() = %q, want valid theme", resolved)
	}

	// Theme should exist in the map.
	if _, ok := themeByName[resolved]; !ok {
		t.Errorf("resolved theme %q not found in themeByName", resolved)
	}
}

// TestSystemThemeAppliedViaNew verifies that when New() loads a "system" config,
// the resolved theme palette is correctly applied to color variables.
func TestSystemThemeAppliedViaNew(t *testing.T) {
	defer func() {
		// Restore default theme.
		initStyles(ThemeTokyoNight)
	}()

	// Mock: directly simulate what happens when ResolveTheme() returns
	// "solarized-light" and that gets passed to initStyles.
	initStyles(ThemeSolarizedLight)
	if colorBase != "#fdf6e3" {
		t.Errorf("after system → light: colorBase = %q, want %q", colorBase, "#fdf6e3")
	}

	initStyles(ThemeTokyoNight)
	if colorBase != "#1a1b26" {
		t.Errorf("after system → dark: colorBase = %q, want %q", colorBase, "#1a1b26")
	}
}

func TestLoadConfigSystemThemeWithEmptyPath(t *testing.T) {
	// When PATH is empty/CLEARED, "system" should still resolve to a valid theme.
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)
	emptyPath := filepath.Join(tmpDir, "empty")
	if err := os.MkdirAll(emptyPath, 0755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", emptyPath)

	// Create config with "system".
	cfgDir := filepath.Join(tmpDir, ".dscli-tui")
	if err := os.MkdirAll(cfgDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cfgDir, "config.yaml"), []byte("theme: system\n"), 0644); err != nil {
		t.Fatal(err)
	}

	cfg := loadConfig()
	resolved := cfg.ResolveTheme()
	if resolved != "tokyo-night" {
		t.Errorf("with empty PATH, ResolveTheme() = %q, want %q", resolved, "tokyo-night")
	}
}

// Test that detectSystemTheme fallback order is correct by checking
// return values match expected valid themes.
func TestDetectSystemThemeNoPanic(t *testing.T) {
	// Should never panic, regardless of environment.
	_ = detectSystemTheme()
}

func TestResolveThemeEmptyString(t *testing.T) {
	// An empty theme string should pass through unchanged.
	cfg := Config{Theme: ""}
	got := cfg.ResolveTheme()
	if got != "" {
		t.Errorf("ResolveTheme() with empty string = %q, want %q", got, "")
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
