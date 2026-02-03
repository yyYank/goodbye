package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "goodbye",
	Short: "A CLI tool for managing development environment package migrations",
	Long: `goodbye is a CLI tool for macOS that helps organize and migrate development
environment package management step by step.

Key features:
  - Homebrew export / import
  - Homebrew -> mise gradual migration (brew --mise)
  - User-defined commands for flexible retrieval
  - Future extensions (uv, etc.) friendly structure

All commands are dry-run by default. Use --apply to make actual changes.`,
}

// Execute runs the root command
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	// Global flags can be added here if needed
}
