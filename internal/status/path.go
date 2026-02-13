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
				suggested := buildPathSuggestion(line, rule.Pattern, rule.Replacement)

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
	if err := validateSuggestedPath(issue.Suggestion); err != nil {
		return err
	}

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

// buildPathSuggestion creates a replacement line and avoids duplicated path segments
// like ".../share/share/..." when replacement and suffix overlap.
func buildPathSuggestion(line, pattern, replacement string) string {
	idx := strings.Index(line, pattern)
	if idx < 0 {
		return line
	}

	suffix := line[idx+len(pattern):]
	trimmedSuffix := strings.TrimPrefix(suffix, "/")
	trimmedReplacement := strings.TrimSuffix(replacement, "/")

	lastSegment := filepath.Base(trimmedReplacement)
	if lastSegment != "." && lastSegment != "/" && strings.HasPrefix(trimmedSuffix, lastSegment+"/") {
		trimmedSuffix = strings.TrimPrefix(trimmedSuffix, lastSegment+"/")
	}

	var combined string
	if strings.HasSuffix(replacement, "/") {
		combined = replacement + trimmedSuffix
	} else if trimmedSuffix != "" {
		combined = replacement + "/" + trimmedSuffix
	} else {
		combined = replacement
	}

	return line[:idx] + combined
}

func validateSuggestedPath(line string) error {
	pathToken, ok := extractPathToken(line)
	if !ok {
		return nil
	}

	expanded := expandKnownVariables(pathToken)
	if strings.Contains(expanded, "$") {
		return nil
	}

	if _, err := os.Stat(expanded); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("skipped fix: replacement path does not exist: %s", pathToken)
		}
		return fmt.Errorf("failed to validate replacement path %s: %w", pathToken, err)
	}

	return nil
}

func extractPathToken(line string) (string, bool) {
	fields := strings.Fields(line)
	for _, field := range fields {
		token := strings.Trim(field, "\"'")
		if strings.Contains(token, "/") {
			return token, true
		}
	}
	return "", false
}

func expandKnownVariables(path string) string {
	result := path
	homebrewPrefix := resolveHomebrewPrefix()
	if homebrewPrefix != "" {
		result = strings.ReplaceAll(result, "$HOMEBREW_PREFIX", homebrewPrefix)
		result = strings.ReplaceAll(result, "${HOMEBREW_PREFIX}", homebrewPrefix)
	}
	return os.ExpandEnv(result)
}

func resolveHomebrewPrefix() string {
	if prefix := os.Getenv("HOMEBREW_PREFIX"); prefix != "" {
		return prefix
	}

	candidates := []string{"/opt/homebrew", "/usr/local"}
	for _, candidate := range candidates {
		if info, err := os.Stat(candidate); err == nil && info.IsDir() {
			return candidate
		}
	}

	return ""
}
