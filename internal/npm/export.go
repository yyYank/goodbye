package npm

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type NpmExportConfig struct {
	GlobalCmd string `toml:"global_cmd"`
}

type ExportOptions struct {
	Dir     string
	DryRun  bool
	Verbose bool
}

func DefaultExportConfig() *NpmExportConfig {
	return &NpmExportConfig{
		GlobalCmd: "npm list -g --depth=0 --parseable",
	}
}

func Export(cfg *NpmExportConfig, opts ExportOptions) error {
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

	if opts.DryRun {
		fmt.Println("[dry-run] Would create directory:", opts.Dir)
		fmt.Println("[dry-run] Would execute command:")
		fmt.Printf("  global: %s\n", cfg.GlobalCmd)
		fmt.Println("[dry-run] Would create file:")
		fmt.Printf("  %s/npm-global.txt\n", opts.Dir)

		fmt.Println("\n[dry-run] Preview of export content:")

		rawLines, err := runCommand(cfg.GlobalCmd)
		if err != nil {
			fmt.Printf("  global: (error: %v)\n", err)
		} else {
			packages := parseGlobalPackages(rawLines)
			fmt.Printf("  global (%d packages): %s\n", len(packages), truncateList(packages, 5))
		}

		return nil
	}

	if err := os.MkdirAll(opts.Dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", opts.Dir, err)
	}

	rawLines, err := runCommand(cfg.GlobalCmd)
	if err != nil {
		return fmt.Errorf("failed to get global packages: %w", err)
	}

	packages := parseGlobalPackages(rawLines)
	if err := writeLines(filepath.Join(opts.Dir, "npm-global.txt"), packages); err != nil {
		return fmt.Errorf("failed to write npm-global.txt: %w", err)
	}
	fmt.Printf("Exported %d global packages to %s/npm-global.txt\n", len(packages), opts.Dir)

	fmt.Println("\nExport completed successfully!")
	return nil
}

func parseGlobalPackages(lines []string) []string {
	var packages []string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		packages = append(packages, filepath.Base(line))
	}
	return packages
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
