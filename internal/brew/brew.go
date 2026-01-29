package brew

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/yyYank/goodbye/internal/config"
)

// ExportOptions represents options for the export command
type ExportOptions struct {
	Dir     string
	DryRun  bool
	Verbose bool
}

// ImportOptions represents options for the import command
type ImportOptions struct {
	Dir      string
	DryRun   bool
	Verbose  bool
	Only     string // formula, cask, or tap
	SkipTaps bool
	Continue bool
}

// Export exports the current Homebrew environment to files
func Export(cfg *config.Config, opts ExportOptions) error {
	if opts.Dir == "" {
		opts.Dir = "."
	}

	// Expand ~ to home directory
	if strings.HasPrefix(opts.Dir, "~") {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("failed to get home directory: %w", err)
		}
		opts.Dir = filepath.Join(homeDir, opts.Dir[1:])
	}

	if opts.DryRun {
		fmt.Println("[dry-run] Would create directory:", opts.Dir)
		fmt.Println("[dry-run] Would execute commands:")
		fmt.Printf("  formula: %s\n", cfg.Brew.Export.FormulaCmd)
		fmt.Printf("  cask:    %s\n", cfg.Brew.Export.CaskCmd)
		fmt.Printf("  tap:     %s\n", cfg.Brew.Export.TapCmd)
		fmt.Println("[dry-run] Would create files:")
		fmt.Printf("  %s/formula.txt\n", opts.Dir)
		fmt.Printf("  %s/cask.txt\n", opts.Dir)
		fmt.Printf("  %s/tap.txt\n", opts.Dir)

		// Show what would be exported
		fmt.Println("\n[dry-run] Preview of export content:")

		formulas, err := runCommand(cfg.Brew.Export.FormulaCmd)
		if err != nil {
			fmt.Printf("  formula: (error: %v)\n", err)
		} else {
			fmt.Printf("  formula (%d items): %s\n", len(formulas), truncateList(formulas, 5))
		}

		casks, err := runCommand(cfg.Brew.Export.CaskCmd)
		if err != nil {
			fmt.Printf("  cask: (error: %v)\n", err)
		} else {
			fmt.Printf("  cask (%d items): %s\n", len(casks), truncateList(casks, 5))
		}

		taps, err := runCommand(cfg.Brew.Export.TapCmd)
		if err != nil {
			fmt.Printf("  tap: (error: %v)\n", err)
		} else {
			fmt.Printf("  tap (%d items): %s\n", len(taps), truncateList(taps, 5))
		}

		return nil
	}

	// Create directory
	if err := os.MkdirAll(opts.Dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", opts.Dir, err)
	}

	// Export formula
	formulas, err := runCommand(cfg.Brew.Export.FormulaCmd)
	if err != nil {
		return fmt.Errorf("failed to get formulas: %w", err)
	}
	if err := writeLines(filepath.Join(opts.Dir, "formula.txt"), formulas); err != nil {
		return fmt.Errorf("failed to write formula.txt: %w", err)
	}
	fmt.Printf("Exported %d formulas to %s/formula.txt\n", len(formulas), opts.Dir)

	// Export cask
	casks, err := runCommand(cfg.Brew.Export.CaskCmd)
	if err != nil {
		return fmt.Errorf("failed to get casks: %w", err)
	}
	if err := writeLines(filepath.Join(opts.Dir, "cask.txt"), casks); err != nil {
		return fmt.Errorf("failed to write cask.txt: %w", err)
	}
	fmt.Printf("Exported %d casks to %s/cask.txt\n", len(casks), opts.Dir)

	// Export tap
	taps, err := runCommand(cfg.Brew.Export.TapCmd)
	if err != nil {
		return fmt.Errorf("failed to get taps: %w", err)
	}
	if err := writeLines(filepath.Join(opts.Dir, "tap.txt"), taps); err != nil {
		return fmt.Errorf("failed to write tap.txt: %w", err)
	}
	fmt.Printf("Exported %d taps to %s/tap.txt\n", len(taps), opts.Dir)

	fmt.Println("\nExport completed successfully!")
	return nil
}

