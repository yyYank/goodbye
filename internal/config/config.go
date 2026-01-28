package config

import (
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

// Config represents the ~/.goodbye.toml configuration
type Config struct {
	Brew BrewConfig `toml:"brew"`
}

// BrewConfig represents brew-related configuration
type BrewConfig struct {
	Export BrewExportConfig `toml:"export"`
}

// BrewExportConfig represents brew export command configuration
type BrewExportConfig struct {
	FormulaCmd string `toml:"formula_cmd"`
	CaskCmd    string `toml:"cask_cmd"`
	TapCmd     string `toml:"tap_cmd"`
}

// DefaultConfig returns the default configuration
func DefaultConfig() *Config {
	return &Config{
		Brew: BrewConfig{
			Export: BrewExportConfig{
				FormulaCmd: "brew list --installed-on-request",
				CaskCmd:    "brew list --cask",
				TapCmd:     "brew tap",
			},
		},
	}
}

// Load loads the configuration from ~/.goodbye.toml
// If the file does not exist, returns the default configuration
func Load() (*Config, error) {
	config := DefaultConfig()

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return config, nil
	}

	configPath := filepath.Join(homeDir, ".goodbye.toml")
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return config, nil
	}

	if _, err := toml.DecodeFile(configPath, config); err != nil {
		return nil, err
	}

	return config, nil
}
