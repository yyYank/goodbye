package cargo

import (
	"bytes"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/yyYank/goodbye/internal/transfer"
)

func Test一覧からcrateとバージョンを抽出する(t *testing.T) {
	input := "ripgrep v14.1.1:\n    rg\nlocal v0.1.0 (/tmp/local):\n    local\ngit-tool v1.0.0 (https://example.com/repo#abc):\n    git-tool\ncargo-edit v0.13.0:\n    cargo-add\n    cargo-rm\n"
	items, warnings, err := parseList([]byte(input))
	if err != nil || !reflect.DeepEqual(items, []string{"ripgrep@14.1.1", "cargo-edit@0.13.0"}) || len(warnings) != 2 {
		t.Fatalf("%v %v %v", items, warnings, err)
	}
	items, _, err = parseList(nil)
	if err != nil || len(items) != 0 {
		t.Fatalf("%v %v", items, err)
	}
	for _, input := range []string{"unexpected output", "tool vbanana:\n    tool", strings.Repeat("x", 70000)} {
		if _, _, err := parseList([]byte(input)); err == nil {
			t.Errorf("accepted malformed output")
		}
	}
}

func Test復元はcrateのバージョンを厳密固定する(t *testing.T) {
	args, err := installArgs("ripgrep@14.1.1")
	if err != nil || !reflect.DeepEqual(args, []string{"install", "ripgrep", "--version", "=14.1.1"}) {
		t.Fatalf("%v %v", args, err)
	}
	for _, spec := range []string{"ripgrep", "rg@latest", "rg@^14.0.0", "--path@1.0.0", "x@1.0.0 --git bad", "../x@1.0.0"} {
		if _, err := installArgs(spec); err == nil {
			t.Errorf("accepted %q", spec)
		}
	}
}

func Test一覧取得から復元までcrate名を渡す(t *testing.T) {
	var calls [][]string
	run := func(name string, args ...string) ([]byte, error) {
		calls = append(calls, append([]string{name}, args...))
		return []byte("ripgrep v14.1.1:\n    rg\n"), nil
	}
	m := newManager(run, run)
	var out bytes.Buffer
	opts := transfer.Options{Dir: t.TempDir(), Out: &out}
	if err := m.Export(opts); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(opts.Dir, "cargo-tools.txt"))
	if err != nil || string(data) != "ripgrep@14.1.1\n" {
		t.Fatalf("%q %v", data, err)
	}
	if err := m.Import(opts); err != nil {
		t.Fatal(err)
	}
	want := [][]string{{"cargo", "install", "--list", "--color", "never"}, {"cargo", "install", "ripgrep", "--version", "=14.1.1"}}
	if !reflect.DeepEqual(calls, want) {
		t.Fatal(calls)
	}
}
