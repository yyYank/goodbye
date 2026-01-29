package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/yyYank/goodbye/internal/brew"
	"github.com/yyYank/goodbye/internal/config"
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

Reads .mise.toml, .tool-versions, or brew export files (formula.txt)
and installs the tools on the current system.`,
	Example: `  # Dry-run (default) - preview what will be imported
  goodbye import mise --dir ~/goodbye-export

  # Actually import
  goodbye import mise --dir ~/goodbye-export --apply

  # Import from specific file
  goodbye import mise --dir ~/goodbye-export --file .tool-versions --apply

  # Import and set as global
  goodbye import mise --dir ~/goodbye-export --apply --global

  # Import from brew export files
  goodbye import mise --from brew --dir ~/goodbye-export-brew

  # Import from brew with specific version
  goodbye import mise --from brew --dir ~/goodbye-export-brew --version 3.12 --apply

  # Continue on errors
  goodbye import mise --dir ~/goodbye-export --apply --continue`,
	RunE: runImportMise,
}

var (
	importDir         string
	importApply       bool
	importVerbose     bool
	importOnly        string
	importSkipTaps    bool
	importContinue    bool
	importMiseFile    string
	importMiseGlobal  bool
	importMiseFrom    string
	importMiseVersion string
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
	importMiseCmd.Flags().StringVar(&importMiseFrom, "from", "", "Import source format (e.g., 'brew' to import from brew export files)")
	importMiseCmd.Flags().StringVar(&importMiseVersion, "version", "latest", "Version to install (default: latest)")
}

func runImportBrew(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	opts := brew.ImportOptions{
		Dir:      importDir,
		DryRun:   !importApply,
		Verbose:  importVerbose,
		Only:     importOnly,
		SkipTaps: importSkipTaps,
		Continue: importContinue,
	}

	return brew.Import(cfg, opts)
}

func runImportMise(cmd *cobra.Command, args []string) error {
	opts := mise.ImportOptions{
		Dir:      importDir,
		File:     importMiseFile,
		DryRun:   !importApply,
		Verbose:  importVerbose,
		Continue: importContinue,
		Global:   importMiseGlobal,
		FromBrew: importMiseFrom == "brew",
		Version:  importMiseVersion,
	}

	return mise.Import(opts)
}
