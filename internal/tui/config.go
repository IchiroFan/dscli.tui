// Package tui implements the Bubble Tea application for dscli.tui.
//
// Config loads the user's preferences from ~/.dscli-tui/config.yaml.
package tui

import (
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Config represents ~/.dscli-tui/config.yaml.
type Config struct {
	Theme string `yaml:"theme"` // theme name: tokyo-night, dracula, monokai, nord, solarized-light
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
