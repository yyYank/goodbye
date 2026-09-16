package npm

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestParseGlobalPackages(t *testing.T) {
	tests := []struct {
		name     string
		lines    []string
		expected []string
	}{
		{
			// parseable出力からパッケージ名を抽出する
			name:     "パス一覧からパッケージ名を抽出できる",
			lines:    []string{"/usr/local/lib/node_modules/npm", "/usr/local/lib/node_modules/typescript", "/usr/local/lib/node_modules/yarn"},
			expected: []string{"npm", "typescript", "yarn"},
		},
		{
			// 空行は無視する
			name:     "空のリストは空を返す",
			lines:    []string{},
			expected: nil,
		},
		{
			// 単一パッケージ
			name:     "単一パッケージを抽出できる",
			lines:    []string{"/opt/homebrew/lib/node_modules/npm"},
			expected: []string{"npm"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseGlobalPackages(tt.lines)
			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("parseGlobalPackages() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestWriteAndReadLines(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "npm_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	tests := []struct {
		name  string
		lines []string
	}{
		{
			name:  "空リスト",
			lines: []string{},
		},
		{
			name:  "単一行",
			lines: []string{"express"},
		},
		{
			name:  "複数行",
			lines: []string{"express", "typescript", "yarn"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filePath := filepath.Join(tmpDir, tt.name+".txt")

			err := writeLines(filePath, tt.lines)
			if err != nil {
				t.Fatalf("writeLines() error = %v", err)
			}

			result, err := readLines(filePath)
			if err != nil {
				t.Fatalf("readLines() error = %v", err)
			}

			if len(tt.lines) == 0 && len(result) == 0 {
				return
			}

			if !reflect.DeepEqual(result, tt.lines) {
				t.Errorf("readLines() = %v, want %v", result, tt.lines)
			}
		})
	}
}

func TestExportDryRun(t *testing.T) {
	// dry-runモードではファイルを作らずプレビューだけ表示する
	opts := ExportOptions{
		Dir:    "/tmp/npm-export-test-dryrun",
		DryRun: true,
	}

	cfg := &NpmExportConfig{
		GlobalCmd: "echo 'test-package'",
	}

	err := Export(cfg, opts)
	if err != nil {
		t.Errorf("Export() dry-run error = %v", err)
	}

	// dry-runなのでディレクトリは作られていないはず
	if _, err := os.Stat(opts.Dir); !os.IsNotExist(err) {
		t.Error("Export() dry-run should not create directory")
	}
}

func TestExportApply(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "npm_export_apply_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	exportDir := filepath.Join(tmpDir, "output")
	opts := ExportOptions{
		Dir:    exportDir,
		DryRun: false,
	}

	// echoでダミーのparseable出力を返すコマンドを使う
	cfg := &NpmExportConfig{
		GlobalCmd: "echo '/fake/node_modules/express\n/fake/node_modules/typescript'",
	}

	err = Export(cfg, opts)
	if err != nil {
		t.Fatalf("Export() apply error = %v", err)
	}

	// ファイルが作られているか確認
	content, err := readLines(filepath.Join(exportDir, "npm-global.txt"))
	if err != nil {
		t.Fatalf("Failed to read npm-global.txt: %v", err)
	}

	expected := []string{"express", "typescript"}
	if !reflect.DeepEqual(content, expected) {
		t.Errorf("npm-global.txt content = %v, want %v", content, expected)
	}
}
