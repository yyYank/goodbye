package cmd

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/yyYank/goodbye/internal/transfer"
)

func Test移行コマンドのフラグと既定のドライラン(t *testing.T) {
	for _, direction := range []string{"import", "export"} {
		t.Run(direction, func(t *testing.T) {
			var got transfer.Options
			failure := errors.New("action failure")
			cmd := newToolTransferCommand("test", direction, func(opts transfer.Options) error { got = opts; return failure })
			var out bytes.Buffer
			cmd.SetOut(&out)
			if err := cmd.RunE(cmd, nil); !errors.Is(err, failure) {
				t.Fatal(err)
			}
			if !got.DryRun || got.Dir != "." || got.Out != &out {
				t.Fatalf("%+v", got)
			}
			args := []string{"--dir", "/tmp/tools", "--apply", "-v"}
			if direction == "import" {
				args = append(args, "--continue")
			}
			if err := cmd.ParseFlags(args); err != nil {
				t.Fatal(err)
			}
			if err := cmd.RunE(cmd, nil); !errors.Is(err, failure) {
				t.Fatal(err)
			}
			if got.DryRun || !got.Verbose || got.Dir != "/tmp/tools" || got.Continue != (direction == "import") {
				t.Fatalf("%+v", got)
			}
			if cmd.Args(cmd, []string{"unexpected"}) == nil {
				t.Fatal("accepted extra argument")
			}
		})
	}
}

func Test全ツールのCLIから正しいインストール引数を渡す(t *testing.T) {
	for _, tc := range []struct {
		name, file, spec string
		args             []string
	}{
		{"pnpm", "pnpm-global.txt", "@scope/tool@1.2.3", []string{"add", "-g", "--", "@scope/tool@1.2.3"}},
		{"go", "go-tools.txt", "example.com/tool@v1.2.3", []string{"install", "example.com/tool@v1.2.3"}},
		{"cargo", "cargo-tools.txt", "ripgrep@14.1.1", []string{"install", "ripgrep", "--version", "=14.1.1"}},
		{"uv", "uv-tools.txt", "ruff==0.9.1", []string{"tool", "install", "--", "ruff==0.9.1"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			log := filepath.Join(dir, "calls")
			t.Setenv("GOODBYE_TEST_CALLS", log)
			t.Setenv("PATH", dir)
			if err := os.WriteFile(filepath.Join(dir, tc.name), []byte("#!/bin/sh\nprintf '%s\\n' \"$@\" > \"$GOODBYE_TEST_CALLS\"\n"), 0700); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(dir, tc.file), []byte(tc.spec+"\n"), 0600); err != nil {
				t.Fatal(err)
			}
			cmd, _, err := rootCmd.Find([]string{"import", tc.name})
			if err != nil || cmd.Name() != tc.name || cmd.Parent() != importCmd {
				t.Fatalf("command not registered: %v", err)
			}
			var out bytes.Buffer
			cmd.SetOut(&out)
			if err := cmd.ParseFlags([]string{"--dir", dir, "--apply=false"}); err != nil {
				t.Fatal(err)
			}
			if err := cmd.RunE(cmd, nil); err != nil {
				t.Fatal(err)
			}
			if _, err := os.Stat(log); !os.IsNotExist(err) {
				t.Fatal("dry-run executed installer")
			}
			if err := cmd.ParseFlags([]string{"--apply"}); err != nil {
				t.Fatal(err)
			}
			if err := cmd.RunE(cmd, nil); err != nil {
				t.Fatal(err)
			}
			data, err := os.ReadFile(log)
			if err != nil {
				t.Fatal(err)
			}
			want := bytes.Join(func() [][]byte {
				var v [][]byte
				for _, arg := range tc.args {
					v = append(v, []byte(arg))
				}
				return v
			}(), []byte("\n"))
			if !reflect.DeepEqual(data, append(want, '\n')) {
				t.Fatalf("%q want %q", data, want)
			}
			exp, _, err := rootCmd.Find([]string{"export", tc.name})
			if err != nil || exp.Name() != tc.name || exp.Parent() != exportCmd {
				t.Fatalf("export not registered: %v", err)
			}
		})
	}
}
