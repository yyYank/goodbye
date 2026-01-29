package config

import (
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

// Config represents the ~/.goodbye.toml configuration
type Config struct {
	Brew BrewConfig `toml:"brew"`
	Mise MiseConfig `toml:"mise"`
}

// BrewConfig represents brew-related configuration
type BrewConfig struct {
	Export BrewExportConfig `toml:"export"`
	Import BrewImportConfig `toml:"import"`
}

// BrewExportConfig represents brew export command configuration
type BrewExportConfig struct {
	FormulaCmd string `toml:"formula_cmd"`
	CaskCmd    string `toml:"cask_cmd"`
	TapCmd     string `toml:"tap_cmd"`
}

// BrewImportConfig represents brew import command configuration
type BrewImportConfig struct {
	CaskFile    string `toml:"cask_file"`
	FormulaFile string `toml:"formula_file"`
}

// MiseConfig represents mise-related configuration
type MiseConfig struct {
	Commands     MiseCommandsConfig     `toml:"commands"`
	KnownMappings map[string]string     `toml:"known_mappings"`
}

// MiseCommandsConfig represents mise command configurations
type MiseCommandsConfig struct {
	RegistryCmd     string `toml:"registry_cmd"`
	RegistryJSONCmd string `toml:"registry_json_cmd"`
	CurrentCmd      string `toml:"current_cmd"`
	ListCmd         string `toml:"list_cmd"`
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
			Import: BrewImportConfig{
				CaskFile:    "cask.txt",
				FormulaFile: "formula.txt",
			},
		},
		Mise: MiseConfig{
			Commands: MiseCommandsConfig{
				RegistryCmd:     "mise registry",
				RegistryJSONCmd: "mise registry --json",
				CurrentCmd:      "mise current",
				ListCmd:         "mise list",
			},
			KnownMappings: map[string]string{
				"node":      "node",
				"nodejs":    "node",
				"python":    "python",
				"python3":   "python",
				"ruby":      "ruby",
				"go":        "go",
				"golang":    "go",
				"rust":      "rust",
				"rustup":    "rust",
				"java":      "java",
				"openjdk":   "java",
				"deno":      "deno",
				"bun":       "bun",
				"terraform": "terraform",
				"kubectl":   "kubectl",
				"helm":      "helm",
				"awscli":    "awscli",
				"yarn":      "yarn",
				"pnpm":      "pnpm",
				"gradle":    "gradle",
				"maven":     "maven",
				"kotlin":    "kotlin",
				"scala":     "scala",
				"elixir":    "elixir",
				"erlang":    "erlang",
				"lua":       "lua",
				"luajit":    "luajit",
				"perl":      "perl",
				"php":       "php",
				"zig":       "zig",
				"nim":       "nim",
				"crystal":   "crystal",
				"julia":     "julia",
				"r":         "r",
				"dotnet":    "dotnet",
				"flutter":   "flutter",
				"dart":      "dart",
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
