package brew

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestTruncateList(t *testing.T) {
	tests := []struct {
		name     string
		items    []string
		max      int
		expected string
	}{
		{
			name:     "empty list",
			items:    []string{},
			max:      5,
			expected: "",
		},
		{
			name:     "list shorter than max",
			items:    []string{"a", "b", "c"},
			max:      5,
			expected: "a, b, c",
		},
		{
			name:     "list equal to max",
			items:    []string{"a", "b", "c", "d", "e"},
			max:      5,
			expected: "a, b, c, d, e",
		},
		{
			name:     "list longer than max",
			items:    []string{"a", "b", "c", "d", "e", "f", "g"},
			max:      5,
			expected: "a, b, c, d, e, ...",
		},
		{
			name:     "single item",
			items:    []string{"only"},
			max:      5,
			expected: "only",
		},
		{
			name:     "max of 1",
			items:    []string{"a", "b", "c"},
			max:      1,
			expected: "a, ...",
		},
		{
			name:     "max of 0",
			items:    []string{"a", "b", "c"},
			max:      0,
			expected: ", ...", // strings.Join on empty slice returns "" + ", ..."
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := truncateList(tt.items, tt.max)
			if result != tt.expected {
				t.Errorf("truncateList() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestWriteAndReadLines(t *testing.T) {
	// Create a temporary directory for testing
	tmpDir, err := os.MkdirTemp("", "brew_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	tests := []struct {
		name  string
		lines []string
	}{
		{
			name:  "empty lines",
			lines: []string{},
		},
		{
			name:  "single line",
			lines: []string{"hello"},
		},
		{
			name:  "multiple lines",
			lines: []string{"line1", "line2", "line3"},
		},
		{
			name:  "lines with spaces",
			lines: []string{"hello world", "foo bar baz"},
		},
		{
			name:  "lines with special characters",
			lines: []string{"package@1.0.0", "another/package", "name-with-dash"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filePath := filepath.Join(tmpDir, tt.name+".txt")

			// Test writeLines
			err := writeLines(filePath, tt.lines)
			if err != nil {
				t.Fatalf("writeLines() error = %v", err)
			}

			// Test readLines
			result, err := readLines(filePath)
			if err != nil {
				t.Fatalf("readLines() error = %v", err)
			}

			// For empty input, readLines returns nil, not empty slice
			if len(tt.lines) == 0 && len(result) == 0 {
				return
			}

			if !reflect.DeepEqual(result, tt.lines) {
				t.Errorf("readLines() = %v, want %v", result, tt.lines)
			}
		})
	}
}

func TestReadLinesNonExistent(t *testing.T) {
	_, err := readLines("/nonexistent/path/file.txt")
	if err == nil {
		t.Error("readLines() should return error for non-existent file")
	}
}

func TestWriteLinesInvalidPath(t *testing.T) {
	err := writeLines("/nonexistent/directory/file.txt", []string{"test"})
	if err == nil {
		t.Error("writeLines() should return error for invalid path")
	}
}

func TestImportOptionsValidation(t *testing.T) {
	tests := []struct {
		name    string
		only    string
		wantErr bool
	}{
		{
			name:    "empty only (all)",
			only:    "",
			wantErr: false,
		},
		{
			name:    "only formula",
			only:    "formula",
			wantErr: false,
		},
		{
			name:    "only cask",
			only:    "cask",
			wantErr: false,
		},
		{
			name:    "only tap",
			only:    "tap",
			wantErr: false,
		},
		{
			name:    "invalid only value",
			only:    "invalid",
			wantErr: true,
		},
	}

	// Create a temporary directory with test files
	tmpDir, err := os.MkdirTemp("", "import_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create empty test files
	for _, filename := range []string{"formula.txt", "cask.txt", "tap.txt"} {
		if err := os.WriteFile(filepath.Join(tmpDir, filename), []byte{}, 0644); err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := ImportOptions{
				Dir:    tmpDir,
				DryRun: true,
				Only:   tt.only,
			}
			err := Import(opts)
			if (err != nil) != tt.wantErr {
				t.Errorf("Import() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestImportNonExistentDirectory(t *testing.T) {
	opts := ImportOptions{
		Dir:    "/nonexistent/directory",
		DryRun: true,
	}
	err := Import(opts)
	if err == nil {
		t.Error("Import() should return error for non-existent directory")
	}
}

func TestImportSkipTaps(t *testing.T) {
	// Create a temporary directory with test files
	tmpDir, err := os.MkdirTemp("", "import_skip_taps_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create test files
	if err := os.WriteFile(filepath.Join(tmpDir, "tap.txt"), []byte("homebrew/core\n"), 0644); err != nil {
		t.Fatalf("Failed to create tap.txt: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "formula.txt"), []byte{}, 0644); err != nil {
		t.Fatalf("Failed to create formula.txt: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "cask.txt"), []byte{}, 0644); err != nil {
		t.Fatalf("Failed to create cask.txt: %v", err)
	}

	opts := ImportOptions{
		Dir:      tmpDir,
		DryRun:   true,
		SkipTaps: true,
	}

	// This should not error - taps should be skipped
	err = Import(opts)
	if err != nil {
		t.Errorf("Import() with SkipTaps error = %v", err)
	}
}

func TestExportOptionsDefaults(t *testing.T) {
	opts := ExportOptions{}

	// Dir should default to empty, which gets set to "." in Export()
	if opts.Dir != "" {
		t.Errorf("ExportOptions.Dir default = %q, want %q", opts.Dir, "")
	}

	// DryRun should default to false
	if opts.DryRun != false {
		t.Error("ExportOptions.DryRun should default to false")
	}

	// Verbose should default to false
	if opts.Verbose != false {
		t.Error("ExportOptions.Verbose should default to false")
	}
}

func TestImportOptionsDefaults(t *testing.T) {
	opts := ImportOptions{}

	if opts.Dir != "" {
		t.Errorf("ImportOptions.Dir default = %q, want %q", opts.Dir, "")
	}

	if opts.DryRun != false {
		t.Error("ImportOptions.DryRun should default to false")
	}

	if opts.Verbose != false {
		t.Error("ImportOptions.Verbose should default to false")
	}

	if opts.Only != "" {
		t.Errorf("ImportOptions.Only default = %q, want %q", opts.Only, "")
	}

	if opts.SkipTaps != false {
		t.Error("ImportOptions.SkipTaps should default to false")
	}

	if opts.Continue != false {
		t.Error("ImportOptions.Continue should default to false")
	}
}

func TestImportFileWithComments(t *testing.T) {
	// Create a temporary directory
	tmpDir, err := os.MkdirTemp("", "import_comments_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a file with comments and empty lines
	content := `# This is a comment
package1

# Another comment
package2

package3`
	if err := os.WriteFile(filepath.Join(tmpDir, "formula.txt"), []byte(content), 0644); err != nil {
		t.Fatalf("Failed to create formula.txt: %v", err)
	}

	opts := ImportOptions{
		Dir:    tmpDir,
		DryRun: true,
		Only:   "formula",
	}

	// This should not error - comments and empty lines should be skipped
	err = Import(opts)
	if err != nil {
		t.Errorf("Import() with comments error = %v", err)
	}
}
