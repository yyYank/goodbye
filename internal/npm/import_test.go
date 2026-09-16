package npm

import (
	"os"
	"path/filepath"
	"testing"
)

func TestImportDryRun(t *testing.T) {
	// dry-runモードではインストールせずプレビューだけ表示する
	tmpDir, err := os.MkdirTemp("", "npm_import_dryrun")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	if err := writeLines(filepath.Join(tmpDir, "npm-global.txt"), []string{"express", "typescript"}); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	cfg := &NpmImportConfig{
		GlobalInstallCmd: "npm install -g",
	}

	opts := ImportOptions{
		Dir:    tmpDir,
		DryRun: true,
	}

	err = Import(cfg, opts)
	if err != nil {
		t.Errorf("Import() dry-run error = %v", err)
	}
}

func TestImportNonExistentDirectory(t *testing.T) {
	// 存在しないディレクトリはエラーになる
	cfg := &NpmImportConfig{
		GlobalInstallCmd: "npm install -g",
	}

	opts := ImportOptions{
		Dir:    "/nonexistent/directory",
		DryRun: true,
	}

	err := Import(cfg, opts)
	if err == nil {
		t.Error("Import() should return error for non-existent directory")
	}
}

func TestImportMissingFile(t *testing.T) {
	// npm-global.txtがない場合はスキップ（エラーにならない）
	tmpDir, err := os.MkdirTemp("", "npm_import_missing")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	cfg := &NpmImportConfig{
		GlobalInstallCmd: "npm install -g",
	}

	opts := ImportOptions{
		Dir:     tmpDir,
		DryRun:  true,
		Verbose: true,
	}

	err = Import(cfg, opts)
	if err != nil {
		t.Errorf("Import() should not error when file is missing: %v", err)
	}
}

func TestImportEmptyFile(t *testing.T) {
	// 空ファイルはスキップ（エラーにならない）
	tmpDir, err := os.MkdirTemp("", "npm_import_empty")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	if err := os.WriteFile(filepath.Join(tmpDir, "npm-global.txt"), []byte{}, 0644); err != nil {
		t.Fatalf("Failed to create empty file: %v", err)
	}

	cfg := &NpmImportConfig{
		GlobalInstallCmd: "npm install -g",
	}

	opts := ImportOptions{
		Dir:    tmpDir,
		DryRun: true,
	}

	err = Import(cfg, opts)
	if err != nil {
		t.Errorf("Import() should not error on empty file: %v", err)
	}
}

func TestImportWithComments(t *testing.T) {
	// コメント行と空行はスキップする
	tmpDir, err := os.MkdirTemp("", "npm_import_comments")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	content := "# This is a comment\nexpress\n\n# Another comment\ntypescript\n"
	if err := os.WriteFile(filepath.Join(tmpDir, "npm-global.txt"), []byte(content), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	cfg := &NpmImportConfig{
		GlobalInstallCmd: "npm install -g",
	}

	opts := ImportOptions{
		Dir:    tmpDir,
		DryRun: true,
	}

	err = Import(cfg, opts)
	if err != nil {
		t.Errorf("Import() with comments error = %v", err)
	}
}

func TestImportApplyWithContinue(t *testing.T) {
	// --continueモードではエラーがあっても続行する
	tmpDir, err := os.MkdirTemp("", "npm_import_continue")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	if err := writeLines(filepath.Join(tmpDir, "npm-global.txt"), []string{"fake-nonexistent-pkg-12345"}); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	cfg := &NpmImportConfig{
		GlobalInstallCmd: "false",
	}

	opts := ImportOptions{
		Dir:      tmpDir,
		DryRun:   false,
		Continue: true,
	}

	err = Import(cfg, opts)
	if err != nil {
		t.Errorf("Import() with --continue should not return error: %v", err)
	}
}

func TestImportApplyWithoutContinue(t *testing.T) {
	// --continueなしではエラーで停止する
	tmpDir, err := os.MkdirTemp("", "npm_import_nocontinue")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	if err := writeLines(filepath.Join(tmpDir, "npm-global.txt"), []string{"fake-nonexistent-pkg-12345"}); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	cfg := &NpmImportConfig{
		GlobalInstallCmd: "false",
	}

	opts := ImportOptions{
		Dir:      tmpDir,
		DryRun:   false,
		Continue: false,
	}

	err = Import(cfg, opts)
	if err == nil {
		t.Error("Import() without --continue should return error on failure")
	}
}
