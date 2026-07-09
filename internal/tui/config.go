// Package tui implements the Bubble Tea application for dscli.tui.
//
// Config loads the user's preferences from ~/.dscli-tui/config.yaml.
package tui

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// ThemeSystem is the config value that tells dscli.tui to auto-detect the OS
// color scheme preference and choose between a dark or light theme.
const ThemeSystem = "system"

// Config represents ~/.dscli-tui/config.yaml.
type Config struct {
	Theme string `yaml:"theme"` // theme name: tokyo-night, dracula, monokai, nord, solarized-light, or "system"
}

// defaultConfig returns the default configuration.
func defaultConfig() Config {
	return Config{Theme: "tokyo-night"}
}

// configPath returns the absolute path to ~/.dscli-tui/config.yaml.
func configPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".dscli-tui", "config.yaml"), nil
}

// loadConfig reads the config file from the user's home directory.
// If the file does not exist or cannot be parsed, defaults are returned
// without error (the user may not have created a config yet).
func loadConfig() Config {
	cfg := defaultConfig()

	path, err := configPath()
	if err != nil {
		return cfg
	}

	data, err := os.ReadFile(path)
	if err != nil {
		// File doesn't exist or can't be read — use defaults.
		return cfg
	}

	var fileCfg Config
	if err := yaml.Unmarshal(data, &fileCfg); err != nil {
		// Invalid YAML — use defaults.
		return cfg
	}

	if fileCfg.Theme != "" {
		cfg.Theme = fileCfg.Theme
	}
	return cfg
}

// ResolveTheme resolves the effective theme name from the config value.
// If the config value is "system", it auto-detects the OS color scheme
// preference and returns the appropriate built-in theme name.
func (c Config) ResolveTheme() string {
	if c.Theme != ThemeSystem {
		return c.Theme
	}
	return detectSystemTheme()
}

// detectSystemTheme detects the OS color scheme preference.
// Returns "tokyo-night" (dark) or "solarized-light" (light).
func detectSystemTheme() string {
	// 1. GNOME 42+: gsettings get org.gnome.desktop.interface color-scheme
	out, err := exec.Command("gsettings", "get", "org.gnome.desktop.interface", "color-scheme").Output()
	if err == nil {
		s := strings.TrimSpace(string(out))
		if s == "'prefer-dark'" {
			return "tokyo-night"
		}
		if s == "'prefer-light'" {
			return "solarized-light"
		}
		// "default" — fall through to gtk-theme check
	}

	// 2. GTK theme name (older GNOME, Cinnamon, Budgie, XFCE)
	out, err = exec.Command("gsettings", "get", "org.gnome.desktop.interface", "gtk-theme").Output()
	if err == nil {
		s := strings.ToLower(strings.TrimSpace(string(out)))
		if strings.Contains(s, "dark") {
			return "tokyo-night"
		}
		if strings.Contains(s, "light") {
			return "solarized-light"
		}
	}

	// 3. macOS
	out, err = exec.Command("defaults", "read", "-g", "AppleInterfaceStyle").Output()
	if err == nil {
		s := strings.TrimSpace(string(out))
		if s == "Dark" {
			return "tokyo-night"
		}
	}

	// Default to dark (safe for terminal users)
	return "tokyo-night"
}
