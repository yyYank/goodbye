package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/yyYank/goodbye/internal/config"
	"github.com/yyYank/goodbye/internal/mise"
)

var brewCmd = &cobra.Command{
	Use:   "brew",
	Short: "Migrate tools from Homebrew to other package managers",
	Long: `Migrate tools from Homebrew to other package managers.

Use --mise to migrate Homebrew-managed tools that can be replaced by mise.
By default the command runs in dry-run mode — only candidates are shown.`,
	Example: `  # Preview migration candidates (dry-run)
  goodbye brew --mise

  # Actually perform migration
  goodbye brew --mise --apply`,
	RunE: runBrew,
}

var (
	brewMise    bool
	brewApply   bool
	brewVerbose bool
)

func init() {
	rootCmd.AddCommand(brewCmd)

	brewCmd.Flags().BoolVar(&brewMise, "mise", false, "Migrate tools from Homebrew to mise")
	brewCmd.Flags().BoolVar(&brewApply, "apply", false, "Actually perform the migration (default is dry-run)")
	brewCmd.Flags().BoolVarP(&brewVerbose, "verbose", "v", false, "Verbose output")
}

func runBrew(cmd *cobra.Command, args []string) error {
	if !brewMise {
		return fmt.Errorf("please specify a migration target (e.g., --mise)")
	}

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	opts := mise.MigrateOptions{
		DryRun:  !brewApply,
		Verbose: brewVerbose,
	}

	return mise.Migrate(cfg, opts)
}
