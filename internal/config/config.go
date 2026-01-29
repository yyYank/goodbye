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
	CaskFile          string `toml:"cask_file"`
	FormulaFile       string `toml:"formula_file"`
	FormulaInstallCmd string `toml:"formula_install_cmd"`
	CaskInstallCmd    string `toml:"cask_install_cmd"`
	TapCmd            string `toml:"tap_cmd"`
}

// MiseConfig represents mise-related configuration
type MiseConfig struct {
	Commands     MiseCommandsConfig     `toml:"commands"`
	KnownMappings map[string]string     `toml:"known_mappings"`
}

// MiseCommandsConfig represents mise command configurations
type MiseCommandsConfig struct {
	RegistryCmd      string `toml:"registry_cmd"`
	RegistryJSONCmd  string `toml:"registry_json_cmd"`
	CurrentCmd       string `toml:"current_cmd"`
	ListCmd          string `toml:"list_cmd"`
	InstallCmd       string `toml:"install_cmd"`
	UseGlobalCmd     string `toml:"use_global_cmd"`
	BrewUninstallCmd string `toml:"brew_uninstall_cmd"`
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
				CaskFile:          "cask.txt",
				FormulaFile:       "formula.txt",
				FormulaInstallCmd: "brew install",
				CaskInstallCmd:    "brew install --cask",
				TapCmd:            "brew tap",
			},
		},
		Mise: MiseConfig{
			Commands: MiseCommandsConfig{
				RegistryCmd:      "mise registry",
				RegistryJSONCmd:  "mise registry --json",
				CurrentCmd:       "mise current",
				ListCmd:          "mise list",
				InstallCmd:       "mise install %s@latest",
				UseGlobalCmd:     "mise use -g %s@latest",
				BrewUninstallCmd: "brew uninstall %s",
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
// User config is merged on top of defaults (partial override)
func Load() (*Config, error) {
	defaults := DefaultConfig()

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return defaults, nil
	}

	configPath := filepath.Join(homeDir, ".goodbye.toml")
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return defaults, nil
	}

	var userConfig Config
	if _, err := toml.DecodeFile(configPath, &userConfig); err != nil {
		return nil, err
	}

	return mergeConfig(defaults, &userConfig), nil
}

// mergeConfig merges user config on top of defaults
// Empty string fields keep the default value
// Maps are merged (user values override defaults for same keys)
func mergeConfig(defaults, user *Config) *Config {
	result := defaults

	// Brew Export
	if user.Brew.Export.FormulaCmd != "" {
		result.Brew.Export.FormulaCmd = user.Brew.Export.FormulaCmd
	}
	if user.Brew.Export.CaskCmd != "" {
		result.Brew.Export.CaskCmd = user.Brew.Export.CaskCmd
	}
	if user.Brew.Export.TapCmd != "" {
		result.Brew.Export.TapCmd = user.Brew.Export.TapCmd
	}

	// Brew Import
	if user.Brew.Import.CaskFile != "" {
		result.Brew.Import.CaskFile = user.Brew.Import.CaskFile
	}
	if user.Brew.Import.FormulaFile != "" {
		result.Brew.Import.FormulaFile = user.Brew.Import.FormulaFile
	}
	if user.Brew.Import.FormulaInstallCmd != "" {
		result.Brew.Import.FormulaInstallCmd = user.Brew.Import.FormulaInstallCmd
	}
	if user.Brew.Import.CaskInstallCmd != "" {
		result.Brew.Import.CaskInstallCmd = user.Brew.Import.CaskInstallCmd
	}
	if user.Brew.Import.TapCmd != "" {
		result.Brew.Import.TapCmd = user.Brew.Import.TapCmd
	}

	// Mise Commands
	if user.Mise.Commands.RegistryCmd != "" {
		result.Mise.Commands.RegistryCmd = user.Mise.Commands.RegistryCmd
	}
	if user.Mise.Commands.RegistryJSONCmd != "" {
		result.Mise.Commands.RegistryJSONCmd = user.Mise.Commands.RegistryJSONCmd
	}
	if user.Mise.Commands.CurrentCmd != "" {
		result.Mise.Commands.CurrentCmd = user.Mise.Commands.CurrentCmd
	}
	if user.Mise.Commands.ListCmd != "" {
		result.Mise.Commands.ListCmd = user.Mise.Commands.ListCmd
	}
	if user.Mise.Commands.InstallCmd != "" {
		result.Mise.Commands.InstallCmd = user.Mise.Commands.InstallCmd
	}
	if user.Mise.Commands.UseGlobalCmd != "" {
		result.Mise.Commands.UseGlobalCmd = user.Mise.Commands.UseGlobalCmd
	}
	if user.Mise.Commands.BrewUninstallCmd != "" {
		result.Mise.Commands.BrewUninstallCmd = user.Mise.Commands.BrewUninstallCmd
	}

	// Mise KnownMappings - merge maps (user overrides defaults for same keys)
	if user.Mise.KnownMappings != nil {
		for k, v := range user.Mise.KnownMappings {
			result.Mise.KnownMappings[k] = v
		}
	}

	return result
}
