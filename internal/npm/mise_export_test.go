package npm

import (
	"bytes"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/BurntSushi/toml"
)

func Test詳細JSONからルートを除外しスコープとバージョンを保持する(t *testing.T) {
	input := `{"name":"lib","dependencies":{"@scope/cli":{"name":"@scope/cli","version":"1.2.3"},"tool":{"version":"2.0.0"},"local":{"version":"1.0.0","link":true},"alias":{"name":"actual","version":"1.0.0"},"git":{"version":"1.0.0","resolved":"git+https://example.com/repo"},"missing":{}}}`
	items, warnings, err := parsePinnedPackages([]byte(input))
	if err != nil || !reflect.DeepEqual(items, []string{"@scope/cli@1.2.3", "tool@2.0.0"}) || len(warnings) != 4 {
		t.Fatalf("%v %v %v", items, warnings, err)
	}
	for _, input := range []string{`null`, `[]`, `oops`, `{"dependencies":{"tool":{"version":1}}}`} {
		if _, _, err := parsePinnedPackages([]byte(input)); err == nil {
			t.Errorf("accepted %s", input)
		}
	}
	items, _, err = parsePinnedPackages([]byte(`{}`))
	if err != nil || len(items) != 0 {
		t.Fatalf("%v %v", items, err)
	}
}

func Test通常形式を変えずmise形式だけ詳細一覧を使う(t *testing.T) {
	dir := t.TempDir()
	bin := filepath.Join(dir, "bin")
	if err := os.Mkdir(bin, 0700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Setenv("GOODBYE_NPM_LOG", filepath.Join(dir, "args"))
	script := "#!/bin/sh\nprintf '%s\\n' \"$@\" > \"$GOODBYE_NPM_LOG\"\nprintf '%s' '{\"dependencies\":{\"@scope/cli\":{\"version\":\"1.2.3\"}}}'\n"
	if err := os.WriteFile(filepath.Join(bin, "npm"), []byte(script), 0700); err != nil {
		t.Fatal(err)
	}
	cfg := &NpmExportConfig{GlobalCmd: "printf '/root/node_modules/legacy\\n'"}
	var out bytes.Buffer
	exportDir := filepath.Join(dir, "export")
	if err := Export(cfg, ExportOptions{Dir: exportDir, Format: "mise", DryRun: true, Out: &out}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(exportDir); !os.IsNotExist(err) {
		t.Fatal("dry-run created output")
	}
	if !strings.Contains(out.String(), `"npm:@scope/cli" = "1.2.3"`) {
		t.Fatal(out.String())
	}
	if err := Export(cfg, ExportOptions{Dir: exportDir, Format: "mise", Out: &out}); err != nil {
		t.Fatal(err)
	}
	var got struct{ Tools map[string]string }
	if _, err := toml.DecodeFile(filepath.Join(exportDir, ".mise.toml"), &got); err != nil {
		t.Fatal(err)
	}
	if got.Tools["npm:@scope/cli"] != "1.2.3" {
		t.Fatal(got.Tools)
	}
	args, _ := os.ReadFile(filepath.Join(dir, "args"))
	if string(args) != "list\n-g\n--depth=0\n--json\n--long\n" {
		t.Fatalf("%q", args)
	}
	if _, err := os.Stat(filepath.Join(exportDir, "npm-global.txt")); !os.IsNotExist(err) {
		t.Fatal("mise format wrote text file")
	}
	if err := Export(cfg, ExportOptions{Dir: exportDir, Format: "text"}); err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(filepath.Join(exportDir, "npm-global.txt"))
	if string(data) != "legacy\n" {
		t.Fatalf("%q", data)
	}
}

func Test不正なnpm出力形式は実行せず拒否する(t *testing.T) {
	if err := Export(&NpmExportConfig{GlobalCmd: "true"}, ExportOptions{Format: "invalid", Dir: t.TempDir()}); err == nil {
		t.Fatal("expected format error")
	}
}
