package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/yyYank/goodbye/internal/brew"
	"github.com/yyYank/goodbye/internal/config"
	"github.com/yyYank/goodbye/internal/dotfiles"
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

var importDotfilesCmd = &cobra.Command{
	Use:   "dotfiles",
	Short: "Import dotfiles to home directory",
	Long: `Import dotfiles from a synced repository to your home directory.

Reads dotfiles from the local repository (synced via 'goodbye sync')
and copies or symlinks them to your home directory.

Files to import are configured in ~/.goodbye.toml under [dotfiles].`,
	Example: `  # Dry-run (default) - preview what will be imported
  goodbye import dotfiles

  # Actually import
  goodbye import dotfiles --apply

  # Use copy instead of symlink
  goodbye import dotfiles --apply --copy

  # Import without backup
  goodbye import dotfiles --apply --no-backup

  # Continue on errors
  goodbye import dotfiles --apply --continue`,
	RunE: runImportDotfiles,
}

var (
	importDir            string
	importApply          bool
	importVerbose        bool
	importOnly           string
	importSkipTaps       bool
	importContinue       bool
	importMiseFile       string
	importMiseGlobal     bool
	importDotfilesCopy   bool
	importDotfilesNoBack bool
)

func init() {
	rootCmd.AddCommand(importCmd)
	importCmd.AddCommand(importBrewCmd)
	importCmd.AddCommand(importMiseCmd)
	importCmd.AddCommand(importDotfilesCmd)

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

	importDotfilesCmd.Flags().BoolVar(&importApply, "apply", false, "Actually perform the import (default is dry-run)")
	importDotfilesCmd.Flags().BoolVarP(&importVerbose, "verbose", "v", false, "Verbose output")
	importDotfilesCmd.Flags().BoolVar(&importDotfilesCopy, "copy", false, "Copy files instead of creating symlinks")
	importDotfilesCmd.Flags().BoolVar(&importDotfilesNoBack, "no-backup", false, "Do not backup existing files")
	importDotfilesCmd.Flags().BoolVar(&importContinue, "continue", false, "Continue on errors")
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
	}

	return mise.Import(opts)
}

func runImportDotfiles(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Determine symlink setting: config default, unless --copy flag is set
	useSymlink := cfg.Dotfiles.Symlink
	if importDotfilesCopy {
		useSymlink = false
	}

	// Determine backup setting: config default, unless --no-backup flag is set
	useBackup := cfg.Dotfiles.Backup
	if importDotfilesNoBack {
		useBackup = false
	}

	opts := dotfiles.ImportOptions{
		DryRun:   !importApply,
		Verbose:  importVerbose,
		Symlink:  useSymlink,
		Backup:   useBackup,
		Files:    cfg.Dotfiles.Files,
		Continue: importContinue,
	}

	return dotfiles.Import(cfg, opts)
}
