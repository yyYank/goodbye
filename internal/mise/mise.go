package mise

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"

	"github.com/yyYank/goodbye/internal/config"
)

// MigrateOptions represents options for the brew --mise command
type MigrateOptions struct {
	DryRun  bool
	Verbose bool
}

// RegistryEntry represents an entry from mise registry
type RegistryEntry struct {
	Short   string   `json:"short"`
	Full    string   `json:"full"`
	Aliases []string `json:"aliases,omitempty"`
}

// MigrationCandidate represents a tool that can be migrated
type MigrationCandidate struct {
	BrewName      string
	NormalizedName string
	MiseName      string
}

// Migrate performs the brew to mise migration
func Migrate(cfg *config.Config, opts MigrateOptions) error {
	// Step 1: Get Homebrew formula list
	fmt.Println("Getting Homebrew formula list...")
	formulas, err := getBrewFormulas(cfg)
	if err != nil {
		return fmt.Errorf("failed to get brew formulas: %w", err)
	}
	fmt.Printf("Found %d formulas\n", len(formulas))

	// Step 2: Get mise registry
	fmt.Println("Getting mise registry...")
	registry, err := getMiseRegistry(cfg)
	if err != nil {
		return fmt.Errorf("failed to get mise registry: %w", err)
	}
	fmt.Printf("Found %d tools in mise registry\n", len(registry))

	// Step 3: Find migration candidates
	candidates := findCandidates(formulas, registry, cfg)
	if len(candidates) == 0 {
		fmt.Println("\nNo migration candidates found.")
		return nil
	}

	fmt.Printf("\nFound %d migration candidates:\n", len(candidates))
	fmt.Println(strings.Repeat("-", 60))
	fmt.Printf("%-25s %-20s %s\n", "BREW", "NORMALIZED", "MISE")
	fmt.Println(strings.Repeat("-", 60))
	for _, c := range candidates {
		fmt.Printf("%-25s %-20s %s\n", c.BrewName, c.NormalizedName, c.MiseName)
	}
	fmt.Println(strings.Repeat("-", 60))

	if opts.DryRun {
		fmt.Println("\n[dry-run] Would perform the following actions:")
		for _, c := range candidates {
			fmt.Printf("  1. mise install %s@latest\n", c.MiseName)
			fmt.Printf("  2. mise use -g %s@latest\n", c.MiseName)
			fmt.Printf("  3. Verify installation\n")
			fmt.Printf("  4. brew uninstall %s\n", c.BrewName)
			fmt.Println()
		}
		fmt.Println("\nTo apply these changes, run with --apply")
		return nil
	}

	// Step 4: Confirm
	fmt.Print("\nDo you want to proceed with migration? [y/N]: ")
	var response string
	fmt.Scanln(&response)
	if strings.ToLower(response) != "y" {
		fmt.Println("Migration cancelled.")
		return nil
	}

	// Step 5-8: Migrate each candidate
	var succeeded, failed []MigrationCandidate
	for _, c := range candidates {
		fmt.Printf("\nMigrating %s -> %s\n", c.BrewName, c.MiseName)

		// Install with mise
		fmt.Printf("  Installing %s with mise...\n", c.MiseName)
		installCmd := cfg.Mise.Commands.InstallCmd
		if installCmd == "" {
			installCmd = "mise install %s@latest"
		}
		if err := runCommand(fmt.Sprintf(installCmd, c.MiseName), opts.Verbose); err != nil {
			fmt.Printf("  Failed to install: %v\n", err)
			failed = append(failed, c)
			continue
		}

		// Set global
		fmt.Printf("  Setting %s as global...\n", c.MiseName)
		useGlobalCmd := cfg.Mise.Commands.UseGlobalCmd
		if useGlobalCmd == "" {
			useGlobalCmd = "mise use -g %s@latest"
		}
		if err := runCommand(fmt.Sprintf(useGlobalCmd, c.MiseName), opts.Verbose); err != nil {
			fmt.Printf("  Failed to set global: %v\n", err)
			failed = append(failed, c)
			continue
		}

		// Verify installation
		fmt.Printf("  Verifying installation...\n")
		if err := verifyInstallation(cfg, c.MiseName); err != nil {
			fmt.Printf("  Verification failed: %v\n", err)
			failed = append(failed, c)
			continue
		}

		// Uninstall from brew
		fmt.Printf("  Uninstalling %s from brew...\n", c.BrewName)
		brewUninstallCmd := cfg.Mise.Commands.BrewUninstallCmd
		if brewUninstallCmd == "" {
			brewUninstallCmd = "brew uninstall %s"
		}
		if err := runCommand(fmt.Sprintf(brewUninstallCmd, c.BrewName), opts.Verbose); err != nil {
			fmt.Printf("  Warning: Failed to uninstall from brew: %v\n", err)
			// Still consider it a success since mise is working
		}

		fmt.Printf("  Successfully migrated %s!\n", c.BrewName)
		succeeded = append(succeeded, c)
	}

	// Summary
	fmt.Println("\n" + strings.Repeat("=", 60))
	fmt.Println("Migration Summary")
	fmt.Println(strings.Repeat("=", 60))
	fmt.Printf("Succeeded: %d\n", len(succeeded))
	for _, c := range succeeded {
		fmt.Printf("  - %s -> %s\n", c.BrewName, c.MiseName)
	}
	if len(failed) > 0 {
		fmt.Printf("Failed: %d\n", len(failed))
		for _, c := range failed {
			fmt.Printf("  - %s\n", c.BrewName)
		}
	}

	return nil
}

