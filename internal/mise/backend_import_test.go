package mise

import (
	"bytes"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/yyYank/goodbye/internal/miseconfig"
)

func Test引用符付きバックエンド名を正しい名前で読む(t *testing.T) {
	input := "[tools]\n\"npm:@scope/cli\" = \"1.2.3\" # pinned\n'go:example.com/tool' = 'v1.0.0'\n\"cargo:ripgrep\" = [\"14.1.1\"]\n"
	got, err := ParseTOML(input)
	want := []InstalledTool{{Name: "npm:@scope/cli", Version: "1.2.3"}, {Name: "go:example.com/tool", Version: "v1.0.0"}, {Name: "cargo:ripgrep", Version: "14.1.1"}}
	if err != nil || !reflect.DeepEqual(got, want) {
		t.Fatalf("%+v %v", got, err)
	}
	for _, input := range []string{"[tools", "[tools]\nnode=123", "[tools]\nnode={version='22',postinstall='echo setup'}", "[tools]\nnode=['22',1]"} {
		if _, err := ParseTOML(input); err == nil {
			t.Errorf("accepted invalid or unsupported config: %s", input)
		}
	}
}

func Test生成した設定をグローバルへ固定バージョンで復元する(t *testing.T) {
	dir := t.TempDir()
	log := filepath.Join(dir, "calls")
	var out bytes.Buffer
	if err := miseconfig.Export(dir, map[string]string{"npm:@scope/cli": "1.2.3", "go:example.com/tool": "v1.0.0"}, false, &out); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Setenv("GOODBYE_MISE_LOG", log)
	script := "#!/bin/sh\nprintf '%s\\n' \"$@\" >> \"$GOODBYE_MISE_LOG\"\nif [ \"$1\" = install ] && [ \"$2\" = -g ]; then exit 2; fi\n"
	if err := os.WriteFile(filepath.Join(dir, "mise"), []byte(script), 0700); err != nil {
		t.Fatal(err)
	}
	if err := Import(ImportOptions{Dir: dir, Global: true, DryRun: true}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(log); !os.IsNotExist(err) {
		t.Fatal("dry-run executed mise")
	}
	if err := Import(ImportOptions{Dir: dir, Global: true}); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(log)
	if err != nil {
		t.Fatal(err)
	}
	want := "use\n-g\n--pin\ngo:example.com/tool@v1.0.0\nuse\n-g\n--pin\nnpm:@scope/cli@1.2.3\n"
	if string(got) != want {
		t.Fatalf("got %q want %q", got, want)
	}
}
