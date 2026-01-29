package mise

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/yyYank/goodbye/internal/config"
)

// ImportOptions represents options for the mise import command
type ImportOptions struct {
	Dir      string
	File     string // specific file to import (e.g., .mise.toml or .tool-versions)
	DryRun   bool
	Verbose  bool
	Continue bool
	Global   bool   // use -g flag when installing
	FromBrew bool   // import from brew export files (formula.txt)
	Version  string // version to install (default: "latest")
}

// Import imports mise tools from a configuration file
func Import(opts ImportOptions) error {
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

	// Set default version
	if opts.Version == "" {
		opts.Version = "latest"
	}

	// Check if we should import from brew export files
	if opts.FromBrew {
		return importFromBrew(opts)
	}

	// Find the configuration file
	var filePath string
	var fileType string

	if opts.File != "" {
		filePath = filepath.Join(opts.Dir, opts.File)
		if _, err := os.Stat(filePath); os.IsNotExist(err) {
			return fmt.Errorf("file does not exist: %s", filePath)
		}
		if strings.HasSuffix(opts.File, ".toml") {
			fileType = "toml"
		} else {
			fileType = "tool-versions"
		}
	} else {
		// Try .mise.toml first, then .tool-versions, then formula.txt (brew export)
		tomlPath := filepath.Join(opts.Dir, ".mise.toml")
		tvPath := filepath.Join(opts.Dir, ".tool-versions")
		formulaPath := filepath.Join(opts.Dir, "formula.txt")

		if _, err := os.Stat(tomlPath); err == nil {
			filePath = tomlPath
			fileType = "toml"
		} else if _, err := os.Stat(tvPath); err == nil {
			filePath = tvPath
			fileType = "tool-versions"
		} else if _, err := os.Stat(formulaPath); err == nil {
			// Auto-detect brew export files
			return importFromBrew(opts)
		} else {
			return fmt.Errorf("no mise configuration file found in %s (looked for .mise.toml, .tool-versions, and formula.txt)", opts.Dir)
		}
	}

	// Read and parse the file
	content, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read %s: %w", filePath, err)
	}

	var tools []InstalledTool
	switch fileType {
	case "toml":
		tools, err = ParseTOML(string(content))
	case "tool-versions":
		tools, err = ParseToolVersions(string(content))
	}

	if err != nil {
		return fmt.Errorf("failed to parse %s: %w", filePath, err)
	}

	if len(tools) == 0 {
		fmt.Println("No tools found in configuration file.")
		return nil
	}

	fmt.Printf("Found %d tools in %s\n", len(tools), filePath)

	if opts.DryRun {
		fmt.Println("\n[dry-run] Would install the following tools:")
		for _, tool := range tools {
			globalFlag := ""
			if opts.Global {
				globalFlag = " -g"
			}
			fmt.Printf("  mise install%s %s@%s\n", globalFlag, tool.Name, tool.Version)
		}
		return nil
	}

	// Install tools
	var succeeded, failed []InstalledTool
	for _, tool := range tools {
		fmt.Printf("\nInstalling %s@%s...\n", tool.Name, tool.Version)

		args := []string{"install"}
		if opts.Global {
			args = append(args, "-g")
		}
		args = append(args, fmt.Sprintf("%s@%s", tool.Name, tool.Version))

		cmd := exec.Command("mise", args...)
		if opts.Verbose {
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
		}

		if err := cmd.Run(); err != nil {
			fmt.Printf("  Failed to install %s@%s: %v\n", tool.Name, tool.Version, err)
			failed = append(failed, tool)
			if !opts.Continue {
				return fmt.Errorf("installation failed for %s@%s", tool.Name, tool.Version)
			}
			continue
		}

		fmt.Printf("  Successfully installed %s@%s\n", tool.Name, tool.Version)
		succeeded = append(succeeded, tool)

		// Set as global if requested
		if opts.Global {
			useCmd := exec.Command("mise", "use", "-g", fmt.Sprintf("%s@%s", tool.Name, tool.Version))
			if opts.Verbose {
				useCmd.Stdout = os.Stdout
				useCmd.Stderr = os.Stderr
			}
			if err := useCmd.Run(); err != nil {
				fmt.Printf("  Warning: Failed to set %s@%s as global: %v\n", tool.Name, tool.Version, err)
			}
		}
	}

	// Summary
	fmt.Println("\n" + strings.Repeat("=", 60))
	fmt.Println("Import Summary")
	fmt.Println(strings.Repeat("=", 60))
	fmt.Printf("Succeeded: %d\n", len(succeeded))
	for _, tool := range succeeded {
		fmt.Printf("  - %s@%s\n", tool.Name, tool.Version)
	}
	if len(failed) > 0 {
		fmt.Printf("Failed: %d\n", len(failed))
		for _, tool := range failed {
			fmt.Printf("  - %s@%s\n", tool.Name, tool.Version)
		}
	}

	fmt.Println("\nImport completed!")
	return nil
}

// ParseTOML parses a .mise.toml file and extracts tools
func ParseTOML(content string) ([]InstalledTool, error) {
	var tools []InstalledTool
	inToolsSection := false

	scanner := bufio.NewScanner(strings.NewReader(content))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Check for section headers
		if strings.HasPrefix(line, "[") {
			inToolsSection = strings.HasPrefix(line, "[tools]")
			continue
		}

		// Parse tool entries in [tools] section
		if inToolsSection {
			parts := strings.SplitN(line, "=", 2)
			if len(parts) == 2 {
				name := strings.TrimSpace(parts[0])
				version := strings.TrimSpace(parts[1])
				// Remove quotes
				version = strings.Trim(version, `"'`)

				// Handle array format: ["3.12", "3.11"]
				if strings.HasPrefix(version, "[") {
					version = strings.Trim(version, "[]")
					versions := strings.Split(version, ",")
					for _, v := range versions {
						v = strings.TrimSpace(v)
						v = strings.Trim(v, `"'`)
						if v != "" {
							tools = append(tools, InstalledTool{
								Name:    name,
								Version: v,
							})
						}
					}
				} else {
					tools = append(tools, InstalledTool{
						Name:    name,
						Version: version,
					})
				}
			}
		}
	}

	return tools, scanner.Err()
}

