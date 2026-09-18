package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/yyYank/goodbye/internal/transfer"
	"github.com/yyYank/goodbye/internal/uninstall"
)

func Test削除CLIの個別全件ファイル対話選択(t *testing.T) {
	file := filepath.Join(t.TempDir(), "npm-global.txt")
	os.WriteFile(file, []byte("# exported\nfoo@1.0.0\n"), 0600)
	for _, tc := range []struct {
		args  []string
		input string
		want  []string
	}{
		{[]string{"npm", "foo"}, "", nil},
		{[]string{"npm", "foo", "--apply"}, "", []string{"foo"}},
		{[]string{"npm", "--all", "--apply"}, "", []string{"foo", "bar"}},
		{[]string{"npm", "--file", file, "--apply"}, "", []string{"foo"}},
		{[]string{"npm", "--tui", "--apply"}, "j \ndelete\n", []string{"bar"}},
		{[]string{"npm", "--tui", "--apply"}, "j \nno\n", nil},
		{[]string{"npm", "--tui"}, "ljl\n", nil},
	} {
		var calls []string
		c := newUninstallCommand(func(string) ([]uninstall.Item, error) {
			return []uninstall.Item{{Name: "foo", Version: "1.0.0", Command: []string{"npm", "uninstall", "-g", "--", "foo"}}, {Name: "bar", Command: []string{"npm", "uninstall", "-g", "--", "bar"}}}, nil
		}, func(_ string, args ...string) ([]byte, error) {
			calls = append(calls, args[len(args)-1])
			return nil, nil
		})
		c.SetArgs(tc.args)
		c.SetIn(strings.NewReader(tc.input))
		c.SetOut(&bytes.Buffer{})
		c.SetErr(&bytes.Buffer{})
		if err := c.Execute(); err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(calls, tc.want) {
			t.Fatalf("args=%v calls=%v want=%v", tc.args, calls, tc.want)
		}
	}
}

func Test削除CLIは不正な選択で何も消さない(t *testing.T) {
	for _, tc := range []struct {
		args  []string
		input string
	}{
		{[]string{"npm", "foo", "--all", "--apply"}, ""},
		{[]string{"npm", "--all", "--tui", "--apply"}, ""},
		{[]string{"npm", "--apply"}, ""},
		{[]string{"npm", "foo", "missing", "--apply"}, ""},
		{[]string{"npm", "--tui", "--apply"}, "99\ndelete\n"},
		{[]string{"npm", "--tui", "--apply"}, "1\n"},
	} {
		c := newUninstallCommand(func(string) ([]uninstall.Item, error) {
			return []uninstall.Item{{Name: "foo", Command: []string{"npm", "uninstall", "foo"}}}, nil
		}, func(string, ...string) ([]byte, error) { t.Fatal("削除を実行した"); return nil, nil })
		c.SetArgs(tc.args)
		c.SetIn(strings.NewReader(tc.input))
		c.SetOut(&bytes.Buffer{})
		c.SetErr(&bytes.Buffer{})
		if err := c.Execute(); err == nil {
			t.Fatalf("args=%v", tc.args)
		}
	}
}

func Test削除CLIから実プロセスへ正しい引数を渡す(t *testing.T) {
	dir := t.TempDir()
	log := filepath.Join(dir, "calls")
	t.Setenv("PATH", dir)
	t.Setenv("UNINSTALL_TEST_LOG", log)
	script := `#!/bin/sh
case "$1" in
 root) printf '/tmp/global/node_modules\n' ;;
 list) printf '{"dependencies":{"foo":{"version":"1.2.3"}}}\n' ;;
 uninstall) printf '%s\n' "$@" >> "$UNINSTALL_TEST_LOG" ;;
 *) exit 1 ;;
esac
`
	if err := os.WriteFile(filepath.Join(dir, "npm"), []byte(script), 0700); err != nil {
		t.Fatal(err)
	}
	run := uninstall.Runner(transfer.Output)
	for _, apply := range []bool{false, true} {
		c := newUninstallCommand(func(name string) ([]uninstall.Item, error) { return uninstall.List(name, run) }, run)
		args := []string{"npm", "foo"}
		if apply {
			args = append(args, "--apply")
		}
		c.SetArgs(args)
		c.SetOut(&bytes.Buffer{})
		if err := c.Execute(); err != nil {
			t.Fatal(err)
		}
		data, err := os.ReadFile(log)
		if !apply && !os.IsNotExist(err) {
			t.Fatalf("dry-runで実行: %q %v", data, err)
		}
		if apply && (err != nil || string(data) != "uninstall\n-g\n--\nfoo\n") {
			t.Fatalf("%q %v", data, err)
		}
	}
}
