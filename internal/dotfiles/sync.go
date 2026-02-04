package dotfiles

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/yyYank/goodbye/internal/config"
)

// SyncOptions represents options for syncing dotfiles repository
type SyncOptions struct {
	Repository string
	LocalPath  string
	DryRun     bool
	Verbose    bool
}

// Sync clones or updates the dotfiles repository and saves the config
func Sync(cfg *config.Config, opts SyncOptions) error {
	// Expand tilde in local path
	localPath := expandTilde(opts.LocalPath)

	if opts.DryRun {
		fmt.Println("[dry-run] Would sync dotfiles repository")
		fmt.Printf("  Repository: %s\n", opts.Repository)
		fmt.Printf("  Local path: %s\n", localPath)

		if _, err := os.Stat(localPath); os.IsNotExist(err) {
			fmt.Println("  Action: Clone repository")
		} else {
			fmt.Println("  Action: Pull latest changes")
		}
		fmt.Println()
		fmt.Println("[dry-run] Would update ~/.goodbye.toml with repository URL")
		return nil
	}

	// Clone or pull the repository
	if _, err := os.Stat(localPath); os.IsNotExist(err) {
		// Clone
		if opts.Verbose {
			fmt.Printf("Cloning %s to %s...\n", opts.Repository, localPath)
		}
		if err := gitClone(opts.Repository, localPath, opts.Verbose); err != nil {
			return fmt.Errorf("failed to clone repository: %w", err)
		}
		fmt.Printf("Successfully cloned dotfiles to %s\n", localPath)
	} else {
		// Pull
		if opts.Verbose {
			fmt.Printf("Pulling latest changes in %s...\n", localPath)
		}
		if err := gitPull(localPath, opts.Verbose); err != nil {
			return fmt.Errorf("failed to pull repository: %w", err)
		}
		fmt.Printf("Successfully updated dotfiles in %s\n", localPath)
	}

	// Update config
	cfg.Dotfiles.Repository = opts.Repository
	cfg.Dotfiles.LocalPath = opts.LocalPath

	if err := config.Save(cfg); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	fmt.Println("Configuration saved to ~/.goodbye.toml")
	return nil
}

// expandTilde expands ~ to user's home directory
func expandTilde(path string) string {
	if strings.HasPrefix(path, "~") {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return path
		}
		return filepath.Join(homeDir, path[1:])
	}
	return path
}

// gitClone clones a repository
func gitClone(repo, dest string, verbose bool) error {
	cmd := exec.Command("git", "clone", repo, dest)
	if verbose {
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
	}
	return cmd.Run()
}

// gitPull pulls the latest changes
func gitPull(dir string, verbose bool) error {
	cmd := exec.Command("git", "-C", dir, "pull")
	if verbose {
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
	}
	return cmd.Run()
}
