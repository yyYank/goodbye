package cmd

import (
	"github.com/spf13/cobra"
	"github.com/yyYank/goodbye/internal/brew"
	"github.com/yyYank/goodbye/internal/mise"
)

var importCmd = &cobra.Command{
	Use:   "import",
	Short: "Import package manager configurations",
	Long:  `Import package manager configurations from exported files.`,
}

var importBrewCmd = &cobra.Command{
	Use:   "brew",
	Short: "Import Homebrew configuration",
	Long: `Import a Homebrew environment from exported files.

Reads the files created by 'goodbye export brew' and installs
the packages on the current system.`,
	Example: `  # Dry-run (default) - preview what will be imported
  goodbye import brew --dir ~/goodbye-export

  # Actually import
  goodbye import brew --dir ~/goodbye-export --apply

  # Import only formulas
  goodbye import brew --dir ~/goodbye-export --only formula --apply

  # Import without taps
  goodbye import brew --dir ~/goodbye-export --skip-taps --apply

  # Continue on errors
  goodbye import brew --dir ~/goodbye-export --apply --continue`,
	RunE: runImportBrew,
}

var importMiseCmd = &cobra.Command{
	Use:   "mise",
	Short: "Import mise configuration",
	Long: `Import mise tools from a configuration file.

Reads .mise.toml or .tool-versions files and installs
the tools on the current system.`,
	Example: `  # Dry-run (default) - preview what will be imported
  goodbye import mise --dir ~/goodbye-export

  # Actually import
  goodbye import mise --dir ~/goodbye-export --apply

  # Import from specific file
  goodbye import mise --dir ~/goodbye-export --file .tool-versions --apply

  # Import and set as global
  goodbye import mise --dir ~/goodbye-export --apply --global

  # Continue on errors
  goodbye import mise --dir ~/goodbye-export --apply --continue`,
	RunE: runImportMise,
}

var (
	importDir        string
	importApply      bool
	importVerbose    bool
	importOnly       string
	importSkipTaps   bool
	importContinue   bool
	importMiseFile   string
	importMiseGlobal bool
)

func init() {
	rootCmd.AddCommand(importCmd)
	importCmd.AddCommand(importBrewCmd)
	importCmd.AddCommand(importMiseCmd)

	importBrewCmd.Flags().StringVar(&importDir, "dir", ".", "Directory containing exported files")
	importBrewCmd.Flags().BoolVar(&importApply, "apply", false, "Actually perform the import (default is dry-run)")
	importBrewCmd.Flags().BoolVarP(&importVerbose, "verbose", "v", false, "Verbose output")
	importBrewCmd.Flags().StringVar(&importOnly, "only", "", "Import only specific type (formula, cask, or tap)")
	importBrewCmd.Flags().BoolVar(&importSkipTaps, "skip-taps", false, "Skip importing taps")
	importBrewCmd.Flags().BoolVar(&importContinue, "continue", false, "Continue on errors")

	importMiseCmd.Flags().StringVar(&importDir, "dir", ".", "Directory containing exported files")
	importMiseCmd.Flags().BoolVar(&importApply, "apply", false, "Actually perform the import (default is dry-run)")
	importMiseCmd.Flags().BoolVarP(&importVerbose, "verbose", "v", false, "Verbose output")
	importMiseCmd.Flags().StringVar(&importMiseFile, "file", "", "Specific file to import (e.g., .mise.toml or .tool-versions)")
	importMiseCmd.Flags().BoolVar(&importMiseGlobal, "global", false, "Set imported tools as global")
	importMiseCmd.Flags().BoolVar(&importContinue, "continue", false, "Continue on errors")
}

func runImportBrew(cmd *cobra.Command, args []string) error {
	opts := brew.ImportOptions{
		Dir:      importDir,
		DryRun:   !importApply,
		Verbose:  importVerbose,
		Only:     importOnly,
		SkipTaps: importSkipTaps,
		Continue: importContinue,
	}

	return brew.Import(opts)
}

func runImportMise(cmd *cobra.Command, args []string) error {
	opts := mise.ImportOptions{
		Dir:      importDir,
		File:     importMiseFile,
		DryRun:   !importApply,
		Verbose:  importVerbose,
		Continue: importContinue,
		Global:   importMiseGlobal,
	}

	return mise.Import(opts)
}
