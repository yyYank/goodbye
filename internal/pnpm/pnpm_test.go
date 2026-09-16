package pnpm

import (
	"bytes"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/yyYank/goodbye/internal/transfer"
)

func Test一覧取得から復元まで固定バージョンを渡す(t *testing.T) {
	var calls [][]string
	run := func(name string, args ...string) ([]byte, error) {
		calls = append(calls, append([]string{name}, args...))
		if args[0] == "list" {
			return []byte(`[{"dependencies":{"@scope/tool":{"version":"1.2.3"}}}]`), nil
		}
		return nil, nil
	}
	m := newManager(run, run)
	var out bytes.Buffer
	opts := transfer.Options{Dir: t.TempDir(), Out: &out}
	if err := m.Export(opts); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(opts.Dir, "pnpm-global.txt"))
	if err != nil || string(data) != "@scope/tool@1.2.3\n" {
		t.Fatalf("%s %v", data, err)
	}
	if err := m.Import(opts); err != nil {
		t.Fatal(err)
	}
	want := [][]string{{"pnpm", "list", "-g", "--depth=0", "--json"}, {"pnpm", "add", "-g", "--", "@scope/tool@1.2.3"}}
	if !reflect.DeepEqual(calls, want) {
		t.Fatal(calls)
	}
}

func Test一覧からスコープと固定バージョンを保持する(t *testing.T) {
	items, warnings, err := parseList([]byte(`[{"dependencies":{"typescript":{"version":"5.7.2"},"@scope/tool":{"version":"1.2.3"},"local":{"version":"link:../local"}}}]`))
	if err != nil || !reflect.DeepEqual(items, []string{"@scope/tool@1.2.3", "typescript@5.7.2"}) || len(warnings) != 1 {
		t.Fatalf("items=%v warnings=%v err=%v", items, warnings, err)
	}
}

func Test任意依存も保持しエイリアスを誤復元しない(t *testing.T) {
	input := `[{"dependencies":{"alias":{"from":"actual-package","version":"1.2.3"},"normal":{"from":"normal","version":"2.0.0"}},"optionalDependencies":{"optional":{"version":"3.0.0"}},"devDependencies":{"dev":{"version":"4.0.0"}}}]`
	items, warnings, err := parseList([]byte(input))
	if err != nil || !reflect.DeepEqual(items, []string{"dev@4.0.0", "normal@2.0.0", "optional@3.0.0"}) || len(warnings) != 1 {
		t.Fatalf("%v %v %v", items, warnings, err)
	}
}

func Test一覧の空と不正JSONを区別する(t *testing.T) {
	for _, input := range []string{`[]`, `[{}]`} {
		items, _, err := parseList([]byte(input))
		if err != nil || len(items) != 0 {
			t.Fatalf("%s: %v %v", input, items, err)
		}
	}
	for _, input := range []string{`oops`, `null`, `{}`, `[{"dependencies":{"x":{"version":12}}}]`} {
		if _, _, err := parseList([]byte(input)); err == nil {
			t.Errorf("accepted %s", input)
		}
	}
}

func Test復元引数は固定バージョンだけを許可する(t *testing.T) {
	args, err := installArgs("@scope/tool@1.2.3-beta.1+build")
	if err != nil || !reflect.DeepEqual(args, []string{"add", "-g", "--", "@scope/tool@1.2.3-beta.1+build"}) {
		t.Fatalf("%v %v", args, err)
	}
	for _, spec := range []string{"tool", "tool@latest", "tool@^1.0.0", "--global", "tool@1.0.0;touch /tmp/x", "../tool@1.0.0", "tool@file:foo"} {
		if _, err := installArgs(spec); err == nil {
			t.Errorf("accepted %q", spec)
		}
	}
}
