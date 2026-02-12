package status

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/yyYank/goodbye/internal/config"
)

// Options represents options for the status command
type Options struct {
	DryRun   bool
	Verbose  bool
	Only     string // "paths", "tools", "dotfiles", or "" for all
	Continue bool
}

// Issue represents a detected issue
type Issue struct {
	Type        string // "path", "tool", "dotfiles"
	File        string // File path where issue was found
	Line        int    // Line number (0 if not applicable)
	Description string // Description of the issue
	Current     string // Current value
	Suggestion  string // Suggested fix
}

// Result represents the result of status checks
type Result struct {
	PathIssues     []Issue
	ToolIssues     []Issue
	DotfilesIssues []Issue
}

// Check performs all status checks and returns the results
func Check(cfg *config.Config, opts Options) (*Result, error) {
	result := &Result{}

	// Path checks
	if opts.Only == "" || opts.Only == "paths" {
		pathIssues, err := CheckPaths(cfg, opts)
		if err != nil && !opts.Continue {
			return nil, fmt.Errorf("path check failed: %w", err)
		}
		result.PathIssues = pathIssues
	}

	// Tool checks
	if opts.Only == "" || opts.Only == "tools" {
		toolIssues, err := CheckTools(cfg, opts)
		if err != nil && !opts.Continue {
			return nil, fmt.Errorf("tool check failed: %w", err)
		}
		result.ToolIssues = toolIssues
	}

	// Dotfiles checks
	if opts.Only == "" || opts.Only == "dotfiles" {
		dotfilesIssues, err := CheckDotfiles(cfg, opts)
		if err != nil && !opts.Continue {
			return nil, fmt.Errorf("dotfiles check failed: %w", err)
		}
		result.DotfilesIssues = dotfilesIssues
	}

	return result, nil
}

// PrintResult prints the status check results
func PrintResult(result *Result, opts Options) {
	totalIssues := len(result.PathIssues) + len(result.ToolIssues) + len(result.DotfilesIssues)

	if totalIssues == 0 {
		fmt.Println("No issues found. Your environment is in sync!")
		return
	}

	fmt.Printf("[status] Found %d issue(s)\n\n", totalIssues)

	// Print path issues
	if len(result.PathIssues) > 0 {
		fmt.Printf("=== Path Issues (%d found) ===\n", len(result.PathIssues))
		for i, issue := range result.PathIssues {
			fmt.Printf("  %d. %s:%d\n", i+1, issue.File, issue.Line)
			fmt.Printf("     Current: %s\n", issue.Current)
			fmt.Printf("     Suggestion: %s\n", issue.Suggestion)
			if opts.Verbose && issue.Description != "" {
				fmt.Printf("     Reason: %s\n", issue.Description)
			}
			fmt.Println()
		}
	}

	// Print tool issues
	if len(result.ToolIssues) > 0 {
		fmt.Printf("=== Tool Issues (%d found) ===\n", len(result.ToolIssues))
		for i, issue := range result.ToolIssues {
			fmt.Printf("  %d. %s - %s\n", i+1, issue.Current, issue.Description)
			fmt.Printf("     Suggestion: %s\n", issue.Suggestion)
			fmt.Println()
		}
	}

	// Print dotfiles issues
	if len(result.DotfilesIssues) > 0 {
		fmt.Printf("=== Dotfiles Issues (%d found) ===\n", len(result.DotfilesIssues))
		for i, issue := range result.DotfilesIssues {
			fmt.Printf("  %d. %s - %s\n", i+1, issue.File, issue.Description)
			fmt.Printf("     Suggestion: %s\n", issue.Suggestion)
			fmt.Println()
		}
	}

	if opts.DryRun {
		fmt.Println("[dry-run] Run with --apply to fix interactively.")
	}
}

// ApplyFixes interactively applies fixes for detected issues
func ApplyFixes(cfg *config.Config, result *Result, opts Options) error {
	reader := bufio.NewReader(os.Stdin)

	// Apply path fixes
	for _, issue := range result.PathIssues {
		fmt.Printf("\nFix path issue in %s:%d?\n", issue.File, issue.Line)
		fmt.Printf("  Current: %s\n", issue.Current)
		fmt.Printf("  Replace with: %s\n", issue.Suggestion)
		fmt.Print("Apply? [y/N/q]: ")

		response, _ := reader.ReadString('\n')
		response = strings.TrimSpace(strings.ToLower(response))

		switch response {
		case "y", "yes":
			if err := applyPathFix(issue); err != nil {
				fmt.Printf("  Error: %v\n", err)
				if !opts.Continue {
					return err
				}
			} else {
				fmt.Println("  Applied!")
			}
		case "q", "quit":
			fmt.Println("Aborted.")
			return nil
		default:
			fmt.Println("  Skipped.")
		}
	}

	// Apply tool fixes
	for _, issue := range result.ToolIssues {
		fmt.Printf("\nInstall missing tool: %s?\n", issue.Current)
		fmt.Printf("  Suggestion: %s\n", issue.Suggestion)
		fmt.Print("Apply? [y/N/q]: ")

		response, _ := reader.ReadString('\n')
		response = strings.TrimSpace(strings.ToLower(response))

		switch response {
		case "y", "yes":
			if err := applyToolFix(issue); err != nil {
				fmt.Printf("  Error: %v\n", err)
				if !opts.Continue {
					return err
				}
			} else {
				fmt.Println("  Applied!")
			}
		case "q", "quit":
			fmt.Println("Aborted.")
			return nil
		default:
			fmt.Println("  Skipped.")
		}
	}

	// Apply dotfiles fixes
	for _, issue := range result.DotfilesIssues {
		fmt.Printf("\nFix dotfiles issue: %s?\n", issue.File)
		fmt.Printf("  Issue: %s\n", issue.Description)
		fmt.Printf("  Suggestion: %s\n", issue.Suggestion)
		fmt.Print("Apply? [y/N/q]: ")

		response, _ := reader.ReadString('\n')
		response = strings.TrimSpace(strings.ToLower(response))

		switch response {
		case "y", "yes":
			if err := applyDotfilesFix(cfg, issue); err != nil {
				fmt.Printf("  Error: %v\n", err)
				if !opts.Continue {
					return err
				}
			} else {
				fmt.Println("  Applied!")
			}
		case "q", "quit":
			fmt.Println("Aborted.")
			return nil
		default:
			fmt.Println("  Skipped.")
		}
	}

	fmt.Println("\nAll fixes processed.")
	return nil
}
