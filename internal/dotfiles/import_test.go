package dotfiles

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/yyYank/goodbye/internal/config"
)

func TestImportDirectory_Symlink(t *testing.T) {
	// Create temp directories
	srcDir := t.TempDir()
	dstDir := t.TempDir()

	// Create source directory with files
	claudeDir := filepath.Join(srcDir, "claude")
	if err := os.MkdirAll(claudeDir, 0755); err != nil {
		t.Fatalf("failed to create source dir: %v", err)
	}

	// Create test files in source
	testFile := filepath.Join(claudeDir, "settings.json")
	if err := os.WriteFile(testFile, []byte(`{"key": "value"}`), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	// Create subdirectory with file
	subDir := filepath.Join(claudeDir, "subdir")
	if err := os.MkdirAll(subDir, 0755); err != nil {
		t.Fatalf("failed to create subdir: %v", err)
	}
	subFile := filepath.Join(subDir, "config.toml")
	if err := os.WriteFile(subFile, []byte(`[config]`), 0644); err != nil {
		t.Fatalf("failed to create subfile: %v", err)
	}

	// Test symlink creation
	dst := filepath.Join(dstDir, ".claude")
	err := importDirectory(claudeDir, dst, true, false, false)
	if err != nil {
		t.Fatalf("importDirectory() error = %v", err)
	}

	// Verify symlink was created
	info, err := os.Lstat(dst)
	if err != nil {
		t.Fatalf("failed to stat destination: %v", err)
	}
	if info.Mode()&os.ModeSymlink == 0 {
		t.Error("expected symlink, got regular file/directory")
	}

	// Verify symlink target
	target, err := os.Readlink(dst)
	if err != nil {
		t.Fatalf("failed to read symlink: %v", err)
	}
	if target != claudeDir {
		t.Errorf("symlink target = %v, want %v", target, claudeDir)
	}
}

func TestImportDirectory_Copy(t *testing.T) {
	// Create temp directories
	srcDir := t.TempDir()
	dstDir := t.TempDir()

	// Create source directory with files
	claudeDir := filepath.Join(srcDir, "claude")
	if err := os.MkdirAll(claudeDir, 0755); err != nil {
		t.Fatalf("failed to create source dir: %v", err)
	}

	// Create test files in source
	testContent := `{"key": "value"}`
	testFile := filepath.Join(claudeDir, "settings.json")
	if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	// Create subdirectory with file
	subDir := filepath.Join(claudeDir, "subdir")
	if err := os.MkdirAll(subDir, 0755); err != nil {
		t.Fatalf("failed to create subdir: %v", err)
	}
	subContent := `[config]`
	subFile := filepath.Join(subDir, "config.toml")
	if err := os.WriteFile(subFile, []byte(subContent), 0644); err != nil {
		t.Fatalf("failed to create subfile: %v", err)
	}

	// Test copy
	dst := filepath.Join(dstDir, ".claude")
	err := importDirectory(claudeDir, dst, false, false, false)
	if err != nil {
		t.Fatalf("importDirectory() error = %v", err)
	}

	// Verify directory was created (not symlink)
	info, err := os.Lstat(dst)
	if err != nil {
		t.Fatalf("failed to stat destination: %v", err)
	}
	if info.Mode()&os.ModeSymlink != 0 {
		t.Error("expected directory, got symlink")
	}
	if !info.IsDir() {
		t.Error("expected directory")
	}

	// Verify file contents
	dstFile := filepath.Join(dst, "settings.json")
	content, err := os.ReadFile(dstFile)
	if err != nil {
		t.Fatalf("failed to read copied file: %v", err)
	}
	if string(content) != testContent {
		t.Errorf("file content = %v, want %v", string(content), testContent)
	}

	// Verify subdirectory and file
	dstSubFile := filepath.Join(dst, "subdir", "config.toml")
	subFileContent, err := os.ReadFile(dstSubFile)
	if err != nil {
		t.Fatalf("failed to read copied subfile: %v", err)
	}
	if string(subFileContent) != subContent {
		t.Errorf("subfile content = %v, want %v", string(subFileContent), subContent)
	}
}

func TestImportDirectory_Backup(t *testing.T) {
	// Create temp directories
	srcDir := t.TempDir()
	dstDir := t.TempDir()

	// Create source directory
	claudeDir := filepath.Join(srcDir, "claude")
	if err := os.MkdirAll(claudeDir, 0755); err != nil {
		t.Fatalf("failed to create source dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(claudeDir, "new.txt"), []byte("new"), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	// Create existing destination directory
	dst := filepath.Join(dstDir, ".claude")
	if err := os.MkdirAll(dst, 0755); err != nil {
		t.Fatalf("failed to create existing dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dst, "old.txt"), []byte("old"), 0644); err != nil {
		t.Fatalf("failed to create existing file: %v", err)
	}

	// Test with backup enabled
	err := importDirectory(claudeDir, dst, true, true, false)
	if err != nil {
		t.Fatalf("importDirectory() error = %v", err)
	}

	// Verify backup was created
	entries, err := os.ReadDir(dstDir)
	if err != nil {
		t.Fatalf("failed to read dir: %v", err)
	}

	foundBackup := false
	for _, entry := range entries {
		if entry.Name() != ".claude" && filepath.HasPrefix(entry.Name(), ".claude.backup.") {
			foundBackup = true
			// Verify backup contains old file
			backupPath := filepath.Join(dstDir, entry.Name(), "old.txt")
			content, err := os.ReadFile(backupPath)
			if err != nil {
				t.Fatalf("failed to read backup file: %v", err)
			}
			if string(content) != "old" {
				t.Errorf("backup content = %v, want 'old'", string(content))
			}
			break
		}
	}
	if !foundBackup {
		t.Error("expected backup directory to be created")
	}
}

func TestImport_WithDirectories(t *testing.T) {
	// Create temp directories
	repoDir := t.TempDir()
	homeDir := t.TempDir()

	// Create source structure: repo/macOS/claude/
	claudeDir := filepath.Join(repoDir, "macOS", "claude")
	if err := os.MkdirAll(claudeDir, 0755); err != nil {
		t.Fatalf("failed to create claude dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(claudeDir, "settings.json"), []byte(`{}`), 0644); err != nil {
		t.Fatalf("failed to create settings file: %v", err)
	}

	// Create config
	cfg := &config.Config{
		Dotfiles: config.DotfilesConfig{
			LocalPath: repoDir,
			SourceDir: "",
			Files:     []string{},
			Directories: []config.DirectoryMap{
				{Source: "macOS/claude", Target: ".claude"},
			},
			Symlink: true,
			Backup:  false,
		},
	}

	// Mock home directory by modifying the target path
	// We need to test the actual Import function behavior
	// For this test, we'll verify the directory mapping logic

	// Verify DirectoryMap structure
	if len(cfg.Dotfiles.Directories) != 1 {
		t.Fatalf("expected 1 directory mapping, got %d", len(cfg.Dotfiles.Directories))
	}
	if cfg.Dotfiles.Directories[0].Source != "macOS/claude" {
		t.Errorf("Source = %v, want 'macOS/claude'", cfg.Dotfiles.Directories[0].Source)
	}
	if cfg.Dotfiles.Directories[0].Target != ".claude" {
		t.Errorf("Target = %v, want '.claude'", cfg.Dotfiles.Directories[0].Target)
	}

	// Test the actual directory import
	src := filepath.Join(repoDir, "macOS", "claude")
	dst := filepath.Join(homeDir, ".claude")
	err := importDirectory(src, dst, true, false, false)
	if err != nil {
		t.Fatalf("importDirectory() error = %v", err)
	}

	// Verify the symlink
	info, err := os.Lstat(dst)
	if err != nil {
		t.Fatalf("failed to stat destination: %v", err)
	}
	if info.Mode()&os.ModeSymlink == 0 {
		t.Error("expected symlink")
	}
}

func TestCopyDirectory(t *testing.T) {
	srcDir := t.TempDir()
	dstDir := t.TempDir()

	// Create nested structure
	if err := os.MkdirAll(filepath.Join(srcDir, "a", "b", "c"), 0755); err != nil {
		t.Fatalf("failed to create nested dirs: %v", err)
	}
	if err := os.WriteFile(filepath.Join(srcDir, "root.txt"), []byte("root"), 0644); err != nil {
		t.Fatalf("failed to create file: %v", err)
	}
	if err := os.WriteFile(filepath.Join(srcDir, "a", "a.txt"), []byte("a"), 0644); err != nil {
		t.Fatalf("failed to create file: %v", err)
	}
	if err := os.WriteFile(filepath.Join(srcDir, "a", "b", "b.txt"), []byte("b"), 0644); err != nil {
		t.Fatalf("failed to create file: %v", err)
	}
	if err := os.WriteFile(filepath.Join(srcDir, "a", "b", "c", "c.txt"), []byte("c"), 0644); err != nil {
		t.Fatalf("failed to create file: %v", err)
	}

	dst := filepath.Join(dstDir, "copy")
	err := copyDirectory(srcDir, dst)
	if err != nil {
		t.Fatalf("copyDirectory() error = %v", err)
	}

	// Verify all files were copied
	tests := []struct {
		path    string
		content string
	}{
		{"root.txt", "root"},
		{"a/a.txt", "a"},
		{"a/b/b.txt", "b"},
		{"a/b/c/c.txt", "c"},
	}

	for _, tt := range tests {
		content, err := os.ReadFile(filepath.Join(dst, tt.path))
		if err != nil {
			t.Errorf("failed to read %s: %v", tt.path, err)
			continue
		}
		if string(content) != tt.content {
			t.Errorf("%s content = %v, want %v", tt.path, string(content), tt.content)
		}
	}
}
