package uv

import (
	"bytes"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/yyYank/goodbye/internal/transfer"
)

func Test一覧からツールだけを取り出し追加設定は報告する(t *testing.T) {
	input := "ruff v0.9.1 [required: >=0.8]\n- ruff\nblack v25.1.0\n- black\n- blackd\nlocal v1.0 [required: @ file:///tmp/local]\n- local\ngit-tool v1.0 [required: @ git+https://example.com/repo]\n- git-tool\nhttpie v3.2.4 [extras: socks]\n- http\njupyter v1.1.1 [with: pandas>=2]\n- jupyter\n"
	items, warnings, err := parseList([]byte(input))
	if err != nil || !reflect.DeepEqual(items, []string{"ruff==0.9.1", "black==25.1.0"}) || len(warnings) != 4 {
		t.Fatalf("%v %v %v", items, warnings, err)
	}
	items, _, err = parseList(nil)
	if err != nil || len(items) != 0 {
		t.Fatalf("%v %v", items, err)
	}
	for _, input := range []string{"garbage", "ruff vlatest", strings.Repeat("x", 70000)} {
		if _, _, err := parseList([]byte(input)); err == nil {
			t.Error("accepted malformed output")
		}
	}
}

func Test復元は固定PEP440バージョンを渡す(t *testing.T) {
	for _, spec := range []string{"ruff==0.9.1", "tool==1!2.0rc1.post2.dev3+local.1", "tool==1.0", "tool==2"} {
		args, err := installArgs(spec)
		if err != nil || !reflect.DeepEqual(args, []string{"tool", "install", "--", spec}) {
			t.Fatalf("%v %v", args, err)
		}
	}
	for _, spec := range []string{"ruff", "ruff>=1", "ruff==1.*", "ruff==latest", "-x==1", "x==1;echo", "x @ file:///tmp/x", "x==1 --python evil"} {
		if _, err := installArgs(spec); err == nil {
			t.Errorf("accepted %q", spec)
		}
	}
}

func Test詳細一覧取得から復元までツール名を渡す(t *testing.T) {
	var calls [][]string
	run := func(name string, args ...string) ([]byte, error) {
		calls = append(calls, append([]string{name}, args...))
		return []byte("ruff v0.9.1\n- ruff\n"), nil
	}
	m := newManager(run, run)
	var out bytes.Buffer
	opts := transfer.Options{Dir: t.TempDir(), Out: &out}
	if err := m.Export(opts); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(opts.Dir, "uv-tools.txt"))
	if err != nil || string(data) != "ruff==0.9.1\n" {
		t.Fatalf("%q %v", data, err)
	}
	if err := m.Import(opts); err != nil {
		t.Fatal(err)
	}
	want := [][]string{{"uv", "tool", "list", "--show-version-specifiers", "--show-extras", "--show-with", "--color", "never", "--offline"}, {"uv", "tool", "install", "--", "ruff==0.9.1"}}
	if !reflect.DeepEqual(calls, want) {
		t.Fatal(calls)
	}
}
