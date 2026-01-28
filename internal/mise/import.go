package mise

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// ImportOptions represents options for the mise import command
type ImportOptions struct {
	Dir      string
	File     string // specific file to import (e.g., .mise.toml or .tool-versions)
	DryRun   bool
	Verbose  bool
	Continue bool
	Global   bool // use -g flag when installing
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
		// Try .mise.toml first, then .tool-versions
		tomlPath := filepath.Join(opts.Dir, ".mise.toml")
		tvPath := filepath.Join(opts.Dir, ".tool-versions")

		if _, err := os.Stat(tomlPath); err == nil {
			filePath = tomlPath
			fileType = "toml"
		} else if _, err := os.Stat(tvPath); err == nil {
			filePath = tvPath
			fileType = "tool-versions"
		} else {
			return fmt.Errorf("no mise configuration file found in %s (looked for .mise.toml and .tool-versions)", opts.Dir)
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
