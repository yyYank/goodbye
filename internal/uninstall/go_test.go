package uninstall

import (
	"bytes"
	"os"
	"path/filepath"
	"runtime/debug"
	"testing"
)

func TestGoはビルド情報で対象を特定しリンクを除外する(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "tool")
	if err := os.WriteFile(path, []byte("binary"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(path, filepath.Join(dir, "alias")); err != nil {
		t.Fatal(err)
	}
	read := func(string) (*debug.BuildInfo, error) {
		return &debug.BuildInfo{Path: "example.com/tool/cmd/tool", Main: debug.Module{Path: "example.com/tool", Version: "v1.0.0"}}, nil
	}
	items, err := scanGo(dir, read)
	if err != nil || len(items) != 1 || items[0].Name != "example.com/tool/cmd/tool" {
		t.Fatalf("%v %v", items, err)
	}
	if err = Execute(items, false, &bytes.Buffer{}, nil); err != nil {
		t.Fatal(err)
	}
	if _, err = os.Stat(path); err != nil {
		t.Fatal(err)
	}
	if err = Execute(items, true, &bytes.Buffer{}, nil); err != nil {
		t.Fatal(err)
	}
	if _, err = os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("削除されていない: %v", err)
	}
}

func TestGoの対象が置き換わった場合は削除しない(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "tool")
	os.WriteFile(path, []byte("binary"), 0700)
	items, err := scanGo(dir, func(string) (*debug.BuildInfo, error) {
		return &debug.BuildInfo{Path: "example.com/tool", Main: debug.Module{Path: "example.com/tool", Version: "v1.0.0"}}, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	os.Rename(path, path+".old")
	os.WriteFile(path, []byte("replacement"), 0700)
	if err = Execute(items, true, &bytes.Buffer{}, nil); err == nil {
		t.Fatal("置換を検知しなかった")
	}
	if _, err = os.Stat(path); err != nil {
		t.Fatal(err)
	}
}

func TestMiseの個別管理領域は削除しない(t *testing.T) {
	for _, path := range []string{"/home/user/.local/share/mise/installs/go-example-com-tool/1/bin", "/home/user/.local/share/mise/installs/npm-foo/1/lib/node_modules"} {
		if err := protectMise(path); err == nil {
			t.Fatal(path)
		}
	}
	t.Setenv("MISE_DATA_DIR", "/tmp/custom-mise")
	if err := protectMise("/tmp/custom-mise/installs/go-foo/1/bin"); err == nil {
		t.Fatal("カスタムmise領域を許可した")
	}
}
