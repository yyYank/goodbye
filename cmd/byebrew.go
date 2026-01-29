package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/yyYank/goodbye/internal/config"
	"github.com/yyYank/goodbye/internal/mise"
)

var goodbyebrewCmd = &cobra.Command{
	Use:   "goodbyebrew",
	Short: "Migrate from Homebrew to other package managers",
	Long: `Migrate tools from Homebrew to other package managers like mise or asdf.

This command helps you gradually migrate version-managed tools from Homebrew
to dedicated version managers.`,
}

var goodbyebrewMiseCmd = &cobra.Command{
	Use:   "mise",
	Short: "Migrate from Homebrew to mise",
	Long: `Migrate Homebrew-managed tools to mise.

This command:
1. Gets your Homebrew formula list
2. Normalizes names (e.g., python@3.12 -> python)
3. Matches against mise registry
4. Shows migration candidates
5. Asks for confirmation
6. Installs with mise
7. Verifies installation
8. Uninstalls from Homebrew (only successful ones)`,
	Example: `  # Dry-run (default) - see migration candidates
  goodbye goodbyebrew mise

  # Actually perform migration
  goodbye goodbyebrew mise --apply`,
	RunE: rungoodbyebrewMise,
}

var goodbyebrewAsdfCmd = &cobra.Command{
	Use:   "asdf",
	Short: "Migrate from Homebrew to asdf",
	Long: `Migrate Homebrew-managed tools to asdf.

Note: asdf requires explicit version specification, so this command
works based on .tool-versions file rather than fully automatic migration.`,
	Example: `  # Dry-run (default)
  goodbye goodbyebrew asdf

  # Actually perform migration
  goodbye goodbyebrew asdf --apply`,
	RunE: rungoodbyebrewAsdf,
}

var (
	goodbyebrewApply   bool
	goodbyebrewVerbose bool
)

func init() {
	rootCmd.AddCommand(goodbyebrewCmd)
	goodbyebrewCmd.AddCommand(goodbyebrewMiseCmd)
	goodbyebrewCmd.AddCommand(goodbyebrewAsdfCmd)

	// Shared flags for goodbyebrew subcommands
	goodbyebrewMiseCmd.Flags().BoolVar(&goodbyebrewApply, "apply", false, "Actually perform the migration (default is dry-run)")
	goodbyebrewMiseCmd.Flags().BoolVarP(&goodbyebrewVerbose, "verbose", "v", false, "Verbose output")

	goodbyebrewAsdfCmd.Flags().BoolVar(&goodbyebrewApply, "apply", false, "Actually perform the migration (default is dry-run)")
	goodbyebrewAsdfCmd.Flags().BoolVarP(&goodbyebrewVerbose, "verbose", "v", false, "Verbose output")
}

func rungoodbyebrewMise(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	opts := mise.MigrateOptions{
		DryRun:  !goodbyebrewApply,
		Verbose: goodbyebrewVerbose,
	}

	return mise.Migrate(cfg, opts)
}

func rungoodbyebrewAsdf(cmd *cobra.Command, args []string) error {
	// TODO: Implement asdf migration
	fmt.Println("asdf migration is not yet implemented.")
	fmt.Println("asdf requires .tool-versions file for version specification.")
	fmt.Println("Please use 'goodbye goodbyebrew mise' for now, or contribute to implement this feature!")
	return nil
}
