package uninstall

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestCargoの設定先を一覧と削除で固定する(t *testing.T) {
	home := t.TempDir()
	root := filepath.Join(home, "tools")
	t.Setenv("CARGO_HOME", home)
	t.Setenv("CARGO_INSTALL_ROOT", "")
	os.WriteFile(filepath.Join(home, "config.toml"), []byte("[install]\nroot = '"+root+"'\n"), 0600)
	items, err := List("cargo", func(_ string, args ...string) ([]byte, error) {
		if !reflect.DeepEqual(args, []string{"install", "--list", "--color", "never", "--root", root}) {
			t.Errorf("%v", args)
		}
		return []byte("rg v1.2.3:\n    rg\n"), nil
	})
	if err != nil || len(items) != 1 {
		t.Fatalf("%v %v", items, err)
	}
	if !reflect.DeepEqual(items[0].Command, []string{"cargo", "uninstall", "--root", root, "--", "rg"}) {
		t.Fatal(items[0])
	}
	t.Setenv("CARGO_INSTALL_ROOT", "/tmp/mise/installs/cargo-rg/1")
	if _, err := List("cargo", func(string, ...string) ([]byte, error) { t.Fatal("保護対象を実行"); return nil, nil }); err == nil {
		t.Fatal("mise領域を許可")
	}
}
