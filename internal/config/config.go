package config

import (
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

// Config represents the ~/.goodbye.toml configuration
type Config struct {
	Brew     BrewConfig     `toml:"brew"`
	Mise     MiseConfig     `toml:"mise"`
	Dotfiles DotfilesConfig `toml:"dotfiles"`
	Status   StatusConfig   `toml:"status"`
}

// StatusConfig represents status command configuration
type StatusConfig struct {
	PathRules  []PathRule  `toml:"path_rules"`
	ToolChecks []ToolCheck `toml:"tool_checks"`
}

// PathRule represents a path replacement rule for status checks
type PathRule struct {
	Pattern     string `toml:"pattern"`     // Pattern to detect (plain string match)
	Replacement string `toml:"replacement"` // Suggested replacement
	Description string `toml:"description"` // Description of the rule
}

// ToolCheck represents a tool installation check
type ToolCheck struct {
	Name    string `toml:"name"`    // Tool name
	Command string `toml:"command"` // Command to check if installed
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

// DotfilesConfig represents dotfiles-related configuration
type DotfilesConfig struct {
	Repository  string          `toml:"repository"`
	LocalPath   string          `toml:"local_path"`
	SourceDir   string          `toml:"source_dir"`
	Files       []string        `toml:"files"`
	Directories []DirectoryMap  `toml:"directories"`
	Symlink     bool            `toml:"symlink"`
	Backup      bool            `toml:"backup"`
}

// DirectoryMap represents a directory mapping from source to target
type DirectoryMap struct {
	Source string `toml:"source"` // Source directory relative to dotfiles repo (e.g., "macOS/claude")
	Target string `toml:"target"` // Target directory relative to home (e.g., ".claude")
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
		Dotfiles: DotfilesConfig{
			Repository: "",
			LocalPath:  "~/.dotfiles",
			SourceDir:  "",
			Files: []string{
				".zshrc",
				".bashrc",
				".bash_profile",
				".vimrc",
				".gitconfig",
				".tmux.conf",
			},
			Directories: []DirectoryMap{},
			Symlink:     true,
			Backup:      true,
		},
		Status: StatusConfig{
			PathRules: []PathRule{
				{
					Pattern:     "/usr/local/share/",
					Replacement: "$HOMEBREW_PREFIX/share/",
					Description: "Replace hardcoded Intel Homebrew path with architecture-independent variable",
				},
				{
					Pattern:     "/usr/local/bin/",
					Replacement: "$HOMEBREW_PREFIX/bin/",
					Description: "Replace hardcoded Intel Homebrew bin path with architecture-independent variable",
				},
				{
					Pattern:     "/opt/homebrew/share/",
					Replacement: "$HOMEBREW_PREFIX/share/",
					Description: "Replace hardcoded Apple Silicon Homebrew path with architecture-independent variable",
				},
				{
					Pattern:     "/opt/homebrew/bin/",
					Replacement: "$HOMEBREW_PREFIX/bin/",
					Description: "Replace hardcoded Apple Silicon Homebrew bin path with architecture-independent variable",
				},
			},
			ToolChecks: []ToolCheck{
				{Name: "mise", Command: "mise --version"},
				{Name: "fzf", Command: "fzf --version"},
				{Name: "starship", Command: "starship --version"},
				{Name: "zoxide", Command: "zoxide --version"},
				{Name: "eza", Command: "eza --version"},
				{Name: "bat", Command: "bat --version"},
				{Name: "fd", Command: "fd --version"},
				{Name: "ripgrep", Command: "rg --version"},
			},
		},
	}
}

// Save saves the configuration to ~/.goodbye.toml
func Save(cfg *Config) error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	configPath := filepath.Join(homeDir, ".goodbye.toml")
	file, err := os.Create(configPath)
	if err != nil {
		return err
	}
	defer file.Close()

	encoder := toml.NewEncoder(file)
	return encoder.Encode(cfg)
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

	// Dotfiles
	if user.Dotfiles.Repository != "" {
		result.Dotfiles.Repository = user.Dotfiles.Repository
	}
	if user.Dotfiles.LocalPath != "" {
		result.Dotfiles.LocalPath = user.Dotfiles.LocalPath
	}
	if user.Dotfiles.SourceDir != "" {
		result.Dotfiles.SourceDir = user.Dotfiles.SourceDir
	}
	if len(user.Dotfiles.Files) > 0 {
		result.Dotfiles.Files = user.Dotfiles.Files
	}
	if len(user.Dotfiles.Directories) > 0 {
		result.Dotfiles.Directories = user.Dotfiles.Directories
	}
	// For bool fields, only override if user has set dotfiles section
	// (indicated by having a non-empty Repository or LocalPath or SourceDir or Files or Directories)
	hasDotfilesSection := user.Dotfiles.Repository != "" || user.Dotfiles.LocalPath != "" || user.Dotfiles.SourceDir != "" || len(user.Dotfiles.Files) > 0 || len(user.Dotfiles.Directories) > 0
	if hasDotfilesSection {
		result.Dotfiles.Symlink = user.Dotfiles.Symlink
		result.Dotfiles.Backup = user.Dotfiles.Backup
	}

	// Status - merge path rules and tool checks (user values extend defaults)
	if len(user.Status.PathRules) > 0 {
		result.Status.PathRules = append(result.Status.PathRules, user.Status.PathRules...)
	}
	if len(user.Status.ToolChecks) > 0 {
		result.Status.ToolChecks = append(result.Status.ToolChecks, user.Status.ToolChecks...)
	}

	return result
}
