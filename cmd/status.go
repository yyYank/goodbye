package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/yyYank/goodbye/internal/config"
	"github.com/yyYank/goodbye/internal/status"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Check environment drift and sync status",
	Long: `Check for environment drift and dotfiles sync status.

This command detects:
- Hardcoded absolute paths that should use environment variables
- Tools declared in dotfiles that are not installed
- Dotfiles sync status (broken symlinks, uncommitted changes, etc.)

By default, this command runs in dry-run mode and only reports issues.
Use --apply to interactively fix detected issues.`,
	Example: `  # Check all status (dry-run)
  goodbye status

  # Check and interactively fix issues
  goodbye status --apply

  # Check only path issues
  goodbye status --only paths

  # Check only tool issues
  goodbye status --only tools

  # Check only dotfiles status
  goodbye status --only dotfiles

  # Verbose output
  goodbye status --verbose`,
	RunE: runStatus,
}

var (
	statusApply    bool
	statusVerbose  bool
	statusOnly     string
	statusContinue bool
)

func init() {
	rootCmd.AddCommand(statusCmd)

	statusCmd.Flags().BoolVar(&statusApply, "apply", false, "Interactively apply fixes (default is dry-run)")
	statusCmd.Flags().BoolVarP(&statusVerbose, "verbose", "v", false, "Verbose output")
	statusCmd.Flags().StringVar(&statusOnly, "only", "", "Check only specific type (paths, tools, or dotfiles)")
	statusCmd.Flags().BoolVar(&statusContinue, "continue", false, "Continue on errors")
}

func runStatus(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Validate --only flag
	if statusOnly != "" && statusOnly != "paths" && statusOnly != "tools" && statusOnly != "dotfiles" {
		return fmt.Errorf("invalid --only value: %s (must be 'paths', 'tools', or 'dotfiles')", statusOnly)
	}

	opts := status.Options{
		DryRun:   !statusApply,
		Verbose:  statusVerbose,
		Only:     statusOnly,
		Continue: statusContinue,
	}

	fmt.Println("[status] Checking environment drift...")
	fmt.Println()

	result, err := status.Check(cfg, opts)
	if err != nil {
		return fmt.Errorf("status check failed: %w", err)
	}

	status.PrintResult(result, opts)

	if statusApply {
		totalIssues := len(result.PathIssues) + len(result.ToolIssues) + len(result.DotfilesIssues)
		if totalIssues > 0 {
			fmt.Println()
			if err := status.ApplyFixes(cfg, result, opts); err != nil {
				return fmt.Errorf("failed to apply fixes: %w", err)
			}
		}
	}

	return nil
}
