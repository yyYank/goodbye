package npm

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func Testエクスポートした名前をスコープ付きでインポートする(t *testing.T) {
	dir := t.TempDir()
	fixture := filepath.Join(dir, "npm-list.txt")
	input := "/opt/node/lib\n/opt/node/lib/node_modules/@anthropic-ai/claude-code\n/opt/node/lib/node_modules/@openai/codex\n/opt/node/lib/node_modules/@first/cli\n/opt/node/lib/node_modules/@second/cli\n/opt/node/lib/node_modules/typescript\n"
	if err := os.WriteFile(fixture, []byte(input), 0600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("GOODBYE_NPM_LIST", fixture)
	if err := Export(&NpmExportConfig{GlobalCmd: `cat "$GOODBYE_NPM_LIST"`}, ExportOptions{Dir: dir}); err != nil {
		t.Fatal(err)
	}
	log := filepath.Join(dir, "installed.txt")
	t.Setenv("GOODBYE_NPM_LOG", log)
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
	if err := os.WriteFile(filepath.Join(dir, "record-npm"), []byte("#!/bin/sh\nprintf '%s\\n' \"$@\" >> \"$GOODBYE_NPM_LOG\"\n"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := Import(&NpmImportConfig{GlobalInstallCmd: "record-npm"}, ImportOptions{Dir: dir}); err != nil {
		t.Fatal(err)
	}
	got, err := readLines(log)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"@anthropic-ai/claude-code", "@openai/codex", "@first/cli", "@second/cli", "typescript"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("installed=%v want=%v", got, want)
	}
}

func Test一覧ルートを除外してスコープ名を保持する(t *testing.T) {
	for _, tc := range []struct {
		name        string
		input, want []string
	}{
		{"ルートだけ", []string{"/usr/local/lib", "/usr/local/lib/node_modules", ""}, nil},
		{"スコープ", []string{"/usr/local/lib/node_modules/@scope/cli"}, []string{"@scope/cli"}},
		{"末尾区切り", []string{"/usr/local/lib/", "/usr/local/lib/node_modules/typescript/"}, []string{"typescript"}},
		{"不完全なスコープ", []string{"/usr/local/lib/node_modules/@scope"}, nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := parseGlobalPackages(tc.input); !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("%v want %v", got, tc.want)
			}
		})
	}
}
