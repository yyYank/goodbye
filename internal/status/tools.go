package status

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/yyYank/goodbye/internal/config"
)

// CheckTools checks if declared tools in dotfiles are actually installed
func CheckTools(cfg *config.Config, opts Options) ([]Issue, error) {
	var issues []Issue

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}

	// Build a map of tools to check
	toolsToCheck := make(map[string]config.ToolCheck)
	for _, tc := range cfg.Status.ToolChecks {
		toolsToCheck[tc.Name] = tc
	}

	// Find tools referenced in dotfiles
	referencedTools := findReferencedTools(cfg, homeDir, opts)

	// Check each referenced tool
	for toolName := range referencedTools {
		tc, exists := toolsToCheck[toolName]
		if !exists {
			// Tool not in our check list, skip
			continue
		}

		// Check if tool is installed
		if !isToolInstalled(tc.Command, opts) {
			issues = append(issues, Issue{
				Type:        "tool",
				Description: "declared in dotfiles but not installed",
				Current:     toolName,
				Suggestion:  fmt.Sprintf("Install with 'brew install %s' or 'mise use -g %s'", toolName, toolName),
			})
		}
	}

	return issues, nil
}

// findReferencedTools finds tools referenced in dotfiles
func findReferencedTools(cfg *config.Config, homeDir string, opts Options) map[string]bool {
	referenced := make(map[string]bool)

	localPath := expandTilde(cfg.Dotfiles.LocalPath, homeDir)
	sourceDir := localPath
	if cfg.Dotfiles.SourceDir != "" {
		sourceDir = filepath.Join(localPath, cfg.Dotfiles.SourceDir)
	}

	// Build list of tool names to look for
	toolNames := make([]string, 0, len(cfg.Status.ToolChecks))
	for _, tc := range cfg.Status.ToolChecks {
		toolNames = append(toolNames, tc.Name)
	}

	// Check each dotfile
	for _, file := range cfg.Dotfiles.Files {
		// Check both source and home paths
		paths := []string{
			filepath.Join(sourceDir, file),
			filepath.Join(homeDir, file),
		}

		for _, filePath := range paths {
			if _, err := os.Stat(filePath); os.IsNotExist(err) {
				continue
			}

			found := findToolsInFile(filePath, toolNames, opts)
			for tool := range found {
				referenced[tool] = true
			}
		}
	}

	return referenced
}

// findToolsInFile searches for tool references in a file
func findToolsInFile(filePath string, toolNames []string, opts Options) map[string]bool {
	found := make(map[string]bool)

	file, err := os.Open(filePath)
	if err != nil {
		if opts.Verbose {
			fmt.Printf("Warning: could not open %s: %v\n", filePath, err)
		}
		return found
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()

		// Skip comments
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "#") {
			continue
		}

		for _, toolName := range toolNames {
			// Look for tool references:
			// - eval "$(starship init zsh)"
			// - source /path/to/fzf.zsh
			// - zoxide init
			// - mise activate
			patterns := []string{
				toolName + " ",           // "starship init"
				toolName + "\"",          // in eval "$(starship"
				"/" + toolName,           // /path/to/starship
				toolName + ".zsh",        // fzf.zsh
				toolName + ".bash",       // fzf.bash
				"eval \"$(" + toolName,   // eval "$(starship
				"source <(" + toolName,   // source <(starship
			}

			for _, pattern := range patterns {
				if strings.Contains(line, pattern) {
					found[toolName] = true
					break
				}
			}
		}
	}

	return found
}

// isToolInstalled checks if a tool is installed by running its version command
func isToolInstalled(command string, opts Options) bool {
	parts := strings.Fields(command)
	if len(parts) == 0 {
		return false
	}

	cmd := exec.Command(parts[0], parts[1:]...)
	err := cmd.Run()
	return err == nil
}

// applyToolFix attempts to install a missing tool
func applyToolFix(issue Issue) error {
	toolName := issue.Current

	// Try mise first, then brew
	fmt.Printf("  Attempting to install %s with mise...\n", toolName)
	miseCmd := exec.Command("mise", "use", "-g", toolName+"@latest")
	miseCmd.Stdout = os.Stdout
	miseCmd.Stderr = os.Stderr
	if err := miseCmd.Run(); err == nil {
		return nil
	}

	fmt.Printf("  mise failed, trying brew...\n")
	brewCmd := exec.Command("brew", "install", toolName)
	brewCmd.Stdout = os.Stdout
	brewCmd.Stderr = os.Stderr
	if err := brewCmd.Run(); err != nil {
		return fmt.Errorf("failed to install with both mise and brew: %w", err)
	}

	return nil
}
