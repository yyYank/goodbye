package gotools

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"runtime/debug"
	"testing"

	"github.com/yyYank/goodbye/internal/transfer"
)

func Testビルド情報からコマンドパスとモジュールバージョンを復元する(t *testing.T) {
	info := &debug.BuildInfo{Path: "example.com/tools/cmd/tool", Main: debug.Module{Path: "example.com/tools", Version: "v1.2.3"}}
	spec, err := buildSpec(info)
	if err != nil || spec != "example.com/tools/cmd/tool@v1.2.3" {
		t.Fatalf("%s %v", spec, err)
	}
	for _, bad := range []*debug.BuildInfo{
		{Path: info.Path, Main: debug.Module{Path: info.Main.Path, Version: "(devel)"}},
		{Path: info.Path, Main: debug.Module{Path: info.Main.Path, Version: "v1.2.3", Replace: &debug.Module{Path: "../local"}}},
		{Path: info.Path, Main: info.Main, Deps: []*debug.Module{{Path: "example.com/dep", Replace: &debug.Module{Path: "../dep"}}}},
		{Path: "command-line-arguments", Main: info.Main},
		{Path: "other.com/tool", Main: info.Main},
	} {
		if _, err := buildSpec(bad); err == nil {
			t.Errorf("accepted %+v", bad)
		}
	}
}

func Testローカル変更を含むGoバイナリはエクスポートしない(t *testing.T) {
	for _, info := range []*debug.BuildInfo{
		{Path: "example.com/tool", Main: debug.Module{Path: "example.com/tool", Version: "v1.0.1-0.20260916082842-7a616f37eb51+dirty"}},
		{Path: "example.com/tool", Main: debug.Module{Path: "example.com/tool", Version: "v1.0.0"}, Settings: []debug.BuildSetting{{Key: "vcs.modified", Value: "true"}}},
	} {
		if _, err := buildSpec(info); err == nil {
			t.Errorf("accepted dirty build: %+v", info)
		}
		dir := t.TempDir()
		if err := os.WriteFile(filepath.Join(dir, "tool"), []byte("fixture"), 0700); err != nil {
			t.Fatal(err)
		}
		items, warnings, err := scanDir(dir, func(string) (*debug.BuildInfo, error) { return info, nil })
		if err != nil || len(items) != 0 || len(warnings) != 1 {
			t.Fatalf("dirty export: %v %v %v", items, warnings, err)
		}
	}
}

func Testインストール引数は完全なパスとバージョンを要求する(t *testing.T) {
	for _, spec := range []string{"example.com/cmd/tool@v1.2.3", "example.com/tool/v2@v2.0.0-20260101000000-abcdef123456", "example.com/tool@v2.0.0+incompatible"} {
		args, err := installArgs(spec)
		if err != nil || !reflect.DeepEqual(args, []string{"install", spec}) {
			t.Fatalf("%v %v", args, err)
		}
	}
	for _, spec := range []string{"tool@v1.2.3", "example.com/tool@latest", "./tool@v1.2.3", "example.com/../tool@v1.2.3", "example.com/...@v1.2.3", "--help", "example.com/tool@v1.2.3;echo x"} {
		if _, err := installArgs(spec); err == nil {
			t.Errorf("accepted %q", spec)
		}
	}
}

func Test配置先を選択し読めないバイナリを報告する(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{"tool", "nongofile"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("fixture"), 0700); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.Mkdir(filepath.Join(dir, "subdir"), 0700); err != nil {
		t.Fatal(err)
	}
	var inspected []string
	read := func(path string) (*debug.BuildInfo, error) {
		inspected = append(inspected, filepath.Base(path))
		if filepath.Base(path) == "nongofile" {
			return nil, errors.New("not a Go executable")
		}
		return &debug.BuildInfo{Path: "example.com/tools/cmd/tool", Main: debug.Module{Path: "example.com/tools", Version: "v1.2.3"}}, nil
	}
	items, warnings, err := scanDir(dir, read)
	if err != nil || !reflect.DeepEqual(items, []string{"example.com/tools/cmd/tool@v1.2.3"}) || len(warnings) != 1 || len(inspected) != 2 {
		t.Fatalf("%v %v %v %v", items, warnings, err, inspected)
	}
	items, _, err = scanDir(filepath.Join(dir, "missing"), read)
	if err != nil || len(items) != 0 {
		t.Fatalf("%v %v", items, err)
	}
	for _, tc := range []struct{ input, want string }{
		{`{"GOBIN":"/custom/bin","GOPATH":"/first:/second"}`, "/custom/bin"},
		{`{"GOBIN":"","GOPATH":"/first:/second"}`, "/first/bin"},
	} {
		got, err := binDir([]byte(tc.input))
		if err != nil || got != tc.want {
			t.Fatalf("%s %v", got, err)
		}
	}
	for _, input := range []string{`{}`, `invalid`, `{"GOBIN":"relative"}`, `{"GOPATH":""}`} {
		if _, err := binDir([]byte(input)); err == nil {
			t.Errorf("accepted %s", input)
		}
	}
}

func Test取得から復元まで実行する(t *testing.T) {
	var calls [][]string
	run := func(name string, args ...string) ([]byte, error) {
		calls = append(calls, append([]string{name}, args...))
		return []byte(`{"GOBIN":"/fixture/bin"}`), nil
	}
	scan := func(dir string) ([]string, []string, error) {
		if dir != "/fixture/bin" {
			t.Fatal(dir)
		}
		return []string{"example.com/tool@v1.2.3"}, nil, nil
	}
	m := newManager(run, run, scan)
	var out bytes.Buffer
	opts := transfer.Options{Dir: t.TempDir(), Out: &out}
	if err := m.Export(opts); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(opts.Dir, "go-tools.txt")); err != nil {
		t.Fatal(err)
	}
	if err := m.Import(opts); err != nil {
		t.Fatal(err)
	}
	want := [][]string{{"go", "env", "-json", "GOBIN", "GOPATH"}, {"go", "install", "example.com/tool@v1.2.3"}}
	if !reflect.DeepEqual(calls, want) {
		t.Fatal(calls)
	}
}