// Import imports a Homebrew environment from exported files
func Import(cfg *config.Config, opts ImportOptions) error {
	if opts.Dir == "" {
		opts.Dir = "."
	}

	// Expand ~ to home directory
	if strings.HasPrefix(opts.Dir, "~") {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("failed to get home directory: %w", err)
		}
		opts.Dir = filepath.Join(homeDir, opts.Dir[1:])
	}

	// Check if directory exists
	if _, err := os.Stat(opts.Dir); os.IsNotExist(err) {
		return fmt.Errorf("directory does not exist: %s", opts.Dir)
	}

	var importTaps, importFormulas, importCasks bool

	switch opts.Only {
	case "":
		importTaps = !opts.SkipTaps
		importFormulas = true
		importCasks = true
	case "tap":
		importTaps = true
	case "formula":
		importFormulas = true
	case "cask":
		importCasks = true
	default:
		return fmt.Errorf("invalid --only value: %s (must be formula, cask, or tap)", opts.Only)
	}

	if opts.DryRun {
		fmt.Println("[dry-run] Would import from directory:", opts.Dir)
	}

	// Import taps first
	if importTaps {
		tapCmd := cfg.Brew.Import.TapCmd
		if tapCmd == "" {
			tapCmd = "brew tap"
		}
		if err := importFile(opts.Dir, "tap.txt", tapCmd, opts); err != nil {
			if !opts.Continue {
				return err
			}
			fmt.Printf("Warning: %v\n", err)
		}
	}

	// Import formulas
	if importFormulas {
		// Use filename from config, fallback to default
		formulaFile := cfg.Brew.Import.FormulaFile
		if formulaFile == "" {
			formulaFile = "formula.txt"
		}
		formulaInstallCmd := cfg.Brew.Import.FormulaInstallCmd
		if formulaInstallCmd == "" {
			formulaInstallCmd = "brew install"
		}
		if err := importFile(opts.Dir, formulaFile, formulaInstallCmd, opts); err != nil {
			if !opts.Continue {
				return err
			}
			fmt.Printf("Warning: %v\n", err)
		}
	}

	// Import casks
	if importCasks {
		// Use filename from config, fallback to default
		caskFile := cfg.Brew.Import.CaskFile
		if caskFile == "" {
			caskFile = "cask.txt"
		}
		caskInstallCmd := cfg.Brew.Import.CaskInstallCmd
		if caskInstallCmd == "" {
			caskInstallCmd = "brew install --cask"
		}
		if err := importFile(opts.Dir, caskFile, caskInstallCmd, opts); err != nil {
			if !opts.Continue {
				return err
			}
			fmt.Printf("Warning: %v\n", err)
		}
	}

	if !opts.DryRun {
		fmt.Println("\nImport completed!")
	}
	return nil
}

func importFile(dir, filename, cmdPrefix string, opts ImportOptions) error {
	filePath := filepath.Join(dir, filename)

	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		if opts.Verbose {
			fmt.Printf("Skipping %s (file not found)\n", filename)
		}
		return nil
	}

	lines, err := readLines(filePath)
	if err != nil {
		return fmt.Errorf("failed to read %s: %w", filename, err)
	}

	if len(lines) == 0 {
		if opts.Verbose {
			fmt.Printf("Skipping %s (empty)\n", filename)
		}
		return nil
	}

	fmt.Printf("\n%s (%d items):\n", filename, len(lines))

	for _, item := range lines {
		item = strings.TrimSpace(item)
		if item == "" || strings.HasPrefix(item, "#") {
			continue
		}

		cmd := fmt.Sprintf("%s %s", cmdPrefix, item)

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

	return nil
}

func runCommand(cmdStr string) ([]string, error) {
	cmd := exec.Command("sh", "-c", cmdStr)
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	var lines []string
	scanner := bufio.NewScanner(strings.NewReader(string(output)))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" {
			lines = append(lines, line)
		}
	}
	return lines, scanner.Err()
}

func runCommandExec(cmdStr string) error {
	cmd := exec.Command("sh", "-c", cmdStr)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func writeLines(path string, lines []string) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	for _, line := range lines {
		if _, err := fmt.Fprintln(file, line); err != nil {
			return err
		}
	}
	return nil
}

func readLines(path string) ([]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var lines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	return lines, scanner.Err()
}

func truncateList(items []string, max int) string {
	if len(items) <= max {
		return strings.Join(items, ", ")
	}
	return strings.Join(items[:max], ", ") + ", ..."
}
