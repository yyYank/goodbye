package dotfiles

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFindBackups(t *testing.T) {
	dir := t.TempDir()

	// Create backup files
	os.WriteFile(filepath.Join(dir, ".zshrc.backup.20260101120000"), []byte("old1"), 0644)
	os.WriteFile(filepath.Join(dir, ".zshrc.backup.20260215071045"), []byte("old2"), 0644)
	os.WriteFile(filepath.Join(dir, ".zshrc.backup.20260110100000"), []byte("old3"), 0644)
	// Unrelated file
	os.WriteFile(filepath.Join(dir, ".bashrc.backup.20260101120000"), []byte("bash"), 0644)

	backups := FindBackups(dir, ".zshrc")
	if len(backups) != 3 {
		t.Fatalf("expected 3 backups, got %d", len(backups))
	}

	// Should be sorted by timestamp descending (latest first)
	if backups[0].Timestamp != "20260215071045" {
		t.Errorf("first backup timestamp = %v, want 20260215071045", backups[0].Timestamp)
	}
	if backups[1].Timestamp != "20260110100000" {
		t.Errorf("second backup timestamp = %v, want 20260110100000", backups[1].Timestamp)
	}
	if backups[2].Timestamp != "20260101120000" {
		t.Errorf("third backup timestamp = %v, want 20260101120000", backups[2].Timestamp)
	}
}

func TestFindBackups_NoBackups(t *testing.T) {
	dir := t.TempDir()

	backups := FindBackups(dir, ".zshrc")
	if len(backups) != 0 {
		t.Fatalf("expected 0 backups, got %d", len(backups))
	}
}

func TestSelectBackup_Latest(t *testing.T) {
	backups := []BackupInfo{
		{OriginalName: ".zshrc", BackupPath: "/tmp/.zshrc.backup.20260215071045", Timestamp: "20260215071045"},
		{OriginalName: ".zshrc", BackupPath: "/tmp/.zshrc.backup.20260110100000", Timestamp: "20260110100000"},
		{OriginalName: ".zshrc", BackupPath: "/tmp/.zshrc.backup.20260101120000", Timestamp: "20260101120000"},
	}

	selected, err := selectBackup(backups, "latest")
	if err != nil {
		t.Fatalf("selectBackup() error = %v", err)
	}
	if selected.Timestamp != "20260215071045" {
		t.Errorf("selected timestamp = %v, want 20260215071045", selected.Timestamp)
	}
}

func TestSelectBackup_SpecificTimestamp(t *testing.T) {
	backups := []BackupInfo{
		{OriginalName: ".zshrc", BackupPath: "/tmp/.zshrc.backup.20260215071045", Timestamp: "20260215071045"},
		{OriginalName: ".zshrc", BackupPath: "/tmp/.zshrc.backup.20260110100000", Timestamp: "20260110100000"},
	}

	selected, err := selectBackup(backups, "20260110100000")
	if err != nil {
		t.Fatalf("selectBackup() error = %v", err)
	}
	if selected.Timestamp != "20260110100000" {
		t.Errorf("selected timestamp = %v, want 20260110100000", selected.Timestamp)
	}
}

func TestSelectBackup_NotFound(t *testing.T) {
	backups := []BackupInfo{
		{OriginalName: ".zshrc", BackupPath: "/tmp/.zshrc.backup.20260215071045", Timestamp: "20260215071045"},
	}

	_, err := selectBackup(backups, "99999999999999")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestRecoverFile(t *testing.T) {
	homeDir := t.TempDir()

	// Create a symlink as the current file (simulating after import --apply)
	dst := filepath.Join(homeDir, ".zshrc")
	os.Symlink("/some/repo/.zshrc", dst)

	// Create a backup file
	backupContent := "original zshrc content"
	backupPath := filepath.Join(homeDir, ".zshrc.backup.20260215071045")
	os.WriteFile(backupPath, []byte(backupContent), 0644)

	// Recover
	err := recoverFile(backupPath, dst, false)
	if err != nil {
		t.Fatalf("recoverFile() error = %v", err)
	}

	// Verify symlink was replaced with the backup content
	info, err := os.Lstat(dst)
	if err != nil {
		t.Fatalf("failed to stat recovered file: %v", err)
	}
	if info.Mode()&os.ModeSymlink != 0 {
		t.Error("expected regular file, got symlink")
	}

	content, err := os.ReadFile(dst)
	if err != nil {
		t.Fatalf("failed to read recovered file: %v", err)
	}
	if string(content) != backupContent {
		t.Errorf("recovered content = %v, want %v", string(content), backupContent)
	}

	// Verify backup file was removed (renamed)
	if _, err := os.Stat(backupPath); !os.IsNotExist(err) {
		t.Error("expected backup file to be removed after recovery")
	}
}

func TestRecoverDirectory(t *testing.T) {
	homeDir := t.TempDir()

	// Create a symlink as the current directory
	dst := filepath.Join(homeDir, ".claude")
	os.Symlink("/some/repo/claude", dst)

	// Create a backup directory
	backupDir := filepath.Join(homeDir, ".claude.backup.20260215071045")
	os.MkdirAll(backupDir, 0755)
	os.WriteFile(filepath.Join(backupDir, "settings.json"), []byte(`{"old": true}`), 0644)

	// Recover
	err := recoverFile(backupDir, dst, false)
	if err != nil {
		t.Fatalf("recoverFile() error = %v", err)
	}

	// Verify symlink was replaced with directory
	info, err := os.Lstat(dst)
	if err != nil {
		t.Fatalf("failed to stat recovered dir: %v", err)
	}
	if info.Mode()&os.ModeSymlink != 0 {
		t.Error("expected directory, got symlink")
	}
	if !info.IsDir() {
		t.Error("expected directory")
	}

	// Verify content
	content, err := os.ReadFile(filepath.Join(dst, "settings.json"))
	if err != nil {
		t.Fatalf("failed to read recovered file: %v", err)
	}
	if string(content) != `{"old": true}` {
		t.Errorf("recovered content = %v, want {\"old\": true}", string(content))
	}
}

func TestRecoverDryRun(t *testing.T) {
	homeDir := t.TempDir()

	// Create a symlink and backup
	dst := filepath.Join(homeDir, ".zshrc")
	os.Symlink("/some/repo/.zshrc", dst)

	backupPath := filepath.Join(homeDir, ".zshrc.backup.20260215071045")
	os.WriteFile(backupPath, []byte("backup"), 0644)

	// FindBackups should find it
	backups := FindBackups(homeDir, ".zshrc")
	if len(backups) != 1 {
		t.Fatalf("expected 1 backup, got %d", len(backups))
	}

	// In dry-run, the symlink and backup should remain untouched
	info, err := os.Lstat(dst)
	if err != nil {
		t.Fatalf("failed to stat file: %v", err)
	}
	if info.Mode()&os.ModeSymlink == 0 {
		t.Error("expected symlink to still exist in dry-run")
	}

	if _, err := os.Stat(backupPath); os.IsNotExist(err) {
		t.Error("expected backup to still exist in dry-run")
	}
}
