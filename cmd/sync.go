package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/yyYank/goodbye/internal/config"
	"github.com/yyYank/goodbye/internal/dotfiles"
)

var syncCmd = &cobra.Command{
	Use:   "sync <repository-url>",
	Short: "Sync dotfiles repository",
	Long: `Clone or update a dotfiles repository and save the configuration.

This command clones the specified dotfiles repository to the local path
(default: ~/.dotfiles) and saves the repository URL to ~/.goodbye.toml.

If the repository already exists locally, it will pull the latest changes.`,
	Example: `  # Dry-run (default) - preview what will happen
  goodbye sync https://github.com/username/dotfiles

  # Actually sync the repository
  goodbye sync https://github.com/username/dotfiles --apply

  # Specify custom local path
  goodbye sync https://github.com/username/dotfiles --path ~/my-dotfiles --apply`,
	Args: cobra.ExactArgs(1),
	RunE: runSync,
}

var (
	syncLocalPath string
	syncApply     bool
	syncVerbose   bool
)

func init() {
	rootCmd.AddCommand(syncCmd)

	syncCmd.Flags().StringVar(&syncLocalPath, "path", "", "Local path to clone/store dotfiles (default: ~/.dotfiles)")
	syncCmd.Flags().BoolVar(&syncApply, "apply", false, "Actually perform the sync (default is dry-run)")
	syncCmd.Flags().BoolVarP(&syncVerbose, "verbose", "v", false, "Verbose output")
}

func runSync(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	repoURL := args[0]
	localPath := syncLocalPath
	if localPath == "" {
		localPath = cfg.Dotfiles.LocalPath
	}

	opts := dotfiles.SyncOptions{
		Repository: repoURL,
		LocalPath:  localPath,
		DryRun:     !syncApply,
		Verbose:    syncVerbose,
	}

	return dotfiles.Sync(cfg, opts)
}
