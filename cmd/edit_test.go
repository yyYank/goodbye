package cmd

import (
	"os"
	"path/filepath"
	"testing"
)

func TestGetEditor(t *testing.T) {
	tests := []struct {
		name       string
		flagValue  string
		envValue   string
		wantEditor string
	}{
		{
			name:       "flag takes precedence over env",
			flagValue:  "emacs",
			envValue:   "nano",
			wantEditor: "emacs",
		},
		{
			name:       "use env when flag is empty",
			flagValue:  "",
			envValue:   "nano",
			wantEditor: "nano",
		},
		{
			name:       "fallback to vim when both empty",
			flagValue:  "",
			envValue:   "",
			wantEditor: "vim",
		},
		{
			name:       "flag with vim",
			flagValue:  "vim",
			envValue:   "",
			wantEditor: "vim",
		},
		{
			name:       "flag with code",
			flagValue:  "code",
			envValue:   "",
			wantEditor: "code",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Save and restore original EDITOR env
			originalEditor := os.Getenv("EDITOR")
			defer os.Setenv("EDITOR", originalEditor)

			os.Setenv("EDITOR", tt.envValue)

			result := getEditor(tt.flagValue)
			if result != tt.wantEditor {
				t.Errorf("getEditor() = %q, want %q", result, tt.wantEditor)
			}
		})
	}
}

func TestGetConfigPath(t *testing.T) {
	path, err := getConfigPath()
	if err != nil {
		t.Fatalf("getConfigPath() error = %v", err)
	}

	homeDir, _ := os.UserHomeDir()
	expected := filepath.Join(homeDir, ".goodbye.toml")

	if path != expected {
		t.Errorf("getConfigPath() = %q, want %q", path, expected)
	}
}

func TestEnsureConfigFileExists(t *testing.T) {
	// Create a temporary directory for testing
	tmpDir, err := os.MkdirTemp("", "edit_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	tests := []struct {
		name       string
		fileExists bool
	}{
		{
			name:       "creates file when it does not exist",
			fileExists: false,
		},
		{
			name:       "does nothing when file exists",
			fileExists: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			configPath := filepath.Join(tmpDir, tt.name+".toml")

			if tt.fileExists {
				// Create the file first
				if err := os.WriteFile(configPath, []byte("existing content"), 0644); err != nil {
					t.Fatalf("Failed to create test file: %v", err)
				}
			}

			created, err := ensureConfigFileExists(configPath)
			if err != nil {
				t.Fatalf("ensureConfigFileExists() error = %v", err)
			}

			// Verify file exists after the call
			if _, err := os.Stat(configPath); os.IsNotExist(err) {
				t.Error("ensureConfigFileExists() did not create the file")
			}

			// Verify return value
			if tt.fileExists && created {
				t.Error("ensureConfigFileExists() returned created=true for existing file")
			}
			if !tt.fileExists && !created {
				t.Error("ensureConfigFileExists() returned created=false for new file")
			}

			// If file existed before, verify content is preserved
			if tt.fileExists {
				content, err := os.ReadFile(configPath)
				if err != nil {
					t.Fatalf("Failed to read file: %v", err)
				}
				if string(content) != "existing content" {
					t.Error("ensureConfigFileExists() modified existing file content")
				}
			}
		})
	}
}

func TestEnsureConfigFileExistsInvalidPath(t *testing.T) {
	_, err := ensureConfigFileExists("/nonexistent/directory/file.toml")
	if err == nil {
		t.Error("ensureConfigFileExists() should return error for invalid path")
	}
}
