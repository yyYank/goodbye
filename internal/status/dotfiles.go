package status

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/yyYank/goodbye/internal/config"
	"github.com/yyYank/goodbye/internal/dotfiles"
)

// CheckDotfiles checks the sync status of dotfiles
func CheckDotfiles(cfg *config.Config, opts Options) ([]Issue, error) {
	var issues []Issue

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}

	localPath := expandTilde(cfg.Dotfiles.LocalPath, homeDir)
	sourceDir := localPath
	if cfg.Dotfiles.SourceDir != "" {
		sourceDir = filepath.Join(localPath, cfg.Dotfiles.SourceDir)
	}

	// Check if repository exists
	if _, err := os.Stat(localPath); os.IsNotExist(err) {
		issues = append(issues, Issue{
			Type:        "dotfiles",
			File:        localPath,
			Description: "dotfiles repository not found",
			Suggestion:  fmt.Sprintf("Run 'goodbye import dotfiles --url <repo-url>' to clone the repository"),
		})
		return issues, nil
	}

	// Check git status for uncommitted changes
	gitIssues := checkGitStatus(localPath, opts)
	issues = append(issues, gitIssues...)

	// Check each dotfile
	for _, file := range cfg.Dotfiles.Files {
		srcPath := filepath.Join(sourceDir, file)
		dstPath := filepath.Join(homeDir, file)

		// Check if source file exists
		if _, err := os.Stat(srcPath); os.IsNotExist(err) {
			continue // Skip files not in repo
		}

		// Check destination file
		dstInfo, err := os.Lstat(dstPath)
		if os.IsNotExist(err) {
			issues = append(issues, Issue{
				Type:        "dotfiles",
				File:        dstPath,
				Description: "file not deployed to home directory",
				Suggestion:  "Run 'goodbye import dotfiles --apply' to deploy",
			})
			continue
		}
		if err != nil {
			if opts.Verbose {
				fmt.Printf("Warning: could not check %s: %v\n", dstPath, err)
			}
			continue
		}

		// Check if it's a symlink
		if dstInfo.Mode()&os.ModeSymlink != 0 {
			// Verify symlink target
			target, err := os.Readlink(dstPath)
			if err != nil {
				issues = append(issues, Issue{
					Type:        "dotfiles",
					File:        dstPath,
					Description: "failed to read symlink",
					Suggestion:  "Run 'goodbye import dotfiles --apply' to recreate",
				})
				continue
			}

			// Check if target exists
			if _, err := os.Stat(target); os.IsNotExist(err) {
				issues = append(issues, Issue{
					Type:        "dotfiles",
					File:        dstPath,
					Description: fmt.Sprintf("broken symlink (target missing: %s)", target),
					Current:     target,
					Suggestion:  "Run 'goodbye import dotfiles --apply' to recreate",
				})
				continue
			}

			// Check if target matches expected source
			absTarget, _ := filepath.Abs(target)
			absSrc, _ := filepath.Abs(srcPath)
			if absTarget != absSrc {
				issues = append(issues, Issue{
					Type:        "dotfiles",
					File:        dstPath,
					Description: fmt.Sprintf("symlink points to wrong target"),
					Current:     target,
					Suggestion:  fmt.Sprintf("Expected: %s. Run 'goodbye import dotfiles --apply' to fix", srcPath),
				})
			}
		} else {
			// It's a regular file, check if config expects symlink
			if cfg.Dotfiles.Symlink {
				issues = append(issues, Issue{
					Type:        "dotfiles",
					File:        dstPath,
					Description: "regular file instead of expected symlink",
					Suggestion:  "Run 'goodbye import dotfiles --apply' to convert to symlink",
				})
			}
		}
	}

	return issues, nil
}

// checkGitStatus checks for uncommitted changes in the dotfiles repository
func checkGitStatus(repoPath string, opts Options) []Issue {
	var issues []Issue

	// Check for uncommitted changes
	cmd := exec.Command("git", "-C", repoPath, "status", "--porcelain")
	output, err := cmd.Output()
	if err != nil {
		if opts.Verbose {
			fmt.Printf("Warning: could not check git status: %v\n", err)
		}
		return issues
	}

	if len(strings.TrimSpace(string(output))) > 0 {
		issues = append(issues, Issue{
			Type:        "dotfiles",
			File:        repoPath,
			Description: "uncommitted changes in dotfiles repository",
			Current:     strings.TrimSpace(string(output)),
			Suggestion:  "Commit your changes with 'git add . && git commit' in the dotfiles repository",
		})
	}

	// Check for unpushed commits
	cmd = exec.Command("git", "-C", repoPath, "status", "-sb")
	output, err = cmd.Output()
	if err == nil {
		statusLine := strings.Split(string(output), "\n")[0]
		if strings.Contains(statusLine, "ahead") {
			issues = append(issues, Issue{
				Type:        "dotfiles",
				File:        repoPath,
				Description: "unpushed commits in dotfiles repository",
				Suggestion:  "Push your changes with 'git push' in the dotfiles repository",
			})
		}
		if strings.Contains(statusLine, "behind") {
			issues = append(issues, Issue{
				Type:        "dotfiles",
				File:        repoPath,
				Description: "local repository is behind remote",
				Suggestion:  "Pull latest changes with 'goodbye import dotfiles --url <repo> --apply'",
			})
		}
	}

	return issues
}

// applyDotfilesFix applies a fix for a dotfiles issue
func applyDotfilesFix(cfg *config.Config, issue Issue) error {
	switch {
	case strings.Contains(issue.Description, "not found"):
		// Need to clone repository - user should run import with --url
		return fmt.Errorf("repository not found. Run 'goodbye import dotfiles --url <repo-url> --apply'")

	case strings.Contains(issue.Description, "not deployed"),
		strings.Contains(issue.Description, "broken symlink"),
		strings.Contains(issue.Description, "wrong target"),
		strings.Contains(issue.Description, "regular file instead"):
		// Re-import the dotfiles
		importOpts := dotfiles.ImportOptions{
			DryRun:   false,
			Verbose:  true,
			Symlink:  cfg.Dotfiles.Symlink,
			Backup:   cfg.Dotfiles.Backup,
			Files:    cfg.Dotfiles.Files,
			Continue: true,
		}
		return dotfiles.Import(cfg, importOpts)

	case strings.Contains(issue.Description, "uncommitted"):
		return fmt.Errorf("please commit your changes manually in %s", issue.File)

	case strings.Contains(issue.Description, "unpushed"):
		return fmt.Errorf("please push your changes manually in %s", issue.File)

	case strings.Contains(issue.Description, "behind"):
		// Pull latest changes
		syncOpts := dotfiles.SyncOptions{
			Repository: cfg.Dotfiles.Repository,
			LocalPath:  cfg.Dotfiles.LocalPath,
			DryRun:     false,
			Verbose:    true,
		}
		return dotfiles.Sync(cfg, syncOpts)

	default:
		return fmt.Errorf("unknown issue type, cannot auto-fix")
	}
}
