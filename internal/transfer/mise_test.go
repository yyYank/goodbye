package transfer

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/BurntSushi/toml"
)

func Test各ツールをmiseバックエンド名で集約する(t *testing.T) {
	dir := t.TempDir()
	for _, tc := range []struct{ name, spec, key, version string }{
		{"pnpm", "@scope/cli@1.2.3", "npm:@scope/cli", "1.2.3"},
		{"npm", "typescript@5.7.2", "npm:typescript", "5.7.2"},
		{"go", "example.com/tools/cmd/tool@v1.2.3", "go:example.com/tools/cmd/tool", "v1.2.3"},
		{"cargo", "ripgrep@14.1.1", "cargo:ripgrep", "14.1.1"},
		{"uv", "black==1!25.1.0", "pypi:black", "1!25.1.0"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var calls []string
			m := testManager(&calls)
			m.Name = tc.name
			m.InstallArgs = func(string) ([]string, error) { return nil, nil }
			m.List = func() ([]string, []string, error) { return []string{tc.spec}, nil, nil }
			var out bytes.Buffer
			if err := m.Export(Options{Dir: dir, Format: "mise", Out: &out}); err != nil {
				t.Fatal(err)
			}
			var config struct{ Tools map[string]string }
			if _, err := toml.DecodeFile(filepath.Join(dir, ".mise.toml"), &config); err != nil {
				t.Fatal(err)
			}
			if config.Tools[tc.key] != tc.version {
				t.Fatalf("%v", config.Tools)
			}
			if len(calls) != 0 {
				t.Fatal("export executed installer")
			}
			if _, err := os.Stat(filepath.Join(dir, m.File)); !os.IsNotExist(err) {
				t.Fatal("mise export created text output")
			}
		})
	}
}

func Test同一取得結果のバージョン競合も拒否する(t *testing.T) {
	var calls []string
	m := testManager(&calls)
	m.Name = "pnpm"
	m.List = func() ([]string, []string, error) { return []string{"tool@1.0.0", "tool@2.0.0"}, nil, nil }
	var out bytes.Buffer
	dir := t.TempDir()
	if err := m.Export(Options{Dir: dir, Format: "mise", Out: &out}); err == nil || !strings.Contains(err.Error(), "conflict") {
		t.Fatalf("%v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, ".mise.toml")); !os.IsNotExist(err) {
		t.Fatal("conflict created config")
	}
}

func Test未知の形式では一覧を取得しない(t *testing.T) {
	var calls []string
	m := testManager(&calls)
	m.List = func() ([]string, []string, error) { t.Fatal("unexpected list"); return nil, nil, nil }
	if err := m.Export(Options{Format: "invalid", Dir: t.TempDir()}); err == nil {
		t.Fatal("expected format error")
	}
}

func Testドライランのmise形式はファイルを変更しない(t *testing.T) {
	var calls []string
	m := testManager(&calls)
	m.Name = "go"
	m.List = func() ([]string, []string, error) { return []string{"example.com/tool@v1.0.0"}, nil, nil }
	dir := filepath.Join(t.TempDir(), "absent")
	var out bytes.Buffer
	if err := m.Export(Options{Dir: dir, Format: "mise", DryRun: true, Out: &out}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), `"go:example.com/tool" = "v1.0.0"`) {
		t.Fatal(out.String())
	}
	if _, err := os.Stat(dir); !os.IsNotExist(err) {
		t.Fatal("dry-run wrote files")
	}
}
