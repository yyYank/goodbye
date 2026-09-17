package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/BurntSushi/toml"
	"github.com/yyYank/goodbye/internal/transfer"
)

func Test五種類のexportだけmise形式を選択できる(t *testing.T) {
	for _, name := range []string{"npm", "pnpm", "go", "cargo", "uv"} {
		cmd, _, err := rootCmd.Find([]string{"export", name})
		if err != nil {
			t.Fatal(err)
		}
		flag := cmd.Flags().Lookup("format")
		if flag == nil || flag.DefValue != "text" {
			t.Errorf("%s: missing format flag or incorrect default", name)
		}
		imp, _, err := rootCmd.Find([]string{"import", name})
		if err != nil {
			t.Fatal(err)
		}
		if imp.Flags().Lookup("format") != nil {
			t.Errorf("%s: export-only flag on import", name)
		}
	}
	var got transfer.Options
	cmd := newToolTransferCommand("test", "export", func(opts transfer.Options) error { got = opts; return nil })
	if err := cmd.ParseFlags([]string{"--format", "mise"}); err != nil {
		t.Fatal(err)
	}
	if err := cmd.RunE(cmd, nil); err != nil {
		t.Fatal(err)
	}
	if got.Format != "mise" || !got.DryRun {
		t.Fatalf("%+v", got)
	}
}

func TestCLIでnpmとpnpmの出力を集約し競合を拒否する(t *testing.T) {
	dir := t.TempDir()
	bin := filepath.Join(dir, "bin")
	if err := os.Mkdir(bin, 0700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", dir)
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Setenv("GOODBYE_NPM_JSON", `{"dependencies":{"@scope/cli":{"version":"1.2.3"}}}`)
	t.Setenv("GOODBYE_PNPM_JSON", `[{"dependencies":{"other":{"version":"2.0.0"}}}]`)
	for name, variable := range map[string]string{"npm": "GOODBYE_NPM_JSON", "pnpm": "GOODBYE_PNPM_JSON"} {
		if err := os.WriteFile(filepath.Join(bin, name), []byte("#!/bin/sh\nprintf '%s' \"$"+variable+"\"\n"), 0700); err != nil {
			t.Fatal(err)
		}
	}
	exportDir := filepath.Join(dir, "export")
	run := func(name string) error {
		cmd, _, err := rootCmd.Find([]string{"export", name})
		if err != nil {
			return err
		}
		oldOut := cmd.OutOrStdout()
		defer cmd.SetOut(oldOut)
		for _, key := range []string{"dir", "apply", "format"} {
			flag := cmd.Flags().Lookup(key)
			if flag == nil {
				t.Fatalf("%s missing %s", name, key)
			}
			old := flag.Value.String()
			defer flag.Value.Set(old)
		}
		var out bytes.Buffer
		cmd.SetOut(&out)
		if err := cmd.ParseFlags([]string{"--dir", exportDir, "--format", "mise", "--apply"}); err != nil {
			return err
		}
		return cmd.RunE(cmd, nil)
	}
	for _, name := range []string{"npm", "pnpm"} {
		if err := run(name); err != nil {
			t.Fatal(err)
		}
	}
	path := filepath.Join(exportDir, ".mise.toml")
	var got struct{ Tools map[string]string }
	if _, err := toml.DecodeFile(path, &got); err != nil {
		t.Fatal(err)
	}
	if len(got.Tools) != 2 || got.Tools["npm:@scope/cli"] != "1.2.3" || got.Tools["npm:other"] != "2.0.0" {
		t.Fatal(got.Tools)
	}
	before, _ := os.ReadFile(path)
	t.Setenv("GOODBYE_PNPM_JSON", `[{"dependencies":{"@scope/cli":{"version":"9.0.0"}}}]`)
	if err := run("pnpm"); err == nil {
		t.Fatal("accepted conflicting version")
	}
	after, _ := os.ReadFile(path)
	if !bytes.Equal(before, after) {
		t.Fatal("conflict modified output")
	}
}
