package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/yyYank/goodbye/internal/brew"
	"github.com/yyYank/goodbye/internal/config"
	"github.com/yyYank/goodbye/internal/mise"
	"github.com/yyYank/goodbye/internal/npm"
)

var exportCmd = &cobra.Command{
	Use:   "export",
	Short: "Export package manager configurations",
	Long:  `Export package manager configurations to files for backup or migration.`,
}

var exportBrewCmd = &cobra.Command{
	Use:   "brew",
	Short: "Export Homebrew configuration",
	Long: `Export the current Homebrew environment to files.

By default, exports:
  - formula: brew list --installed-on-request
  - cask: brew list --cask
  - tap: brew tap

These commands can be customized in ~/.goodbye.toml`,
	Example: `  # Dry-run (default) - preview what will be exported
  goodbye export brew

  # Export to specific directory
  goodbye export brew --dir ~/goodbye-export

  # Actually export
  goodbye export brew --dir ~/goodbye-export --apply`,
	RunE: runExportBrew,
}

var exportNpmCmd = &cobra.Command{
	Use:   "npm",
	Short: "Export npm global packages",
	Long:  `Export globally installed npm packages to a file.`,
	Example: `  # Dry-run (default) - preview what will be exported
  goodbye export npm

  # Export to specific directory
  goodbye export npm --dir ~/npm-export

  # Actually export
  goodbye export npm --dir ~/npm-export --apply`,
	RunE: runExportNpm,
}

var exportMiseCmd = &cobra.Command{
	Use:   "mise",
	Short: "Export mise configuration",
	Long: `Export the current mise environment to a configuration file.

Exports all installed mise tools to either:
  - .mise.toml (default)
  - .tool-versions`,
	Example: `  # Dry-run (default) - preview what will be exported
  goodbye export mise

  # Export to specific directory
  goodbye export mise --dir ~/goodbye-export

  # Export as .tool-versions format
  goodbye export mise --format tool-versions --apply

  # Actually export
  goodbye export mise --dir ~/goodbye-export --apply`,
	RunE: runExportMise,
}

var (
	exportDir        string
	exportApply      bool
	exportVerbose    bool
	exportMiseFormat string
	exportNpmFormat  string
)

func init() {
	rootCmd.AddCommand(exportCmd)
	exportCmd.AddCommand(exportBrewCmd)
	exportCmd.AddCommand(exportNpmCmd)
	exportCmd.AddCommand(exportMiseCmd)

	exportBrewCmd.Flags().StringVar(&exportDir, "dir", ".", "Output directory for exported files")
	exportBrewCmd.Flags().BoolVar(&exportApply, "apply", false, "Actually perform the export (default is dry-run)")
	exportBrewCmd.Flags().BoolVarP(&exportVerbose, "verbose", "v", false, "Verbose output")

	exportNpmCmd.Flags().StringVar(&exportDir, "dir", ".", "Output directory for exported files")
	exportNpmCmd.Flags().BoolVar(&exportApply, "apply", false, "Actually perform the export (default is dry-run)")
	exportNpmCmd.Flags().BoolVarP(&exportVerbose, "verbose", "v", false, "Verbose output")
	exportNpmCmd.Flags().StringVar(&exportNpmFormat, "format", "text", "Output format (text or mise; mise uses npm JSON output and merges into .mise.toml)")

	exportMiseCmd.Flags().StringVar(&exportDir, "dir", ".", "Output directory for exported files")
	exportMiseCmd.Flags().BoolVar(&exportApply, "apply", false, "Actually perform the export (default is dry-run)")
	exportMiseCmd.Flags().BoolVarP(&exportVerbose, "verbose", "v", false, "Verbose output")
	exportMiseCmd.Flags().StringVar(&exportMiseFormat, "format", "toml", "Output format (toml or tool-versions)")
}

func runExportBrew(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to load config: %v\n", err)
		cfg = config.DefaultConfig()
	}

	opts := brew.ExportOptions{
		Dir:     exportDir,
		DryRun:  !exportApply,
		Verbose: exportVerbose,
	}

	return brew.Export(cfg, opts)
}

func runExportNpm(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to load config: %v\n", err)
		cfg = config.DefaultConfig()
	}

	npmCfg := &npm.NpmExportConfig{
		GlobalCmd: cfg.Npm.Export.GlobalCmd,
	}

	opts := npm.ExportOptions{
		Dir:     exportDir,
		DryRun:  !exportApply,
		Verbose: exportVerbose,
		Format:  exportNpmFormat,
		Out:     cmd.OutOrStdout(),
	}

	return npm.Export(npmCfg, opts)
}

func runExportMise(cmd *cobra.Command, args []string) error {
	opts := mise.ExportOptions{
		Dir:     exportDir,
		DryRun:  !exportApply,
		Verbose: exportVerbose,
		Format:  exportMiseFormat,
	}

	return mise.Export(opts)
}