func getBrewFormulas(cfg *config.Config) ([]string, error) {
	// Use command from config, fallback to default
	cmdStr := cfg.Brew.Export.FormulaCmd
	if cmdStr == "" {
		cmdStr = "brew list --installed-on-request"
	}

	cmd := exec.Command("sh", "-c", cmdStr)
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	var formulas []string
	scanner := bufio.NewScanner(strings.NewReader(string(output)))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" {
			formulas = append(formulas, line)
		}
	}
	return formulas, scanner.Err()
}

func getMiseRegistry(cfg *config.Config) (map[string]string, error) {
	// Use command from config, fallback to default
	cmdStr := cfg.Mise.Commands.RegistryCmd
	if cmdStr == "" {
		cmdStr = "mise registry"
	}

	cmd := exec.Command("sh", "-c", cmdStr)
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("mise command failed (is mise installed?): %w", err)
	}

	registry := make(map[string]string)
	scanner := bufio.NewScanner(strings.NewReader(string(output)))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		// Parse registry output (format: "name  backend:path")
		parts := strings.Fields(line)
		if len(parts) >= 1 {
			name := parts[0]
			registry[strings.ToLower(name)] = name
		}
	}
	return registry, scanner.Err()
}

// Attempt to get registry as JSON (newer mise versions)
func getMiseRegistryJSON(cfg *config.Config) ([]RegistryEntry, error) {
	// Use command from config, fallback to default
	cmdStr := cfg.Mise.Commands.RegistryJSONCmd
	if cmdStr == "" {
		cmdStr = "mise registry --json"
	}

	cmd := exec.Command("sh", "-c", cmdStr)
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	var entries []RegistryEntry
	if err := json.Unmarshal(output, &entries); err != nil {
		return nil, err
	}
	return entries, nil
}

func normalizeFormulaName(name string) string {
	// Remove version suffix (e.g., python@3.12 -> python)
	re := regexp.MustCompile(`@[\d.]+$`)
	normalized := re.ReplaceAllString(name, "")

	// Convert to lowercase
	normalized = strings.ToLower(normalized)

	return normalized
}

func findCandidates(formulas []string, registry map[string]string, cfg *config.Config) []MigrationCandidate {
	var candidates []MigrationCandidate

	// Use known mappings from config, with fallback to default
	knownMappings := cfg.Mise.KnownMappings
	if knownMappings == nil || len(knownMappings) == 0 {
		knownMappings = map[string]string{
			"node":       "node",
			"nodejs":     "node",
			"python":     "python",
			"python3":    "python",
			"ruby":       "ruby",
			"go":         "go",
			"golang":     "go",
			"rust":       "rust",
			"rustup":     "rust",
			"java":       "java",
			"openjdk":    "java",
			"deno":       "deno",
			"bun":        "bun",
			"terraform":  "terraform",
			"kubectl":    "kubectl",
			"helm":       "helm",
			"awscli":     "awscli",
			"yarn":       "yarn",
			"pnpm":       "pnpm",
			"gradle":     "gradle",
			"maven":      "maven",
			"kotlin":     "kotlin",
			"scala":      "scala",
			"elixir":     "elixir",
			"erlang":     "erlang",
			"lua":        "lua",
			"luajit":     "luajit",
			"perl":       "perl",
			"php":        "php",
			"zig":        "zig",
			"nim":        "nim",
			"crystal":    "crystal",
			"julia":      "julia",
			"r":          "r",
			"dotnet":     "dotnet",
			"flutter":    "flutter",
			"dart":       "dart",
		}
	}

	for _, formula := range formulas {
		normalized := normalizeFormulaName(formula)

		// Check known mappings first
		if miseName, ok := knownMappings[normalized]; ok {
			if _, exists := registry[miseName]; exists {
				candidates = append(candidates, MigrationCandidate{
					BrewName:       formula,
					NormalizedName: normalized,
					MiseName:       miseName,
				})
				continue
			}
		}

		// Check direct match in registry
		if miseName, exists := registry[normalized]; exists {
			candidates = append(candidates, MigrationCandidate{
				BrewName:       formula,
				NormalizedName: normalized,
				MiseName:       miseName,
			})
		}
	}

	return candidates
}

func runCommand(cmdStr string, verbose bool) error {
	cmd := exec.Command("sh", "-c", cmdStr)
	if verbose {
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
	}
	return cmd.Run()
}

func verifyInstallation(cfg *config.Config, miseName string) error {
	// Use command from config, fallback to default
	cmdTemplate := cfg.Mise.Commands.CurrentCmd
	if cmdTemplate == "" {
		cmdTemplate = "mise current"
	}

	// Build command with miseName argument
	cmdStr := fmt.Sprintf("%s %s", cmdTemplate, miseName)
	cmd := exec.Command("sh", "-c", cmdStr)
	output, err := cmd.Output()
	if err != nil {
		return err
	}
	if strings.TrimSpace(string(output)) == "" {
		return fmt.Errorf("no version installed")
	}
	return nil
}
