package miseconfig

import (
	"bytes"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/BurntSushi/toml"
)

func Test複数のエクスポートを既存設定と集約する(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".mise.toml")
	existing := "[env]\nKEEP = 'yes'\n[tools]\nnode = ['22', '24']\n"
	if err := os.WriteFile(path, []byte(existing), 0600); err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	for _, tools := range []map[string]string{
		{"npm:@scope/cli": "1.2.3"},
		{"go:example.com/tools/cmd/tool": "v1.0.0", "cargo:ripgrep": "14.1.1"},
		{"pypi:black": "25.1.0"},
	} {
		if err := Export(dir, tools, false, &out); err != nil {
			t.Fatal(err)
		}
	}
	var got struct {
		Env   map[string]string
		Tools map[string]any
	}
	if _, err := toml.DecodeFile(path, &got); err != nil {
		t.Fatal(err)
	}
	if got.Env["KEEP"] != "yes" || got.Tools["npm:@scope/cli"] != "1.2.3" || got.Tools["go:example.com/tools/cmd/tool"] != "v1.0.0" || got.Tools["cargo:ripgrep"] != "14.1.1" || got.Tools["pypi:black"] != "25.1.0" || !reflect.DeepEqual(got.Tools["node"], []any{"22", "24"}) {
		t.Fatalf("%+v", got)
	}
	info, err := os.Stat(path)
	if err != nil || info.Mode().Perm() != 0600 {
		t.Fatalf("permissions: %v %v", info, err)
	}
	before, _ := os.ReadFile(path)
	if err := Export(dir, map[string]string{"npm:@scope/cli": "1.2.3"}, false, &out); err != nil {
		t.Fatal(err)
	}
	after, _ := os.ReadFile(path)
	if !bytes.Equal(before, after) {
		t.Fatal("same version rewrote file")
	}
}

func Test競合や不正な設定なら既存ファイルを変更しない(t *testing.T) {
	for _, input := range []string{
		"# keep comment\n[tools]\n'npm:tool' = '2.0.0'\n",
		"[tools]\n'npm:tool' = ['1.0.0', '2.0.0']\n",
		"[tools]\n'npm:tool' = {version='1.0.0', postinstall='echo keep'}\n",
		"tools = 'invalid'\n",
		"[tools\n",
	} {
		t.Run(input, func(t *testing.T) {
			dir := t.TempDir()
			path := filepath.Join(dir, ".mise.toml")
			if err := os.WriteFile(path, []byte(input), 0600); err != nil {
				t.Fatal(err)
			}
			for _, dry := range []bool{true, false} {
				var out bytes.Buffer
				if err := Export(dir, map[string]string{"npm:tool": "1.0.0", "cargo:new": "1.0.0"}, dry, &out); err == nil {
					t.Fatal("expected conflict or parse error")
				}
				got, _ := os.ReadFile(path)
				if string(got) != input {
					t.Fatalf("changed file: %s", got)
				}
			}
		})
	}
}

func Testドライランでは集約内容を表示するだけ(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "missing")
	var out bytes.Buffer
	if err := Export(dir, map[string]string{"npm:@scope/tool": "1.2.3"}, true, &out); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(dir); !os.IsNotExist(err) {
		t.Fatal("dry-run created directory")
	}
	if !strings.Contains(out.String(), `"npm:@scope/tool" = "1.2.3"`) {
		t.Fatal(out.String())
	}
	if err := Export(dir, nil, false, &out); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, ".mise.toml")); err != nil {
		t.Fatal(err)
	}
}

func Test他の書き込み中は設定を上書きしない(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, ".mise.toml.lock"), nil, 0600); err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	if err := Export(dir, map[string]string{"npm:tool": "1.0.0"}, false, &out); err == nil {
		t.Fatal("expected lock error")
	}
	if _, err := os.Stat(filepath.Join(dir, ".mise.toml")); !os.IsNotExist(err) {
		t.Fatal("created output despite lock")
	}
}
