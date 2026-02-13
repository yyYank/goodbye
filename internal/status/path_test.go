package status

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/yyYank/goodbye/internal/config"
)

func TestBuildPathSuggestion(t *testing.T) {
	tests := []struct {
		name        string
		line        string
		pattern     string
		replacement string
		want        string
	}{
		{
			name:        "avoids duplicated share segment",
			line:        "source /opt/homebrew/share/zsh-history-substring-search/zsh-history-substring-search.zsh",
			pattern:     "/opt/homebrew/",
			replacement: "$HOMEBREW_PREFIX/share/",
			want:        "source $HOMEBREW_PREFIX/share/zsh-history-substring-search/zsh-history-substring-search.zsh",
		},
		{
			name:        "normal replacement keeps expected output",
			line:        "source /usr/local/bin/tool-init.sh",
			pattern:     "/usr/local/bin/",
			replacement: "$HOMEBREW_PREFIX/bin/",
			want:        "source $HOMEBREW_PREFIX/bin/tool-init.sh",
		},
		{
			name:        "non-matching line unchanged",
			line:        "eval \"$(zoxide init zsh)\"",
			pattern:     "/opt/homebrew/",
			replacement: "$HOMEBREW_PREFIX/",
			want:        "eval \"$(zoxide init zsh)\"",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := buildPathSuggestion(tt.line, tt.pattern, tt.replacement)
			if got != tt.want {
				t.Fatalf("buildPathSuggestion() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestApplyPathFixValidatesSuggestedPath(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, ".zshrc")

	if err := os.WriteFile(tmpFile, []byte("source /opt/homebrew/share/foo.zsh\n"), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	issue := Issue{
		Type:       "path",
		File:       tmpFile,
		Line:       1,
		Current:    "source /opt/homebrew/share/foo.zsh",
		Suggestion: "source $HOMEBREW_PREFIX/share/share/foo.zsh",
	}

	err := applyPathFix(issue)
	if err == nil {
		t.Fatal("applyPathFix() should fail for non-existent replacement path")
	}
	if !strings.Contains(err.Error(), "replacement path does not exist") {
		t.Fatalf("unexpected error: %v", err)
	}

	content, readErr := os.ReadFile(tmpFile)
	if readErr != nil {
		t.Fatalf("failed to read test file: %v", readErr)
	}
	if string(content) != "source /opt/homebrew/share/foo.zsh\n" {
		t.Fatalf("file should be unchanged, got %q", string(content))
	}
}

func TestApplyPathFixSuccess(t *testing.T) {
	tmpDir := t.TempDir()
	homebrewPrefix := filepath.Join(tmpDir, "homebrew")
	if err := os.MkdirAll(filepath.Join(homebrewPrefix, "share"), 0755); err != nil {
		t.Fatalf("failed to create test directory: %v", err)
	}

	t.Setenv("HOMEBREW_PREFIX", homebrewPrefix)

	tmpFile := filepath.Join(tmpDir, ".zshrc")
	if err := os.WriteFile(tmpFile, []byte("source /opt/homebrew/share/foo.zsh\n"), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	issue := Issue{
		Type:       "path",
		File:       tmpFile,
		Line:       1,
		Current:    "source /opt/homebrew/share/foo.zsh",
		Suggestion: "source $HOMEBREW_PREFIX/share",
	}

	if err := applyPathFix(issue); err != nil {
		t.Fatalf("applyPathFix() error = %v", err)
	}

	content, err := os.ReadFile(tmpFile)
	if err != nil {
		t.Fatalf("failed to read test file: %v", err)
	}
	if string(content) != "source $HOMEBREW_PREFIX/share\n" {
		t.Fatalf("unexpected file content: %q", string(content))
	}
}

func TestCheckFileForPaths(t *testing.T) {
	tmpDir := t.TempDir()
	file := filepath.Join(tmpDir, ".zshrc")
	content := strings.Join([]string{
		"# comment line",
		"source /opt/homebrew/share/zsh-history-substring-search/zsh-history-substring-search.zsh",
		"source /usr/local/bin/tool-init.sh",
	}, "\n")
	if err := os.WriteFile(file, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}

	rules := []config.PathRule{
		{
			Pattern:     "/opt/homebrew/",
			Replacement: "$HOMEBREW_PREFIX/share/",
			Description: "apple silicon",
		},
		{
			Pattern:     "/usr/local/bin/",
			Replacement: "$HOMEBREW_PREFIX/bin/",
			Description: "intel",
		},
	}

	issues, err := checkFileForPaths(file, rules, Options{})
	if err != nil {
		t.Fatalf("checkFileForPaths() error = %v", err)
	}
	if len(issues) != 2 {
		t.Fatalf("checkFileForPaths() issues = %d, want 2", len(issues))
	}

	if issues[0].Line != 2 {
		t.Fatalf("unexpected first issue line: %d", issues[0].Line)
	}
	if strings.Contains(issues[0].Suggestion, "/share/share/") {
		t.Fatalf("unexpected duplicated segment in suggestion: %s", issues[0].Suggestion)
	}
}

func TestCheckPaths(t *testing.T) {
	tmpDir := t.TempDir()
	homeDir := filepath.Join(tmpDir, "home")
	dotfilesDir := filepath.Join(tmpDir, "dotfiles")
	if err := os.MkdirAll(homeDir, 0755); err != nil {
		t.Fatalf("failed to create home dir: %v", err)
	}
	if err := os.MkdirAll(dotfilesDir, 0755); err != nil {
		t.Fatalf("failed to create dotfiles dir: %v", err)
	}

	t.Setenv("HOME", homeDir)

	localFile := filepath.Join(dotfilesDir, ".zshrc")
	homeFile := filepath.Join(homeDir, ".zshrc")
	line := "source /opt/homebrew/share/zsh-history-substring-search/zsh-history-substring-search.zsh\n"
	if err := os.WriteFile(localFile, []byte(line), 0644); err != nil {
		t.Fatalf("failed to write local file: %v", err)
	}
	if err := os.WriteFile(homeFile, []byte(line), 0644); err != nil {
		t.Fatalf("failed to write home file: %v", err)
	}

	cfg := &config.Config{
		Dotfiles: config.DotfilesConfig{
			LocalPath: dotfilesDir,
			Files:     []string{".zshrc"},
		},
		Status: config.StatusConfig{
			PathRules: []config.PathRule{
				{
					Pattern:     "/opt/homebrew/",
					Replacement: "$HOMEBREW_PREFIX/share/",
				},
			},
		},
	}

	issues, err := CheckPaths(cfg, Options{})
	if err != nil {
		t.Fatalf("CheckPaths() error = %v", err)
	}
	if len(issues) != 2 {
		t.Fatalf("CheckPaths() issues = %d, want 2", len(issues))
	}
}

func TestValidateSuggestedPathWithoutPathToken(t *testing.T) {
	err := validateSuggestedPath("eval \"$(zoxide init zsh)\"")
	if err != nil {
		t.Fatalf("validateSuggestedPath() should skip line without path token, got %v", err)
	}
}
