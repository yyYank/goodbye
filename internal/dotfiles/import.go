package dotfiles

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/yyYank/goodbye/internal/config"
)

// ImportOptions represents options for importing dotfiles
type ImportOptions struct {
	DryRun   bool
	Verbose  bool
	Symlink  bool
	Backup   bool
	Files    []string
	Continue bool
}

// ImportResult represents the result of importing a single file
type ImportResult struct {
	File     string
	Action   string
	Success  bool
	Error    error
	Skipped  bool
	BackedUp string
}

// Import copies or symlinks dotfiles from the repository to home directory
func Import(cfg *config.Config, opts ImportOptions) error {
	localPath := expandTilde(cfg.Dotfiles.LocalPath)
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}

	// Check if local path exists
	if _, err := os.Stat(localPath); os.IsNotExist(err) {
		return fmt.Errorf("dotfiles repository not found at %s. Run 'goodbye sync <repo-url>' first", localPath)
	}

	// Calculate source directory (local_path + source_dir)
	sourceDir := localPath
	if cfg.Dotfiles.SourceDir != "" {
		sourceDir = filepath.Join(localPath, cfg.Dotfiles.SourceDir)
	}

	// Use files from options or config
	files := opts.Files
	if len(files) == 0 {
		files = cfg.Dotfiles.Files
	}

	// Use symlink setting from options (already set from config if not overridden)
	useSymlink := opts.Symlink
	useBackup := opts.Backup

	if opts.DryRun {
		fmt.Println("[dry-run] Would import dotfiles from", sourceDir)
		fmt.Printf("  Method: %s\n", methodName(useSymlink))
		fmt.Printf("  Backup: %v\n", useBackup)
		fmt.Println()
	}

	var results []ImportResult
	var hasErrors bool

	for _, file := range files {
		src := filepath.Join(sourceDir, file)
		dst := filepath.Join(homeDir, file)

		result := ImportResult{
			File: file,
		}

		// Check if source file exists
		if _, err := os.Stat(src); os.IsNotExist(err) {
			result.Skipped = true
			result.Action = "skip (not found in repo)"
			results = append(results, result)
			if opts.Verbose || opts.DryRun {
				fmt.Printf("  [skip] %s (not found in repository)\n", file)
			}
			continue
		}

		if opts.DryRun {
			// Check destination status
			if info, err := os.Lstat(dst); err == nil {
				if info.Mode()&os.ModeSymlink != 0 {
					result.Action = fmt.Sprintf("replace symlink → %s", methodName(useSymlink))
				} else {
					if useBackup {
						result.Action = fmt.Sprintf("backup & %s", methodName(useSymlink))
					} else {
						result.Action = fmt.Sprintf("overwrite → %s", methodName(useSymlink))
					}
				}
			} else {
				result.Action = methodName(useSymlink)
			}
			fmt.Printf("  [%s] %s\n", result.Action, file)
			results = append(results, result)
			continue
		}

		// Actual import
		err := importFile(src, dst, useSymlink, useBackup, opts.Verbose)
		if err != nil {
			result.Success = false
			result.Error = err
			hasErrors = true
			fmt.Printf("  [error] %s: %v\n", file, err)
			if !opts.Continue {
				return fmt.Errorf("failed to import %s: %w", file, err)
			}
		} else {
			result.Success = true
			result.Action = methodName(useSymlink)
			fmt.Printf("  [ok] %s (%s)\n", file, result.Action)
		}
		results = append(results, result)
	}

	if opts.DryRun {
		fmt.Println()
		fmt.Println("Run with --apply to actually import the files.")
	} else {
		fmt.Println()
		if hasErrors {
			fmt.Println("Import completed with errors.")
		} else {
			fmt.Println("Import completed successfully.")
		}
	}

	return nil
}

func methodName(symlink bool) string {
	if symlink {
		return "symlink"
	}
	return "copy"
}

func importFile(src, dst string, useSymlink, useBackup bool, verbose bool) error {
	// Check if destination exists
	if info, err := os.Lstat(dst); err == nil {
		// Destination exists
		isSymlink := info.Mode()&os.ModeSymlink != 0

		if useBackup && !isSymlink {
			// Backup existing file
			backupPath := fmt.Sprintf("%s.backup.%s", dst, time.Now().Format("20060102150405"))
			if verbose {
				fmt.Printf("    Backing up %s to %s\n", dst, backupPath)
			}
			if err := os.Rename(dst, backupPath); err != nil {
				return fmt.Errorf("failed to backup: %w", err)
			}
		} else {
			// Remove existing file/symlink
			if err := os.Remove(dst); err != nil {
				return fmt.Errorf("failed to remove existing file: %w", err)
			}
		}
	}

	// Ensure parent directory exists
	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return fmt.Errorf("failed to create parent directory: %w", err)
	}

	if useSymlink {
		// Create symlink
		if verbose {
			fmt.Printf("    Creating symlink: %s → %s\n", dst, src)
		}
		return os.Symlink(src, dst)
	}

	// Copy file
	if verbose {
		fmt.Printf("    Copying: %s → %s\n", src, dst)
	}
	return copyFile(src, dst)
}

func copyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	sourceInfo, err := sourceFile.Stat()
	if err != nil {
		return err
	}

	destFile, err := os.OpenFile(dst, os.O_RDWR|os.O_CREATE|os.O_TRUNC, sourceInfo.Mode())
	if err != nil {
		return err
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, sourceFile)
	return err
}
