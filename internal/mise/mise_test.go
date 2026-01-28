package mise

import (
	"reflect"
	"testing"
)

func TestNormalizeFormulaName(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "simple name",
			input:    "node",
			expected: "node",
		},
		{
			name:     "uppercase name",
			input:    "Node",
			expected: "node",
		},
		{
			name:     "mixed case",
			input:    "NodeJS",
			expected: "nodejs",
		},
		{
			name:     "with version suffix",
			input:    "python@3.12",
			expected: "python",
		},
		{
			name:     "with complex version suffix",
			input:    "python@3.12.1",
			expected: "python",
		},
		{
			name:     "name with dash",
			input:    "aws-cli",
			expected: "aws-cli",
		},
		{
			name:     "name with dash and version",
			input:    "aws-cli@2.0",
			expected: "aws-cli",
		},
		{
			name:     "openjdk with version",
			input:    "openjdk@17",
			expected: "openjdk",
		},
		{
			name:     "go with version",
			input:    "go@1.21",
			expected: "go",
		},
		{
			name:     "already lowercase",
			input:    "terraform",
			expected: "terraform",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := normalizeFormulaName(tt.input)
			if result != tt.expected {
				t.Errorf("normalizeFormulaName(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestFindCandidates(t *testing.T) {
	// Mock registry with common tools
	registry := map[string]string{
		"node":      "node",
		"python":    "python",
		"go":        "go",
		"ruby":      "ruby",
		"rust":      "rust",
		"java":      "java",
		"deno":      "deno",
		"bun":       "bun",
		"terraform": "terraform",
		"kubectl":   "kubectl",
	}

	tests := []struct {
		name     string
		formulas []string
		expected []MigrationCandidate
	}{
		{
			name:     "empty formulas",
			formulas: []string{},
			expected: nil,
		},
		{
			name:     "no matching formulas",
			formulas: []string{"vim", "neovim", "tmux"},
			expected: nil,
		},
		{
			name:     "single matching formula",
			formulas: []string{"node"},
			expected: []MigrationCandidate{
				{BrewName: "node", NormalizedName: "node", MiseName: "node"},
			},
		},
		{
			name:     "multiple matching formulas",
			formulas: []string{"node", "python", "go"},
			expected: []MigrationCandidate{
				{BrewName: "node", NormalizedName: "node", MiseName: "node"},
				{BrewName: "python", NormalizedName: "python", MiseName: "python"},
				{BrewName: "go", NormalizedName: "go", MiseName: "go"},
			},
		},
		{
			name:     "formula with version suffix",
			formulas: []string{"python@3.12"},
			expected: []MigrationCandidate{
				{BrewName: "python@3.12", NormalizedName: "python", MiseName: "python"},
			},
		},
		{
			name:     "mixed matching and non-matching",
			formulas: []string{"vim", "node", "neovim", "python", "tmux"},
			expected: []MigrationCandidate{
				{BrewName: "node", NormalizedName: "node", MiseName: "node"},
				{BrewName: "python", NormalizedName: "python", MiseName: "python"},
			},
		},
		{
			name:     "known mapping - nodejs to node",
			formulas: []string{"nodejs"},
			expected: []MigrationCandidate{
				{BrewName: "nodejs", NormalizedName: "nodejs", MiseName: "node"},
			},
		},
		{
			name:     "known mapping - golang to go",
			formulas: []string{"golang"},
			expected: []MigrationCandidate{
				{BrewName: "golang", NormalizedName: "golang", MiseName: "go"},
			},
		},
		{
			name:     "known mapping - python3 to python",
			formulas: []string{"python3"},
			expected: []MigrationCandidate{
				{BrewName: "python3", NormalizedName: "python3", MiseName: "python"},
			},
		},
		{
			name:     "known mapping - rustup to rust",
			formulas: []string{"rustup"},
			expected: []MigrationCandidate{
				{BrewName: "rustup", NormalizedName: "rustup", MiseName: "rust"},
			},
		},
		{
			name:     "known mapping - openjdk to java",
			formulas: []string{"openjdk"},
			expected: []MigrationCandidate{
				{BrewName: "openjdk", NormalizedName: "openjdk", MiseName: "java"},
			},
		},
		{
			name:     "direct registry match",
			formulas: []string{"terraform"},
			expected: []MigrationCandidate{
				{BrewName: "terraform", NormalizedName: "terraform", MiseName: "terraform"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := findCandidates(tt.formulas, registry)
			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("findCandidates() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestFindCandidatesWithEmptyRegistry(t *testing.T) {
	registry := map[string]string{}
	formulas := []string{"node", "python", "go"}

	result := findCandidates(formulas, registry)
	if result != nil {
		t.Errorf("findCandidates() with empty registry = %v, want nil", result)
	}
}

func TestMigrationCandidateStruct(t *testing.T) {
	candidate := MigrationCandidate{
		BrewName:       "python@3.12",
		NormalizedName: "python",
		MiseName:       "python",
	}

	if candidate.BrewName != "python@3.12" {
		t.Errorf("BrewName = %q, want %q", candidate.BrewName, "python@3.12")
	}
	if candidate.NormalizedName != "python" {
		t.Errorf("NormalizedName = %q, want %q", candidate.NormalizedName, "python")
	}
	if candidate.MiseName != "python" {
		t.Errorf("MiseName = %q, want %q", candidate.MiseName, "python")
	}
}

func TestMigrateOptionsStruct(t *testing.T) {
	opts := MigrateOptions{
		DryRun:  true,
		Verbose: true,
	}

	if opts.DryRun != true {
		t.Error("DryRun should be true")
	}
	if opts.Verbose != true {
		t.Error("Verbose should be true")
	}
}

func TestMigrateOptionsDefaults(t *testing.T) {
	opts := MigrateOptions{}

	if opts.DryRun != false {
		t.Error("DryRun should default to false")
	}
	if opts.Verbose != false {
		t.Error("Verbose should default to false")
	}
}

func TestRegistryEntryStruct(t *testing.T) {
	entry := RegistryEntry{
		Short:   "node",
		Full:    "core:node",
		Aliases: []string{"nodejs", "node.js"},
	}

	if entry.Short != "node" {
		t.Errorf("Short = %q, want %q", entry.Short, "node")
	}
	if entry.Full != "core:node" {
		t.Errorf("Full = %q, want %q", entry.Full, "core:node")
	}
	if len(entry.Aliases) != 2 {
		t.Errorf("Aliases length = %d, want 2", len(entry.Aliases))
	}
}

func TestKnownMappings(t *testing.T) {
	// Test that known mappings cover common tools
	registry := map[string]string{
		"node":      "node",
		"python":    "python",
		"go":        "go",
		"ruby":      "ruby",
		"rust":      "rust",
		"java":      "java",
		"deno":      "deno",
		"bun":       "bun",
		"terraform": "terraform",
		"kubectl":   "kubectl",
		"helm":      "helm",
		"yarn":      "yarn",
		"pnpm":      "pnpm",
	}

	// These should all map correctly
	knownMappingTests := []struct {
		brewName string
		miseName string
	}{
		{"node", "node"},
		{"nodejs", "node"},
		{"python", "python"},
		{"python3", "python"},
		{"go", "go"},
		{"golang", "go"},
		{"ruby", "ruby"},
		{"rust", "rust"},
		{"rustup", "rust"},
		{"java", "java"},
		{"deno", "deno"},
		{"bun", "bun"},
		{"terraform", "terraform"},
		{"kubectl", "kubectl"},
		{"helm", "helm"},
		{"yarn", "yarn"},
		{"pnpm", "pnpm"},
	}

	for _, tt := range knownMappingTests {
		t.Run(tt.brewName+"->"+tt.miseName, func(t *testing.T) {
			candidates := findCandidates([]string{tt.brewName}, registry)
			if len(candidates) != 1 {
				t.Fatalf("Expected 1 candidate, got %d", len(candidates))
			}
			if candidates[0].MiseName != tt.miseName {
				t.Errorf("MiseName = %q, want %q", candidates[0].MiseName, tt.miseName)
			}
		})
	}
}

func TestNormalizeFormulaNameEdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "just version",
			input:    "@3.12",
			expected: "",
		},
		{
			name:     "multiple @ symbols",
			input:    "pkg@1.0@2.0",
			expected: "pkg@1.0", // Removes trailing version pattern @2.0
		},
		{
			name:     "version without dots",
			input:    "python@3",
			expected: "python",
		},
		{
			name:     "all uppercase",
			input:    "PYTHON@3.12",
			expected: "python",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := normalizeFormulaName(tt.input)
			if result != tt.expected {
				t.Errorf("normalizeFormulaName(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}
