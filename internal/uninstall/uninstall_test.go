package uninstall

import (
	"bytes"
	"errors"
	"reflect"
	"strings"
	"testing"
)

func Test削除対象を全件検証して選択する(t *testing.T) {
	items := []Item{{Name: "foo", Version: "1.0.0"}, {Name: "@scope/bar", Version: "2.0.0"}}
	for _, tc := range []struct {
		names []string
		all   bool
		want  int
		bad   bool
	}{
		{[]string{"foo"}, false, 1, false}, {[]string{"foo@1.0.0", "foo"}, false, 1, false},
		{nil, true, 2, false}, {nil, false, 0, true}, {[]string{"foo"}, true, 0, true},
		{[]string{"foo", "missing"}, false, 0, true}, {[]string{"foo@2.0.0"}, false, 0, true},
		{[]string{"@scope/bar@2.0.0"}, false, 1, false},
	} {
		got, err := Select(items, tc.names, tc.all)
		if (err != nil) != tc.bad || len(got) != tc.want {
			t.Fatalf("%+v got=%v err=%v", tc, got, err)
		}
	}
}

func Test削除はapply時だけ実行し失敗を返す(t *testing.T) {
	for _, apply := range []bool{false, true} {
		calls := 0
		var out bytes.Buffer
		failure := errors.New("failed")
		items := []Item{{Name: "foo", Version: "1", Command: []string{"npm", "uninstall", "-g", "--", "foo"}}, {Name: "bar", Command: []string{"npm", "uninstall", "-g", "--", "bar"}}}
		err := Execute(items, apply, &out, func(name string, args ...string) ([]byte, error) { calls++; return nil, failure })
		if apply && (calls != 1 || !errors.Is(err, failure)) {
			t.Fatalf("calls=%d err=%v", calls, err)
		}
		if !apply && (calls != 0 || err != nil || !strings.Contains(out.String(), "dry-run")) {
			t.Fatalf("calls=%d err=%v out=%s", calls, err, out.String())
		}
	}
}

func Test一覧取得から各管理元の削除コマンドを構築する(t *testing.T) {
	t.Setenv("CARGO_INSTALL_ROOT", "/tmp/test-cargo")
	for _, tc := range []struct {
		manager, data, name, version string
		command                      []string
	}{
		{"npm", `{"dependencies":{"@scope/tool":{"version":"1.2.3"}}}`, "@scope/tool", "1.2.3", []string{"npm", "uninstall", "-g", "--", "@scope/tool"}},
		{"pnpm", `[{"dependencies":{"foo":{"version":"1.0.0"}}}]`, "foo", "1.0.0", []string{"pnpm", "remove", "-g", "--", "foo"}},
		{"cargo", "ripgrep v14.1.1:\n    rg\n", "ripgrep", "14.1.1", []string{"cargo", "uninstall", "--root", "/tmp/test-cargo", "--", "ripgrep"}},
		{"uv", "ruff v0.9.1\n- ruff\n", "ruff", "0.9.1", []string{"uv", "tool", "uninstall", "--", "ruff"}},
	} {
		t.Run(tc.manager, func(t *testing.T) {
			items, err := List(tc.manager, func(name string, args ...string) ([]byte, error) {
				if len(args) > 0 && args[0] == "root" {
					return []byte("/tmp/global/node_modules\n"), nil
				}
				if len(args) > 1 && args[0] == "tool" && args[1] == "dir" {
					return []byte("/tmp/uv-tools\n"), nil
				}
				return []byte(tc.data), nil
			})
			if err != nil || len(items) != 1 {
				t.Fatalf("items=%v err=%v", items, err)
			}
			if items[0].Name != tc.name || items[0].Version != tc.version || !reflect.DeepEqual(items[0].Command, tc.command) {
				t.Fatalf("%+v", items[0])
			}
		})
	}
}

func Test一覧取得失敗や不正形式では削除しない(t *testing.T) {
	for _, manager := range []string{"npm", "pnpm", "cargo", "uv", "brew"} {
		if _, err := List(manager, func(string, ...string) ([]byte, error) { return nil, errors.New("failure") }); err == nil {
			t.Fatal(manager)
		}
		if _, err := List(manager, func(string, ...string) ([]byte, error) { return []byte("not a list"), nil }); err == nil {
			t.Fatal(manager)
		}
	}
}

func Test名前自体のアットマークと曖昧な種類を扱う(t *testing.T) {
	items := []Item{{Name: "python@3.12", Version: "3.12.9", Kind: "formula"}, {Name: "same", Kind: "formula"}, {Name: "same", Kind: "cask"}}
	if got, err := Select(items, []string{"python@3.12"}, false); err != nil || len(got) != 1 {
		t.Fatalf("%v %v", got, err)
	}
	if _, err := Select(items, []string{"same"}, false); err == nil {
		t.Fatal("曖昧な種類")
	}
	if got, err := Select(items, []string{"cask:same"}, false); err != nil || len(got) != 1 {
		t.Fatalf("%v %v", got, err)
	}
	if _, err := Select([]Item{{Name: "foo"}}, []string{"foo@"}, false); err == nil {
		t.Fatal("空バージョン")
	}
}

func TestBrewの種類別削除とバージョンを取得する(t *testing.T) {
	items, err := List("brew", func(_ string, args ...string) ([]byte, error) {
		if args[0] == "--prefix" {
			return []byte("/opt/homebrew\n"), nil
		}
		return []byte(`{"formulae":[{"name":"python@3.12","full_name":"python@3.12","installed":[{"version":"3.12.9"}]}],"casks":[{"token":"app","installed":"1.2.3"}]}`), nil
	})
	if err != nil || len(items) != 2 {
		t.Fatalf("%v %v", items, err)
	}
	for _, item := range items {
		if item.Version == "" || !strings.HasPrefix(item.Path, "/opt/homebrew/") || !strings.Contains(strings.Join(item.Command, " "), "--"+item.Kind) {
			t.Fatalf("%+v", item)
		}
	}
}

func TestBrewは選択した依存と利用元をまとめて削除する(t *testing.T) {
	items := []Item{{Name: "lib", Command: []string{"brew", "uninstall", "--formula", "--", "lib"}}, {Name: "app", Command: []string{"brew", "uninstall", "--formula", "--", "app"}}}
	calls := 0
	err := Execute(items, true, &bytes.Buffer{}, func(name string, args ...string) ([]byte, error) {
		calls++
		if !reflect.DeepEqual(args, []string{"uninstall", "--formula", "--", "lib", "app"}) {
			t.Errorf("%v", args)
		}
		return nil, nil
	})
	if err != nil || calls != 1 {
		t.Fatalf("%v %d", err, calls)
	}
}