// ParseToolVersions parses a .tool-versions file
func ParseToolVersions(content string) ([]InstalledTool, error) {
	var tools []InstalledTool

	scanner := bufio.NewScanner(strings.NewReader(content))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Parse "tool version [version2 ...]"
		fields := strings.Fields(line)
		if len(fields) >= 2 {
			name := fields[0]
			for _, version := range fields[1:] {
				tools = append(tools, InstalledTool{
					Name:    name,
					Version: version,
				})
			}
		}
	}

	return tools, scanner.Err()
}

// importFromBrew imports mise tools from brew export files (formula.txt)
func importFromBrew(opts ImportOptions) error {
	formulaPath := filepath.Join(opts.Dir, "formula.txt")

	// Check if formula.txt exists
	if _, err := os.Stat(formulaPath); os.IsNotExist(err) {
		return fmt.Errorf("formula.txt not found in %s", opts.Dir)
	}

	// Read formula.txt
	fmt.Printf("Reading formula.txt from %s...\n", opts.Dir)
	formulas, err := readFormulaFile(formulaPath)
	if err != nil {
		return fmt.Errorf("failed to read formula.txt: %w", err)
	}
	fmt.Printf("Found %d formulas\n", len(formulas))

	if len(formulas) == 0 {
		fmt.Println("No formulas found in formula.txt.")
		return nil
	}

	// Load config for known mappings
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Get mise registry
	fmt.Println("\nGetting mise registry...")
	registry, err := getMiseRegistry(cfg)
	if err != nil {
		return fmt.Errorf("failed to get mise registry: %w", err)
	}
	fmt.Printf("Found %d tools in mise registry\n", len(registry))

	// Find migration candidates
	candidates := findCandidates(formulas, registry, cfg)
	if len(candidates) == 0 {
		fmt.Println("\nNo migration candidates found.")
		fmt.Println("None of the brew formulas match tools available in mise.")
		return nil
	}

	fmt.Printf("\nMigration candidates (%d tools):\n", len(candidates))
	fmt.Println(strings.Repeat("-", 60))
	fmt.Printf("%-25s %s\n", "BREW", "MISE")
	fmt.Println(strings.Repeat("-", 60))
	for _, c := range candidates {
		fmt.Printf("%-25s %s\n", c.BrewName, c.MiseName)
	}
	fmt.Println(strings.Repeat("-", 60))

	if opts.DryRun {
		fmt.Println("\n[dry-run] Would install the following tools:")
		for _, c := range candidates {
			globalFlag := ""
			if opts.Global {
				globalFlag = " -g"
			}
			fmt.Printf("  mise install%s %s@%s\n", globalFlag, c.MiseName, opts.Version)
		}
		fmt.Println("\nTo apply, run with --apply")
		return nil
	}

	// Install tools
	var succeeded, failed []MigrationCandidate
	for _, c := range candidates {
		fmt.Printf("\nInstalling %s@%s...\n", c.MiseName, opts.Version)

		args := []string{"install"}
		if opts.Global {
			args = append(args, "-g")
		}
		args = append(args, fmt.Sprintf("%s@%s", c.MiseName, opts.Version))

		cmd := exec.Command("mise", args...)
		if opts.Verbose {
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
		}

		if err := cmd.Run(); err != nil {
			fmt.Printf("  Failed to install %s@%s: %v\n", c.MiseName, opts.Version, err)
			failed = append(failed, c)
			if !opts.Continue {
				return fmt.Errorf("installation failed for %s@%s", c.MiseName, opts.Version)
			}
			continue
		}

		fmt.Printf("  Successfully installed %s@%s\n", c.MiseName, opts.Version)
		succeeded = append(succeeded, c)

		// Set as global if requested
		if opts.Global {
			useCmd := exec.Command("mise", "use", "-g", fmt.Sprintf("%s@%s", c.MiseName, opts.Version))
			if opts.Verbose {
				useCmd.Stdout = os.Stdout
				useCmd.Stderr = os.Stderr
			}
			if err := useCmd.Run(); err != nil {
				fmt.Printf("  Warning: Failed to set %s@%s as global: %v\n", c.MiseName, opts.Version, err)
			}
		}
	}

	// Summary
	fmt.Println("\n" + strings.Repeat("=", 60))
	fmt.Println("Import Summary")
	fmt.Println(strings.Repeat("=", 60))
	fmt.Printf("Succeeded: %d\n", len(succeeded))
	for _, c := range succeeded {
		fmt.Printf("  - %s (from brew: %s)\n", c.MiseName, c.BrewName)
	}
	if len(failed) > 0 {
		fmt.Printf("Failed: %d\n", len(failed))
		for _, c := range failed {
			fmt.Printf("  - %s\n", c.MiseName)
		}
	}

	fmt.Println("\nImport completed!")
	return nil
}

// readFormulaFile reads a formula.txt file and returns the list of formulas
func readFormulaFile(path string) ([]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var formulas []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		// Skip empty lines and comments
		if line != "" && !strings.HasPrefix(line, "#") {
			formulas = append(formulas, line)
		}
	}
	return formulas, scanner.Err()
}
