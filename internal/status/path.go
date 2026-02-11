package status

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/yyYank/goodbye/internal/config"
)

// CheckPaths checks for hardcoded paths that should be replaced with environment variables
func CheckPaths(cfg *config.Config, opts Options) ([]Issue, error) {
	var issues []Issue

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}

	// Get the list of dotfiles to check
	localPath := expandTilde(cfg.Dotfiles.LocalPath, homeDir)
	sourceDir := localPath
	if cfg.Dotfiles.SourceDir != "" {
		sourceDir = filepath.Join(localPath, cfg.Dotfiles.SourceDir)
	}

	// Check each dotfile
	for _, file := range cfg.Dotfiles.Files {
		filePath := filepath.Join(sourceDir, file)

		// Also check files in home directory if they exist
		homeFilePath := filepath.Join(homeDir, file)
		pathsToCheck := []string{filePath, homeFilePath}

		for _, path := range pathsToCheck {
			if _, err := os.Stat(path); os.IsNotExist(err) {
				continue
			}

			fileIssues, err := checkFileForPaths(path, cfg.Status.PathRules, opts)
			if err != nil {
				if opts.Verbose {
					fmt.Printf("Warning: could not check %s: %v\n", path, err)
				}
				continue
			}
			issues = append(issues, fileIssues...)
		}
	}

	return issues, nil
}

// checkFileForPaths checks a single file for path issues
func checkFileForPaths(filePath string, rules []config.PathRule, opts Options) ([]Issue, error) {
	var issues []Issue

	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	lineNum := 0

	for scanner.Scan() {
		lineNum++
		line := scanner.Text()

		// Skip comments
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "#") {
			continue
		}

		for _, rule := range rules {
			if strings.Contains(line, rule.Pattern) {
				// Create the suggested replacement
				suggested := strings.Replace(line, rule.Pattern, rule.Replacement, 1)

				issues = append(issues, Issue{
					Type:        "path",
					File:        filePath,
					Line:        lineNum,
					Description: rule.Description,
					Current:     line,
					Suggestion:  suggested,
				})
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return issues, nil
}

// applyPathFix applies a single path fix to a file
func applyPathFix(issue Issue) error {
	// Read the file
	content, err := os.ReadFile(issue.File)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	lines := strings.Split(string(content), "\n")
	if issue.Line > len(lines) || issue.Line < 1 {
		return fmt.Errorf("invalid line number: %d", issue.Line)
	}

	// Replace the line
	lines[issue.Line-1] = issue.Suggestion

	// Write the file back
	newContent := strings.Join(lines, "\n")
	if err := os.WriteFile(issue.File, []byte(newContent), 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

// expandTilde expands ~ to user's home directory
func expandTilde(path, homeDir string) string {
	if strings.HasPrefix(path, "~") {
		return filepath.Join(homeDir, path[1:])
	}
	return path
}
