package npm

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type NpmImportConfig struct {
	GlobalInstallCmd string `toml:"global_install_cmd"`
}

type ImportOptions struct {
	Dir      string
	DryRun   bool
	Verbose  bool
	Continue bool
}

func DefaultImportConfig() *NpmImportConfig {
	return &NpmImportConfig{
		GlobalInstallCmd: "npm install -g",
	}
}

func Import(cfg *NpmImportConfig, opts ImportOptions) error {
	if opts.Dir == "" {
		opts.Dir = "."
	}

	if strings.HasPrefix(opts.Dir, "~") {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("failed to get home directory: %w", err)
		}
		opts.Dir = filepath.Join(homeDir, opts.Dir[1:])
	}

	if _, err := os.Stat(opts.Dir); os.IsNotExist(err) {
		return fmt.Errorf("directory does not exist: %s", opts.Dir)
	}

	if opts.DryRun {
		fmt.Println("[dry-run] Would import from directory:", opts.Dir)
	}

	filePath := filepath.Join(opts.Dir, "npm-global.txt")

	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		if opts.Verbose {
			fmt.Println("Skipping npm-global.txt (file not found)")
		}
		return nil
	}

	lines, err := readLines(filePath)
	if err != nil {
		return fmt.Errorf("failed to read npm-global.txt: %w", err)
	}

	if len(lines) == 0 {
		if opts.Verbose {
			fmt.Println("Skipping npm-global.txt (empty)")
		}
		return nil
	}

	fmt.Printf("\nnpm-global.txt (%d packages):\n", len(lines))

	for _, item := range lines {
		item = strings.TrimSpace(item)
		if item == "" || strings.HasPrefix(item, "#") {
			continue
		}

		cmd := fmt.Sprintf("%s %s", cfg.GlobalInstallCmd, item)

		if opts.DryRun {
			fmt.Printf("  [dry-run] %s\n", cmd)
			continue
		}

		if opts.Verbose {
			fmt.Printf("  Running: %s\n", cmd)
		}

		if err := runCommandExec(cmd); err != nil {
			if opts.Continue {
				fmt.Printf("  Error installing %s: %v (continuing...)\n", item, err)
				continue
			}
			return fmt.Errorf("failed to run '%s': %w", cmd, err)
		}
		fmt.Printf("  Installed: %s\n", item)
	}

	if !opts.DryRun {
		fmt.Println("\nImport completed!")
	}
	return nil
}
