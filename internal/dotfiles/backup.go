package dotfiles

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/yyYank/goodbye/internal/config"
)

// BackupOptions represents options for recovering dotfiles from backups
type BackupOptions struct {
	DryRun    bool
	Verbose   bool
	Timestamp string // "latest" or specific timestamp like "20260215071045"
	Continue  bool
}

// BackupInfo represents information about a backup file
type BackupInfo struct {
	OriginalName string
	BackupPath   string
	Timestamp    string
}

// Backup recovers dotfiles from backup files
func Backup(cfg *config.Config, opts BackupOptions) error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}

	if opts.Timestamp == "" {
		opts.Timestamp = "latest"
	}

	if opts.DryRun {
		fmt.Println("[dry-run] Would recover dotfiles from backups")
		fmt.Printf("  Timestamp: %s\n", opts.Timestamp)
		fmt.Println()
	}

	files := cfg.Dotfiles.Files
	var hasErrors bool

	// Recover files
	for _, file := range files {
		dst := filepath.Join(homeDir, file)
		backups := FindBackups(homeDir, file)

		if len(backups) == 0 {
			if opts.Verbose || opts.DryRun {
				fmt.Printf("  [skip] %s (no backup found)\n", file)
			}
			continue
		}

		backup, err := selectBackup(backups, opts.Timestamp)
		if err != nil {
			if opts.Verbose || opts.DryRun {
				fmt.Printf("  [skip] %s (%v)\n", file, err)
			}
			continue
		}

		if opts.DryRun {
			fmt.Printf("  [recover] %s ← %s\n", file, filepath.Base(backup.BackupPath))
			continue
		}

		if err := recoverFile(backup.BackupPath, dst, opts.Verbose); err != nil {
			hasErrors = true
			fmt.Printf("  [error] %s: %v\n", file, err)
			if !opts.Continue {
				return fmt.Errorf("failed to recover %s: %w", file, err)
			}
		} else {
			fmt.Printf("  [ok] %s (recovered from %s)\n", file, filepath.Base(backup.BackupPath))
		}
	}

	// Recover directories
	if len(cfg.Dotfiles.Directories) > 0 {
		if opts.DryRun || opts.Verbose {
			fmt.Println()
			fmt.Println("Directories:")
		}

		for _, dirMap := range cfg.Dotfiles.Directories {
			dst := expandTilde(filepath.Join(homeDir, dirMap.Target))
			backups := FindBackups(homeDir, dirMap.Target)

			if len(backups) == 0 {
				if opts.Verbose || opts.DryRun {
					fmt.Printf("  [skip] %s (no backup found)\n", dirMap.Target)
				}
				continue
			}

			backup, err := selectBackup(backups, opts.Timestamp)
			if err != nil {
				if opts.Verbose || opts.DryRun {
					fmt.Printf("  [skip] %s (%v)\n", dirMap.Target, err)
				}
				continue
			}

			if opts.DryRun {
				fmt.Printf("  [recover] %s ← %s\n", dirMap.Target, filepath.Base(backup.BackupPath))
				continue
			}

			if err := recoverFile(backup.BackupPath, dst, opts.Verbose); err != nil {
				hasErrors = true
				fmt.Printf("  [error] %s: %v\n", dirMap.Target, err)
				if !opts.Continue {
					return fmt.Errorf("failed to recover %s: %w", dirMap.Target, err)
				}
			} else {
				fmt.Printf("  [ok] %s (recovered from %s)\n", dirMap.Target, filepath.Base(backup.BackupPath))
			}
		}
	}

	if opts.DryRun {
		fmt.Println()
		fmt.Println("Run with --apply to actually recover the files.")
	} else {
		fmt.Println()
		if hasErrors {
			fmt.Println("Recovery completed with errors.")
		} else {
			fmt.Println("Recovery completed successfully.")
		}
	}

	return nil
}

// FindBackups searches for backup files matching the pattern <filename>.backup.<timestamp>
func FindBackups(dir, filename string) []BackupInfo {
	var backups []BackupInfo

	prefix := filename + ".backup."

	entries, err := os.ReadDir(dir)
	if err != nil {
		return backups
	}

	for _, entry := range entries {
		name := entry.Name()
		if strings.HasPrefix(name, prefix) {
			timestamp := strings.TrimPrefix(name, prefix)
			backups = append(backups, BackupInfo{
				OriginalName: filename,
				BackupPath:   filepath.Join(dir, name),
				Timestamp:    timestamp,
			})
		}
	}

	// Sort by timestamp descending (latest first)
	sort.Slice(backups, func(i, j int) bool {
		return backups[i].Timestamp > backups[j].Timestamp
	})

	return backups
}

// selectBackup selects a backup based on the timestamp option
func selectBackup(backups []BackupInfo, timestamp string) (*BackupInfo, error) {
	if len(backups) == 0 {
		return nil, fmt.Errorf("no backups found")
	}

	if timestamp == "latest" {
		return &backups[0], nil // already sorted descending
	}

	for i := range backups {
		if backups[i].Timestamp == timestamp {
			return &backups[i], nil
		}
	}

	return nil, fmt.Errorf("no backup found with timestamp %s", timestamp)
}

// recoverFile removes the current file/symlink and renames the backup to the original path
func recoverFile(backupPath, dst string, verbose bool) error {
	// Remove current file/symlink/directory if it exists
	if info, err := os.Lstat(dst); err == nil {
		if verbose {
			fmt.Printf("    Removing current %s\n", dst)
		}
		if info.IsDir() && info.Mode()&os.ModeSymlink == 0 {
			if err := os.RemoveAll(dst); err != nil {
				return fmt.Errorf("failed to remove current file: %w", err)
			}
		} else {
			if err := os.Remove(dst); err != nil {
				return fmt.Errorf("failed to remove current file: %w", err)
			}
		}
	}

	// Rename backup to original path
	if verbose {
		fmt.Printf("    Recovering %s → %s\n", backupPath, dst)
	}
	if err := os.Rename(backupPath, dst); err != nil {
		return fmt.Errorf("failed to recover backup: %w", err)
	}

	return nil
}
